# Example: Rust WASM transform plugin

This crate is a **complete, copy-pasteable** blotless Layer B plugin. It compiles to `wasm32-unknown-unknown` and talks only to the [capability ABI](../../ABI.md) — no WASI filesystem, no network, no env.

What it does: alphabetically sort a leading block of `use ` lines. It does **not** rename public items.

## Layout

```
examples/wasm-rust/
├── Cargo.toml
├── src/lib.rs      # host imports + transform() export
└── README.md       # this file
```

## 1. Build

```bash
rustup target add wasm32-unknown-unknown
cd examples/wasm-rust
cargo test
cargo build --release --target wasm32-unknown-unknown
```

Artifact:

```
target/wasm32-unknown-unknown/release/blotless_rust_transform.wasm
```

Do **not** enable WASI (`wasm32-wasip1`) for blotless plugins. The host does not provide filesystem or sockets; extra WASI imports will fail to instantiate.

## 2. Install (connect to blotless)

`--ast-wasm rust=…` registers the plugin **and** maps `.rs` automatically.

```bash
# install CLI once
CGO_ENABLED=0 go install github.com/blotless/cli/cmd/blotless@latest

# dry-run (no writes)
blotless clean ./src --layer-b \
  --ast-wasm rust=./target/wasm32-unknown-unknown/release/blotless_rust_transform.wasm

# apply
blotless clean ./src --write --layer-b \
  --ast-wasm rust=./target/wasm32-unknown-unknown/release/blotless_rust_transform.wasm
```

### Custom language id / extra extensions

```bash
# language id "zig" is not built-in — map .zig yourself
blotless clean . --write --layer-b \
  --ast-wasm zig=/opt/blotless/zig_transform.wasm \
  --ast-ext .zig=zig
```

Repeatable flags:

| Flag | Example | Meaning |
|------|---------|---------|
| `--ast-wasm` | `rust=./rust_transform.wasm` | Load local `.wasm` for language id `rust` |
| `--ast-ext` | `.zig=zig` | Map extra file extension (`.rs` is automatic for `rust`) |

Rules:

- Path must be a **local file**. URLs are rejected.
- Built-in Go/Python drivers win over WASM for `.go` / `.py`.
- On parse/`fail`, blotless keeps the original bytes for that file.

### From Go (engine)

```go
eng, err := engine.New(engine.Config{
    Paths:   []string{"."},
    LayerB:  true,
    AstWASM: map[string]string{"rust": "./blotless_rust_transform.wasm"},
    // AstExt: map[string]string{".zig": "zig"}, // only if not auto-mapped
})
```

### From Go (ast library)

```go
ast.RegisterWASM("rust", "./blotless_rust_transform.wasm")
out, rep, err := ast.Transform("rust", src, ast.DefaultOpts())
```

## 3. Contract the guest must implement

**Export**

- `memory`
- `transform() -> i32` — `0` ok, non-zero fail

**Import module `blotless` only** — see [ABI.md](../../ABI.md).

Checklist:

- [ ] Parse/transform failure → `fail` + non-zero; host keeps original
- [ ] Do not rename public APIs
- [ ] Stable behavior when `seed` is set
- [ ] Fill `report` JSON (`lang`, `renamed`, `reordered`, `changed`)

## 4. Security

The host (wazero) does **not** give the guest:

- filesystem
- network / DNS
- process env
- subprocesses

Only `src_read` / `src_write` / `get_opts` / `report` / `log` / `fail`.
