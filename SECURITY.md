# Security Policy

## Supported versions

Security fixes land on the **latest release** of `github.com/blotless/ast` and on `main`.

| Version | Supported |
|---------|-----------|
| latest tag | Yes |
| Older tags | Best-effort |

## Reporting a vulnerability

**Do not** open a public GitHub issue.

1. Email: **zikmanv@icloud.com**
2. GitHub Security Advisory (when enabled): https://github.com/blotless/ast/security/advisories/new

Include module version or SHA, impact, and a minimal reproduction (especially WASM / host ABI).

**Initial response:** within 72 hours.

## Security considerations

1. **Zero CGO** — official builds use `CGO_ENABLED=0`.
2. **WASM** — guests get a capability ABI only (no filesystem, network, env). Load `.wasm` from **local explicit paths**. URLs are rejected.
3. **Do not** introduce raw OS-exec plugin loaders.
4. Transforms cannot certify that a vendor watermark detector will fail.

See [ABI.md](ABI.md) and [examples/wasm-rust/README.md](examples/wasm-rust/README.md).
