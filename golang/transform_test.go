package golang_test

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/blotless/ast/golang"
)

func TestTransformRenamesLocalsKeepsExported(t *testing.T) {
	src := []byte(`package p

import (
	"fmt"
	"bytes"
)

type box struct {
	zebra int
	alpha int
}

func Hello(name string) string {
	tmp := name
	return fmt.Sprintf("%s", tmp)
}
`)
	out, r, err := golang.Transform(src, golang.Opts{
		RenameLocals:   true,
		ReorderFields:  true,
		ReorderImports: true,
		Format:         true,
		Seed:           7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Changed {
		t.Fatal("expected change")
	}
	if !bytes.Contains(out, []byte("func Hello(")) {
		t.Fatalf("exported Hello renamed: %s", out)
	}
	if bytes.Contains(out, []byte("tmp :=")) {
		t.Fatalf("tmp not renamed: %s", out)
	}
	if !strings.Contains(string(out), `"bytes"`) {
		t.Fatalf("missing bytes import: %s", out)
	}
	// imports sorted: bytes before fmt
	if i, j := bytes.Index(out, []byte(`"bytes"`)), bytes.Index(out, []byte(`"fmt"`)); i < 0 || j < 0 || i > j {
		t.Fatalf("imports not sorted: %s", out)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "out.go", out, 0); err != nil {
		t.Fatalf("parse out: %v\n%s", err, out)
	}
}

func TestTransformParseError(t *testing.T) {
	_, _, err := golang.Transform([]byte("not go"), golang.Opts{RenameLocals: true})
	if err == nil {
		t.Fatal("expected error")
	}
}
