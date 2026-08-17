# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-08-17

### Added

- Language AST transform library: built-in Go and Python drivers.
- WASM plugin host (wazero, capability ABI, no guest FS/network/env).
- `--ast-wasm` language registration maps conventional extensions (e.g. `rust` → `.rs`).
- Example crate: `examples/wasm-rust`.
- Contributor docs: CONTRIBUTING, SECURITY, ROADMAP, docs/ARCHITECTURE.

### Changed

- deps: wazero v1.9.0 → v1.12.0.

### Security

- WASM paths must be local files; URLs are rejected.

## [0.1.0] - 2026-08-14

Not shipped as a standalone module (helpers lived in `engine/internal/astgo`).
