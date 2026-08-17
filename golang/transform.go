package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Opts configures Go source transform.
type Opts struct {
	RenameLocals   bool
	ReorderFields  bool
	ReorderImports bool
	Format         bool
	Seed           int64
}

// Result summarizes Go transform.
type Result struct {
	Renamed   int
	Reordered int
	Changed   bool
}

// Transform applies semantic-preserving transforms to Go source.
func Transform(src []byte, opts Opts) ([]byte, Result, error) {
	rep := Result{}
	if len(src) == 0 {
		return src, rep, nil
	}
	f, err := Parse("transform.go", src)
	if err != nil {
		return src, rep, err
	}
	out := append([]byte(nil), src...)
	if opts.RenameLocals {
		n, next, err := renameLocals(f, out, opts.Seed)
		if err != nil {
			return src, rep, err
		}
		out = next
		rep.Renamed = n
	}
	if opts.ReorderImports || opts.ReorderFields {
		f2, err := Parse("transform.go", out)
		if err != nil {
			return src, rep, err
		}
		n, next, err := reorder(f2, out, opts)
		if err != nil {
			return src, rep, err
		}
		out = next
		rep.Reordered = n
	}
	if opts.Format {
		formatted, err := format.Source(out)
		if err == nil {
			out = formatted
		}
	}
	rep.Changed = !bytes.Equal(out, src)
	return out, rep, nil
}

func renameLocals(f *File, src []byte, seed int64) (int, []byte, error) {
	// Parser Ident.Obj groups uses of the same local without a type-check.
	// go/types would need complete files and is out of scope for this transform.
	objUses := map[*ast.Object][]token.Pos{} //nolint:staticcheck // SA1019: parser objects, not types.Info
	ast.Inspect(f.AST, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Obj == nil {
			return true
		}
		objUses[id.Obj] = append(objUses[id.Obj], id.Pos())
		return true
	})

	type repl struct {
		start, end int
		name       string
	}
	var reps []repl
	renamed := 0
	seq := 0
	for obj, poses := range objUses {
		if obj == nil || obj.Name == "_" || obj.Name == "" {
			continue
		}
		if unicode.IsUpper(firstRune(obj.Name)) {
			continue
		}
		if isPackageLevel(obj) {
			continue
		}
		if obj.Kind != ast.Var {
			continue
		}
		newName := syntheticName(seed, seq)
		seq++
		renamed++
		for _, p := range poses {
			off := f.Fset.Position(p).Offset
			reps = append(reps, repl{start: off, end: off + len(obj.Name), name: newName})
		}
	}
	sort.Slice(reps, func(i, j int) bool { return reps[i].start > reps[j].start })
	out := append([]byte(nil), src...)
	for _, r := range reps {
		out = concat(out[:r.start], []byte(r.name), out[r.end:])
	}
	return renamed, out, nil
}

func isPackageLevel(obj *ast.Object) bool { //nolint:staticcheck // SA1019: parser objects, not types.Info
	if obj.Decl == nil {
		return true
	}
	switch obj.Decl.(type) {
	case *ast.FuncDecl:
		return true
	case *ast.ValueSpec:
		return true
	case *ast.Field:
		return false
	case *ast.AssignStmt:
		return false
	default:
		return false
	}
}

func reorder(f *File, src []byte, opts Opts) (int, []byte, error) {
	n := 0
	out := src
	if opts.ReorderImports && f.AST.Imports != nil && len(f.AST.Imports) > 1 {
		next, ok := sortImportBlock(f, out)
		if ok {
			out = next
			n++
			f2, err := Parse("transform.go", out)
			if err != nil {
				return n, out, err
			}
			f = f2
		}
	}
	if opts.ReorderFields {
		next, c, err := sortPrivateStructFields(f, out)
		if err != nil {
			return n, out, err
		}
		out = next
		n += c
	}
	return n, out, nil
}

func sortImportBlock(f *File, src []byte) ([]byte, bool) {
	var gen *ast.GenDecl
	for _, d := range f.AST.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		if gen != nil {
			return src, false
		}
		gen = gd
	}
	if gen == nil || len(gen.Specs) < 2 {
		return src, false
	}
	type specInfo struct {
		path string
		text string
	}
	infos := make([]specInfo, 0, len(gen.Specs))
	for _, s := range gen.Specs {
		is := s.(*ast.ImportSpec)
		start := f.Fset.Position(is.Pos()).Offset
		end := f.Fset.Position(is.End()).Offset
		for end < len(src) && (src[end] == '\r' || src[end] == '\n') {
			end++
		}
		path, _ := strconv.Unquote(is.Path.Value)
		infos = append(infos, specInfo{path: path, text: string(src[start:end])})
	}
	sorted := append([]specInfo(nil), infos...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].path < sorted[j].path })
	same := true
	for i := range infos {
		if infos[i].path != sorted[i].path {
			same = false
			break
		}
	}
	if same {
		return src, false
	}
	start := f.Fset.Position(gen.Specs[0].Pos()).Offset
	last := gen.Specs[len(gen.Specs)-1]
	end := f.Fset.Position(last.End()).Offset
	for end < len(src) && (src[end] == '\r' || src[end] == '\n') {
		end++
	}
	var b strings.Builder
	for _, s := range sorted {
		b.WriteString(s.text)
		if !strings.HasSuffix(s.text, "\n") {
			b.WriteByte('\n')
		}
	}
	return concat(src[:start], []byte(b.String()), src[end:]), true
}

func sortPrivateStructFields(f *File, src []byte) ([]byte, int, error) {
	type fieldSpan struct {
		start, end int
		name       string
		text       string
	}
	count := 0
	out := append([]byte(nil), src...)
	var jobs []struct {
		start, end int
		fields     []fieldSpan
	}
	ast.Inspect(f.AST, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil || len(st.Fields.List) < 2 {
			return true
		}
		allUnexported := true
		var fields []fieldSpan
		for _, field := range st.Fields.List {
			if len(field.Names) == 0 {
				allUnexported = false
				break
			}
			for _, id := range field.Names {
				if unicode.IsUpper(firstRune(id.Name)) {
					allUnexported = false
				}
			}
			start := f.Fset.Position(field.Pos()).Offset
			end := f.Fset.Position(field.End()).Offset
			for end < len(src) && src[end] != '\n' {
				end++
			}
			if end < len(src) && src[end] == '\n' {
				end++
			}
			name := field.Names[0].Name
			fields = append(fields, fieldSpan{start: start, end: end, name: name, text: string(src[start:end])})
		}
		if !allUnexported || len(fields) < 2 {
			return true
		}
		sorted := append([]fieldSpan(nil), fields...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })
		same := true
		for i := range fields {
			if fields[i].name != sorted[i].name {
				same = false
				break
			}
		}
		if same {
			return true
		}
		jobs = append(jobs, struct {
			start, end int
			fields     []fieldSpan
		}{start: fields[0].start, end: fields[len(fields)-1].end, fields: sorted})
		return true
	})
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].start > jobs[j].start })
	for _, j := range jobs {
		var b strings.Builder
		for _, fl := range j.fields {
			b.WriteString(fl.text)
		}
		out = concat(out[:j.start], []byte(b.String()), out[j.end:])
		count++
	}
	return out, count, nil
}

func syntheticName(seed int64, seq int) string {
	if seed == 0 {
		seed = 1
	}
	return fmt.Sprintf("_b%x_%d", uint64(seed)&0xffff, seq)
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func concat(a, b, c []byte) []byte {
	out := make([]byte, 0, len(a)+len(b)+len(c))
	out = append(out, a...)
	out = append(out, b...)
	out = append(out, c...)
	return out
}
