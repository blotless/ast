package ast

import (
	"github.com/blotless/ast/golang"
	"github.com/blotless/ast/python"
)

func init() {
	Register(goDriver{})
	Register(pyDriver{})
}

type goDriver struct{}

func (goDriver) Lang() Lang { return LangGo }

func (goDriver) Transform(src []byte, opts Opts) ([]byte, Report, error) {
	out, r, err := golang.Transform(src, golang.Opts{
		RenameLocals:   opts.RenameLocals,
		ReorderFields:  opts.ReorderFields,
		ReorderImports: opts.ReorderImports,
		Format:         opts.Format,
		Seed:           opts.Seed,
	})
	return out, Report{Lang: LangGo, Renamed: r.Renamed, Reordered: r.Reordered, Changed: r.Changed}, err
}

type pyDriver struct{}

func (pyDriver) Lang() Lang { return LangPython }

func (pyDriver) Transform(src []byte, opts Opts) ([]byte, Report, error) {
	out, r, err := python.Transform(src, python.Opts{
		RenameLocals:   opts.RenameLocals,
		ReorderFields:  opts.ReorderFields,
		ReorderImports: opts.ReorderImports,
		Format:         opts.Format,
		Seed:           opts.Seed,
	})
	return out, Report{Lang: LangPython, Renamed: r.Renamed, Reordered: r.Reordered, Changed: r.Changed}, err
}
