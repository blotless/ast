<p align="center">
  <strong>Language AST transform for blotless Layer B</strong><br>
  Go + Python built-in · WASM plugins (wazero capability ABI) · Zero CGO
</p>

<h1 align="center">blotless/ast</h1>

<p align="center">
  Semantic-preserving transforms that disrupt token-sampling watermarks<br>
  (Kirchenbauer / SynthID-Text class). Not a vendor-key verifier.
</p>

<p align="center">
  <b>Language:</b> English | <a href="README.ru.md">Русский</a>
</p>

<p align="center">
  <a href="https://github.com/blotless/ast/actions/workflows/ci.yml"><img src="https://github.com/blotless/ast/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/blotless/ast"><img src="https://pkg.go.dev/badge/github.com/blotless/ast.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/blotless/ast"><img src="https://goreportcard.com/badge/github.com/blotless/ast" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License"></a>
  <a href="https://github.com/blotless/ast/releases"><img src="https://img.shields.io/github/v/release/blotless/ast" alt="Latest Release"></a>
  <a href="https://github.com/blotless/ast"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go" alt="Go Version"></a>
</p>

---

## Overview

**ast** is the language-transform library of the [blotless](https://github.com/blotless) ecosystem. [blotless/engine](https://github.com/blotless/engine) calls `ast.Transform` during `clean --layer-b`. [blotless/cli](https://github.com/blotless/cli) loads extra languages with `--ast-wasm`.

It does **not** certify “human-written” text and does **not** reverse SynthID without a vendor key.

### Key Features

| Category | Capabilities |
|----------|--------------|
| **Built-in Go** | Rename unexported locals/params; reorder private struct fields; sort imports; `gofmt` |
| **Built-in Python** | Rename private locals; sort imports; conservative class-body assign reorder |
| **WASM plugins** | Any language compiled to `wasm32-unknown-unknown`; [capability ABI](ABI.md) only |
| **Sandbox** | [wazero](https://github.com/tetratelabs/wazero), zero CGO; no guest FS / network / env |
| **Registry** | `Register` (in-process) · `RegisterWASM` (local `.wasm`) · `MapExt` |
| **Protect** | Go string literals / `go:generate` stay out of ident rewrites in the Go driver |

---

## Installation

```bash
go get github.com/blotless/ast
```

**Requirements:**

- Go 1.26+
- `CGO_ENABLED=0`

```bash
CGO_ENABLED=0 go test ./...
```

---

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/blotless/ast"
)

func main() {
	src := []byte("package p\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n")
	out, rep, err := ast.Transform(ast.LangGo, src, ast.DefaultOpts())
	if err != nil {
		panic(err)
	}
	fmt.Printf("changed=%v renamed=%d\n%s\n", rep.Changed, rep.Renamed, out)
}
```

In-process drivers implement `ast.Driver` (`Lang` + `Transform`) and are registered with `ast.Register`. Built-in Go and Python register in `init`.

---

## Built-in languages

| Lang | Extensions | Driver |
|------|------------|--------|
| `go` | `.go` | [`golang/`](golang/) |
| `python` | `.py` | [`python/`](python/) |

Defaults (`ast.DefaultOpts`): rename locals, reorder fields, reorder imports, format, `seed=1`.

---

## Add a language via WASM (no Go required)

Write a module in **any** language that compiles to WebAssembly, implement `transform`, import **only** `blotless_*` host functions. Full copy-paste crate: [`examples/wasm-rust/`](examples/wasm-rust/).

### Module shape

```
my-transform/
├── Cargo.toml          # crate-type = ["cdylib"]
└── src/lib.rs          # export transform(); import blotless::*
```

Guest **exports:**

| Export | Type | Meaning |
|--------|------|---------|
| `memory` | memory | Linear memory |
| `transform` | `() -> i32` | `0` = ok; non-zero = fail |

Guest **imports** (module name `blotless`) — full table in [ABI.md](ABI.md):

`abi_version`, `get_opts`, `src_len`, `src_read`, `src_write`, `report`, `log`, `fail`.

Minimal Rust skeleton:

```rust
#[link(wasm_import_module = "blotless")]
extern "C" {
    fn src_len() -> u32;
    fn src_read(dest: u32, off: u32, n: u32) -> u32;
    fn src_write(ptr: u32, n: u32) -> u32;
    fn report(ptr: u32, n: u32) -> u32;
}

#[no_mangle]
pub extern "C" fn transform() -> i32 {
    // src_read → transform → src_write → report
    0
}
```

### Build

```bash
rustup target add wasm32-unknown-unknown
cargo build --release --target wasm32-unknown-unknown
# → target/wasm32-unknown-unknown/release/blotless_rust_transform.wasm
```

Do **not** use WASI (`wasm32-wasip1`). The host does not provide FS or sockets.

### Install / connect

`--ast-wasm rust=path.wasm` loads the plugin and maps **`.rs` automatically**.

```bash
CGO_ENABLED=0 go install github.com/blotless/cli/cmd/blotless@latest

blotless clean ./src --write --layer-b \
  --ast-wasm rust=./blotless_rust_transform.wasm
```

Custom language id (example: Zig):

```bash
blotless clean . --write --layer-b \
  --ast-wasm zig=/opt/blotless/zig_transform.wasm \
  --ast-ext .zig=zig
```

| Flag | Example | Meaning |
|------|---------|---------|
| `--ast-wasm` | `rust=./rust_transform.wasm` | Local `.wasm` for language id |
| `--ast-ext` | `.zig=zig` | Extra extension map (optional; `.rs` is automatic for `rust`) |

**Engine:**

```go
eng, err := engine.New(engine.Config{
    Paths:   []string{"."},
    LayerB:  true,
    AstWASM: map[string]string{"rust": "./blotless_rust_transform.wasm"},
})
```

**Library:**

```go
ast.RegisterWASM("rust", "./blotless_rust_transform.wasm")
out, rep, err := ast.Transform("rust", src, ast.DefaultOpts())
```

Rules:

- Path must be a **local file** (URLs rejected)
- Built-in Go/Python win for `.go` / `.py`
- On `fail` / parse error, blotless keeps original bytes for that file
- Timeout ~5s, bounded guest memory

### Checklist

- [ ] Parse failure → `fail` + non-zero; host keeps original
- [ ] Rename only non-public locals
- [ ] Stable renames when `seed` is set
- [ ] Fill `report` (`lang`, `renamed`, `reordered`, `changed`)
- [ ] No files / sockets / env (imports are not provided)

### Add an in-tree Go driver

1. Package `ast/<lang>/` with `Transform(src []byte, opts Opts) ([]byte, Result, error)` — do **not** import root package `ast` (import cycle).
2. Thin adapter in `builtin.go`.
3. Tests + fixtures.

---

## Security

- WASM paths: local files only
- No raw OS-exec plugins
- Guest cannot open files, dial network, or read env
- See [SECURITY.md](SECURITY.md)

---

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). Host runtime: `wasm_host.go` (wazero).

```
cli --ast-wasm rust=./x.wasm
        │
        ▼
engine.Config.AstWASM  →  ast.RegisterWASM
        │
        ▼
clean --layer-b  →  ast.Transform("rust", src, opts)
        │
        ▼
wazero guest: export transform() · import blotless.*
```

---

## Ecosystem

| Project | Role |
|---------|------|
| [blotless/ast](https://github.com/blotless/ast) | **This repo** |
| [blotless/engine](https://github.com/blotless/engine) | Calls `ast.Transform` on clean |
| [blotless/cli](https://github.com/blotless/cli) | `--layer-b`, `--ast-wasm`, `--ast-ext` |
| [blotless/skills](https://github.com/blotless/skills) | Agent skill pack |

---

## Development

```bash
git clone https://github.com/blotless/ast
cd ast
CGO_ENABLED=0 go test ./...
```

Contributor docs: [CONTRIBUTING.md](CONTRIBUTING.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) · [ROADMAP.md](ROADMAP.md)

---

## Disclaimer

Transforms are **best-effort** token-sequence disruption. They cannot certify that a vendor detector will fail.

---

## License

MIT License — see [LICENSE](LICENSE).

---

<p align="center">
  <strong>blotless</strong> — inspect first, then clean
</p>
