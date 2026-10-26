// 签名 WASM 的校验、进程内接入和暂存快照。验签公钥在 Open 里从环境变量 YOYO_WASM_PUBKEY 解析进 a.wasmPub，本文件不读环境变量。LoadWASM 只把快照写到 refs/staging，不移动 refs/active。

package app

import (
	"crypto/ed25519"
	"encoding/hex"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
)

// ErrUnsignedWASM 表示模块没有可用签名或公钥。ID 为空时错误文本是 unsigned WASM cannot become refs/active。
type ErrUnsignedWASM struct {
	ID string
}

// Error 在 ID 为空时返回 unsigned WASM cannot become refs/active，否则在后面附上 ID。
func (e ErrUnsignedWASM) Error() string {
	if e.ID == "" {
		return "unsigned WASM cannot become refs/active"
	}
	return "unsigned WASM cannot become refs/active: " + e.ID
}

// SetWASMPublicKey 把调用方给出的 ed25519 公钥放进 a.wasmPub。
func (a *App) SetWASMPublicKey(pub ed25519.PublicKey) {
	a.wasmPub = pub
}

// verifySnapshotWASM 校验快照里每个 WASM 插件的签名。公钥长度不是 ed25519.PublicKeySize 时返回 ErrUnsignedWASM，ID 为 missing YOYO_WASM_PUBKEY。没有插件时返回 nil。
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

// syncWASM 把快照里尚未加载的 WASM 交给 evolve.AdmitWASM。解码或读模块失败则跳过该条。这里不改 refs。
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

// LoadWASM 先过 capability 的 load_wasm。AutoAllow 为假时 ForceAsk。公钥长度不是 ed25519.PublicKeySize 时返回 ID 为 missing YOYO_WASM_PUBKEY 的 ErrUnsignedWASM。验签通过后写入 CAS，进程内 Admit，TimeoutMS 为 2000，再把带该插件的快照 Set 到 refs/staging，不移动 refs/active。
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

// StageSignedWASM 把签名字节编码成十六进制后调用 LoadWASM。
func (a *App) StageSignedWASM(id string, bin []byte, export string, sig []byte) error {
	return a.LoadWASM(id, bin, export, hex.EncodeToString(sig))
}
