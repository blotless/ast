# blotless/ast roadmap

> Semantic-preserving transforms for Layer B. WASM for every other language.

## Principles

1. Zero CGO
2. Capability-sandbox plugins (no raw exec)
3. Built-in Go/Python stay conservative (no public API renames)
4. Honest limits — not SynthID verification

## Current State

Shipped **v0.2.0** (2026-08-17).

| Area | Status |
|------|--------|
| Go transform | Initial |
| Python transform | Initial (line-based, not CPython AST) |
| wazero host + ABI v1 | Initial |
| Rust example crate | Initial |
| Auto-map `.rs` on `--ast-wasm rust=` | Initial |

## Near term

- [x] Publish module tags (`v0.2.0+`) on GitHub
- [ ] CI contract tests against committed `testdata/wasm/*.wasm`
- [ ] AssemblyScript / TinyGo example plugins
- [ ] Stronger Python parser (still pure Go)
- [ ] Coverage floor in CI

## Medium term

- [ ] Optional WASI **clock only** if guests need it (still no FS/net)
- [ ] Plugin report schema versioning (`api_version`)
- [ ] Host instance pool for hot trees

## Out of scope

- Pixel/audio SynthID
- Download-from-URL plugin marketplace
- Full WASI filesystem
