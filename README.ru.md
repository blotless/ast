<h1 align="center">blotless/ast</h1>

<p align="center">
  <strong>AST transform для Layer B</strong><br>
  Go + Python встроенные · WASM-плагины (capability ABI) · Zero CGO
</p>

<p align="center">
  <b>Язык:</b> <a href="README.md">English</a> | Русский
</p>

---

Библиотека языковых трансформов экосистемы [blotless](https://github.com/blotless). Не проверяет SynthID по ключу и не сертифицирует «человеческий» текст.

```bash
go get github.com/blotless/ast
```

```go
out, rep, err := ast.Transform(ast.LangGo, src, ast.DefaultOpts())
```

## WASM-модуль (любой язык → `.wasm`)

Полный пример: [`examples/wasm-rust/`](examples/wasm-rust/). Контракт: [ABI.md](ABI.md).

```bash
rustup target add wasm32-unknown-unknown
cargo build --release --target wasm32-unknown-unknown
# → blotless_rust_transform.wasm

CGO_ENABLED=0 go install github.com/blotless/cli/cmd/blotless@latest
blotless clean ./src --write --layer-b \
  --ast-wasm rust=./blotless_rust_transform.wasm
```

`--ast-wasm rust=…` регистрирует плагин и **сам** мапит `.rs`. Другое расширение:

```bash
blotless clean . --write --layer-b \
  --ast-wasm zig=./zig_transform.wasm \
  --ast-ext .zig=zig
```

Гость видит только host-импорты `blotless` (нет FS/сети/env). URL плагинов запрещены.

Полное описание API, чеклист и безопасность — в [README.md](README.md).
Документы репозитория: [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), [ROADMAP.md](ROADMAP.md), [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
