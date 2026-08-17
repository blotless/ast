# blotless AST ABI (api_version = 1)

Guest modules import host functions from module **`blotless`**. No WASI filesystem or network is provided.

## Guest exports

| Export | Type | Meaning |
|--------|------|---------|
| `memory` | memory | Linear memory |
| `transform` | `() -> i32` | `0` success; non-zero failure |

## Host imports (`blotless`)

| Name | Signature | Capability |
|------|-----------|------------|
| `abi_version` | `() -> i32` | Returns `1` |
| `get_opts` | `(ptr, maxlen: i32) -> i32` | Writes JSON opts into guest memory; returns bytes written |
| `src_len` | `() -> i32` | Source length |
| `src_read` | `(dest, off, n: i32) -> i32` | Copies source `[off,off+n)` into guest `dest`; returns copied |
| `src_write` | `(ptr, n: i32) -> i32` | Sets host output from guest bytes; `0` ok |
| `report` | `(ptr, n: i32) -> i32` | JSON report; `0` ok |
| `log` | `(level, ptr, n: i32)` | Truncated host log (optional) |
| `fail` | `(ptr, n: i32)` | Error message for host |

### Opts JSON

```json
{
  "rename_locals": true,
  "reorder_fields": true,
  "reorder_imports": true,
  "format": true,
  "seed": 1
}
```

### Report JSON

```json
{
  "lang": "rust",
  "renamed": 3,
  "reordered": 1,
  "changed": true
}
```

## Limits

Host enforces wall-clock timeout (~5s), max source size (engine `MaxFileBytes`), and guest memory bounds on all pointer ops.
