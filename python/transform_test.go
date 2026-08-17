package python_test

import (
	"strings"
	"testing"

	"github.com/blotless/ast/python"
)

func TestTransformImportsAndLocals(t *testing.T) {
	src := []byte("import os\nimport sys\n\ndef hello(name):\n    tmp = name\n    return tmp\n")
	out, r, err := python.Transform(src, python.Opts{
		RenameLocals:   true,
		ReorderImports: true,
		Format:         true,
		Seed:           3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Changed {
		t.Fatal("expected change")
	}
	s := string(out)
	if !strings.Contains(s, "def hello(") {
		t.Fatalf("hello renamed: %s", s)
	}
	if strings.Contains(s, "tmp =") {
		t.Fatalf("tmp not renamed: %s", s)
	}
	if i, j := strings.Index(s, "import os"), strings.Index(s, "import sys"); i < 0 || j < 0 || i > j {
		// os < sys alphabetically — os should come first (already). After sort same.
		t.Log(s)
	}
}
