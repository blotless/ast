# Contributing to blotless/ast

Thank you for helping extend language transforms for blotless.

---

## Requirements

- **Go 1.26+**
- **`CGO_ENABLED=0`**
- **`GOTOOLCHAIN=local`** in CI
- Optional: Rust + `wasm32-unknown-unknown` for [`examples/wasm-rust`](examples/wasm-rust/)

```bash
git clone https://github.com/blotless/ast
cd ast
unset GOROOT
export GOTOOLCHAIN=local
export CGO_ENABLED=0
go test ./...
```

---

## Workflow

```bash
git checkout -b feat/your-feature
gofmt -w .
go test ./...
git commit -m "feat(golang): rename params in nested functions"
git push origin feat/your-feature
```

Open a PR: https://github.com/blotless/ast/compare

### PR checklist

- [ ] `go test ./...` passes
- [ ] `gofmt` clean
- [ ] ABI changes documented in [ABI.md](ABI.md) and [CHANGELOG](https://github.com/blotless/cli/blob/main/CHANGELOG.md) / this repo README
- [ ] WASM examples still match the host imports
- [ ] No `Co-authored-by:` trailers

### PR title

```
feat(python): sort from-import aliases
fix(wasm): reject URL plugin paths
docs: expand WASM install steps
```

---

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`

| Type | Description |
|------|-------------|
| `feat` / `fix` / `docs` / `test` / `refactor` / `ci` / `chore` | standard |

| Scope | Description |
|-------|-------------|
| `golang` | Go driver |
| `python` | Python driver |
| `wasm` | wazero host / ABI |
| `examples` | plugin examples |

**Author:** `lkmavi <zikmanv@icloud.com>` unless otherwise agreed. Never add `Co-authored-by:`.

---

## Code guidelines

- Zero CGO; wazero only for WASM
- Language packages (`golang/`, `python/`) must not import root `github.com/blotless/ast` (cycle)
- Do not add raw OS-exec plugin loaders
- Guest must not gain FS / network / env
- Prefer small PRs: one driver or one ABI change

---

## Project structure

```
ast/
├── transform.go      # Register, RegisterWASM, Transform, MapExt
├── wasm_host.go      # wazero capability host
├── golang/           # built-in Go
├── python/           # built-in Python
├── examples/wasm-rust/
├── ABI.md
└── docs/ARCHITECTURE.md
```

---

## Questions

- Issues: https://github.com/blotless/ast/issues
- ABI: [ABI.md](ABI.md)
- Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

By contributing you agree the MIT license ([LICENSE](LICENSE)).
