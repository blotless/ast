//! Example blotless Layer B plugin (capability ABI, no WASI).
//!
//! Conservative transform: alphabetically sort a leading run of `use ` lines.
//! Public APIs and string literals are left unchanged.

use std::cmp::Ordering;

#[link(wasm_import_module = "blotless")]
extern "C" {
    fn abi_version() -> u32;
    fn get_opts(ptr: u32, maxlen: u32) -> u32;
    fn src_len() -> u32;
    fn src_read(dest: u32, off: u32, n: u32) -> u32;
    fn src_write(ptr: u32, n: u32) -> u32;
    fn report(ptr: u32, n: u32) -> u32;
    fn fail(ptr: u32, n: u32);
}

const CAP: usize = 2 * 1024 * 1024;
const REPORT_OFF: usize = CAP - 4096;
static mut MEM: [u8; CAP] = [0; CAP];

#[no_mangle]
pub extern "C" fn transform() -> i32 {
    match unsafe { run() } {
        Ok(()) => 0,
        Err(msg) => unsafe {
            write_fail(&msg);
            1
        },
    }
}

unsafe fn run() -> Result<(), String> {
    if abi_version() < 1 {
        return Err("unsupported blotless ABI".into());
    }

    let base = MEM.as_mut_ptr() as u32;
    let opt_n = get_opts(base, 1024);
    let opts = parse_opts(&MEM[..opt_n as usize]);
    if !opts.reorder_imports {
        return echo_unchanged();
    }

    let n = src_len() as usize;
    if n == 0 {
        return echo_unchanged();
    }
    if n >= REPORT_OFF {
        return Err("source larger than plugin buffer".into());
    }

    let got = src_read(base, 0, n as u32) as usize;
    if got != n {
        return Err("src_read short".into());
    }
    let src = std::str::from_utf8(&MEM[..n]).map_err(|_| "source is not UTF-8".to_string())?;
    let (out, reordered) = sort_use_block(src);

    let bytes = out.as_bytes();
    if bytes.len() >= REPORT_OFF {
        return Err("output larger than plugin buffer".into());
    }
    MEM[..bytes.len()].copy_from_slice(bytes);
    if src_write(base, bytes.len() as u32) != 0 {
        return Err("src_write failed".into());
    }

    write_report(reordered, reordered > 0)
}

unsafe fn echo_unchanged() -> Result<(), String> {
    let n = src_len() as usize;
    if n == 0 {
        return write_report(0, false);
    }
    if n >= REPORT_OFF {
        return Err("source larger than plugin buffer".into());
    }
    let base = MEM.as_mut_ptr() as u32;
    if src_read(base, 0, n as u32) as usize != n {
        return Err("src_read short".into());
    }
    if src_write(base, n as u32) != 0 {
        return Err("src_write failed".into());
    }
    write_report(0, false)
}

unsafe fn write_report(reordered: i32, changed: bool) -> Result<(), String> {
    let json = format!(
        r#"{{"lang":"rust","renamed":0,"reordered":{reordered},"changed":{changed}}}"#
    );
    let b = json.as_bytes();
    MEM[REPORT_OFF..REPORT_OFF + b.len()].copy_from_slice(b);
    let ptr = MEM.as_ptr() as usize + REPORT_OFF;
    if report(ptr as u32, b.len() as u32) != 0 {
        return Err("report failed".into());
    }
    Ok(())
}

unsafe fn write_fail(msg: &str) {
    let b = msg.as_bytes();
    let n = b.len().min(1024);
    MEM[..n].copy_from_slice(&b[..n]);
    fail(MEM.as_ptr() as u32, n as u32);
}

struct Opts {
    reorder_imports: bool,
}

fn parse_opts(raw: &[u8]) -> Opts {
    let s = std::str::from_utf8(raw).unwrap_or("");
    Opts {
        reorder_imports: !s.contains("\"reorder_imports\":false"),
    }
}

fn sort_use_block(src: &str) -> (String, i32) {
    let lines: Vec<&str> = src.split_inclusive('\n').collect();
    let mut i = 0usize;
    while i < lines.len() {
        let t = lines[i].trim();
        if t.is_empty() || t.starts_with("//!") || t.starts_with("#[") || t.starts_with("#!") {
            i += 1;
            continue;
        }
        break;
    }
    let start = i;
    while i < lines.len() {
        if lines[i].trim().starts_with("use ") {
            i += 1;
            continue;
        }
        break;
    }
    if i - start < 2 {
        return (src.to_string(), 0);
    }
    let orig = lines[start..i].to_vec();
    let mut block = orig.clone();
    block.sort_by(|a, b| a.trim().cmp(b.trim()).then(Ordering::Equal));
    if block == orig {
        return (src.to_string(), 0);
    }
    let mut out = String::new();
    for line in &lines[..start] {
        out.push_str(line);
    }
    for line in &block {
        out.push_str(line);
    }
    for line in &lines[i..] {
        out.push_str(line);
    }
    (out, 1)
}

#[cfg(test)]
mod tests {
    use super::sort_use_block;

    #[test]
    fn sorts_use() {
        let src = "use std::fmt;\nuse std::cmp;\n\nfn main() {}\n";
        let (out, n) = sort_use_block(src);
        assert_eq!(n, 1);
        assert!(out.starts_with("use std::cmp;\nuse std::fmt;\n"));
    }
}
