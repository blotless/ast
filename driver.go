package ast

// Lang identifies a source language for transform.
type Lang string

const (
	LangGo     Lang = "go"
	LangPython Lang = "python"
)

// Opts configures semantic-preserving AST transform.
type Opts struct {
	RenameLocals   bool  `json:"rename_locals"`
	ReorderFields  bool  `json:"reorder_fields"`
	ReorderImports bool  `json:"reorder_imports"`
	Format         bool  `json:"format"`
	Seed           int64 `json:"seed"`
}

// DefaultOpts returns conservative transform defaults.
func DefaultOpts() Opts {
	return Opts{
		RenameLocals:   true,
		ReorderFields:  true,
		ReorderImports: true,
		Format:         true,
		Seed:           1,
	}
}

// Report summarizes what a driver changed.
type Report struct {
	Lang      Lang `json:"lang"`
	Renamed   int  `json:"renamed"`
	Reordered int  `json:"reordered"`
	Changed   bool `json:"changed"`
}

// Driver is an in-process language transform implementation.
type Driver interface {
	Lang() Lang
	Transform(src []byte, opts Opts) ([]byte, Report, error)
}
