# WASM plugins

L2 evolution loads WebAssembly modules through wazero.

- Default: no filesystem, no network host functions.
- Size cap 8MiB, per-call timeout.
- Mounted as a kernel fiber so `Dispose` unloads the module.
- Guest languages: TinyGo, Rust, AssemblyScript compiling to `wasm32-wasi` or a reactor module with a numeric export (see tests using `add`).
- Modules are ed25519-signed over the raw bytes. Checkout verifies `YOYO_WASM_PUBKEY`. Unsigned snapshots cannot become `refs/active`. Evolve never compiles WASM; `LoadWASM` writes `refs/staging` only.
