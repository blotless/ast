package ast_test

import (
	"path/filepath"
	"testing"

	"github.com/blotless/ast"
)

func TestRegistryGoTransform(t *testing.T) {
	src := []byte("package p\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n")
	out, r, err := ast.Transform(ast.LangGo, src, ast.DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if !r.Changed || r.Lang != ast.LangGo {
		t.Fatalf("%+v out=%s", r, out)
	}
}

func TestWASMEcho(t *testing.T) {
	path := filepath.Join("testdata", "wasm", "echo.wasm")
	if err := ast.RegisterWASM("echo", path); err != nil {
		t.Fatal(err)
	}
	src := []byte("hello wasm")
	out, r, err := ast.Transform("echo", src, ast.DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(src) {
		t.Fatalf("out=%q", out)
	}
	if r.Changed {
		t.Fatalf("report=%+v", r)
	}
}

func TestRegisterWASMRejectsURL(t *testing.T) {
	if err := ast.RegisterWASM("x", "https://evil/x.wasm"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterWASMMapsRustExt(t *testing.T) {
	path := filepath.Join("testdata", "wasm", "echo.wasm")
	if err := ast.RegisterWASM("rust", path); err != nil {
		t.Fatal(err)
	}
	lang, ok := ast.LangFromPath("src/main.rs")
	if !ok || lang != "rust" {
		t.Fatalf("lang=%q ok=%v", lang, ok)
	}
	if !ast.HasDriver("rust") {
		t.Fatal("expected rust driver")
	}
}
