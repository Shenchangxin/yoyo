package app

import (
	"crypto/ed25519"
	"encoding/hex"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
)

type ErrUnsignedWASM struct {
	ID string
}

func (e ErrUnsignedWASM) Error() string {
	if e.ID == "" {
		return "unsigned WASM cannot become refs/active"
	}
	return "unsigned WASM cannot become refs/active: " + e.ID
}

func (a *App) SetWASMPublicKey(pub ed25519.PublicKey) {
	a.wasmPub = pub
}

func (a *App) verifySnapshotWASM(snap artifact.HarnessSnapshot) error {
	if len(snap.WASMPlugins) == 0 {
		return nil
	}
	if len(a.wasmPub) != ed25519.PublicKeySize {
		return ErrUnsignedWASM{ID: "missing YOYO_WASM_PUBKEY"}
	}
	for _, h := range snap.WASMPlugins {
		p, _, err := artifact.Decode[artifact.WASMPlugin](a.CAS, h)
		if err != nil {
			return err
		}
		if p.SigHex == "" {
			return ErrUnsignedWASM{ID: p.ID}
		}
		sig, err := wasm.ParseSig(p.SigHex)
		if err != nil {
			return err
		}
		bin, err := a.CAS.GetRaw(p.ModuleHash)
		if err != nil {
			return err
		}
		if err := wasm.Verify(a.wasmPub, bin, sig); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) syncWASM(snap artifact.HarnessSnapshot) {
	for _, h := range snap.WASMPlugins {
		p, _, err := artifact.Decode[artifact.WASMPlugin](a.CAS, h)
		if err != nil {
			continue
		}
		bin, err := a.CAS.GetRaw(p.ModuleHash)
		if err != nil {
			continue
		}
		if a.WASM.Get(p.ID) != nil {
			continue
		}
		_, _ = evolve.AdmitWASM(a.Kernel, a.WASM, p.ID, bin, p.Export)
	}
}

// LoadWASM verifies the signature, stores the module in CAS, admits it in-process,
// and writes a staging snapshot. It never moves refs/active.
func (a *App) LoadWASM(id string, bin []byte, export, sigHex string) error {
	req := capability.Request{Level: capability.HighRisk, Action: "load_wasm", SessionID: "ui"}
	if !a.Config.AutoAllow {
		req.ForceAsk = true
	}
	if err := a.Caps.Check(req); err != nil {
		return err
	}
	if len(a.wasmPub) != ed25519.PublicKeySize {
		return ErrUnsignedWASM{ID: "missing YOYO_WASM_PUBKEY"}
	}
	sig, err := wasm.ParseSig(sigHex)
	if err != nil {
		return err
	}
	if err := wasm.Verify(a.wasmPub, bin, sig); err != nil {
		return err
	}
	modHash, err := a.CAS.PutRaw(bin)
	if err != nil {
		return err
	}
	plugin := artifact.WASMPlugin{
		ID: id, Name: id, Export: export, ModuleHash: modHash, SigHex: sigHex, TimeoutMS: 2000,
	}
	ph, err := a.CAS.Put(artifact.KindWASMPlugin, id, plugin)
	if err != nil {
		return err
	}
	if _, err := evolve.AdmitWASM(a.Kernel, a.WASM, id, bin, export); err != nil {
		return err
	}
	active := a.ActiveHash()
	base, err := a.LoadSnapshot(active)
	if err != nil {
		return err
	}
	next := artifact.CloneSnapshot(base)
	next.Parent = active
	next.WASMPlugins = append(next.WASMPlugins, ph)
	next.Note = "signed-wasm-stage"
	hash, err := a.CAS.PutSnapshot(next)
	if err != nil {
		return err
	}
	return a.Refs.Set(artifact.RefStaging, hash)
}

func (a *App) StageSignedWASM(id string, bin []byte, export string, sig []byte) error {
	return a.LoadWASM(id, bin, export, hex.EncodeToString(sig))
}
