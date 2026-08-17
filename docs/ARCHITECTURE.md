# Architecture — blotless/ast

This repository is the **language transform** module. [engine](https://github.com/blotless/engine) and [cli](https://github.com/blotless/cli) depend on it. They are **separate GitHub repos**, not a monorepo.

## Place in the ecosystem

```
blotless/cli  ──►  blotless/engine  ──►  blotless/ast
     │                                         │
     └── --ast-wasm rust=./x.wasm ─────────────┘
```

## Public API

| Symbol | Role |
|--------|------|
| `Driver` | In-process: `Lang()` + `Transform(src, opts)` |
| `Register` | Built-in Go/Python (`builtin.go`) |
| `RegisterWASM` | Local `.wasm` + default extension map (e.g. `rust` → `.rs`) |
| `MapExt` | Extra extension (CLI `--ast-ext`) |
| `Transform` | Dispatch: in-process first, else WASM |

## Clean path (engine)

1. Layer B finding `statwm.ast_transform` if `LangFromPath` has a driver
2. After Layer A strip, `ast.Transform(lang, bytes, DefaultOpts())`
3. On error, original bytes kept

## WASM host

`wasm_host.go` instantiates wazero **without WASI**. Host module name: `blotless`. Guest export: `transform`.

See [ABI.md](../ABI.md) and the full plugin: [examples/wasm-rust](../examples/wasm-rust/).

## Non-goals

Raw OS exec, URL plugin fetch, claiming certified human authorship.

## Related

- [README.md](../README.md) — install + WASM tutorial
- [CONTRIBUTING.md](../CONTRIBUTING.md)
- [ROADMAP.md](../ROADMAP.md)
