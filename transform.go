package ast

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	drivers  = map[Lang]Driver{}
	wasmPath = map[Lang]string{}
	extLang  = map[string]Lang{
		".go": LangGo,
		".py": LangPython,
	}
)

// Register adds an in-process driver (built-in languages).
func Register(d Driver) {
	if d == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	drivers[d.Lang()] = d
}

// RegisterWASM binds lang to a local .wasm plugin path (capability-sandboxed).
func RegisterWASM(lang Lang, path string) error {
	lang = Lang(strings.ToLower(strings.TrimSpace(string(lang))))
	path = strings.TrimSpace(path)
	if lang == "" || path == "" {
		return fmt.Errorf("ast: wasm registration requires lang and path")
	}
	if strings.Contains(path, "://") {
		return fmt.Errorf("ast: wasm path must be a local file, not a URL")
	}
	mu.Lock()
	defer mu.Unlock()
	wasmPath[lang] = path
	mapDefaultExtsLocked(lang)
	return nil
}

// DefaultExts returns conventional file extensions for a language id.
func DefaultExts(lang Lang) []string {
	switch Lang(strings.ToLower(string(lang))) {
	case LangGo:
		return []string{".go"}
	case LangPython:
		return []string{".py"}
	case "rust", "rs":
		return []string{".rs"}
	case "javascript", "js":
		return []string{".js", ".mjs", ".cjs", ".jsx"}
	case "typescript", "ts":
		return []string{".ts", ".tsx"}
	case "ruby", "rb":
		return []string{".rb"}
	case "java":
		return []string{".java"}
	case "kotlin", "kt":
		return []string{".kt", ".kts"}
	case "csharp", "cs":
		return []string{".cs"}
	case "php":
		return []string{".php"}
	case "swift":
		return []string{".swift"}
	case "scala":
		return []string{".scala"}
	default:
		if lang == "" {
			return nil
		}
		return []string{"." + string(lang)}
	}
}

func mapDefaultExtsLocked(lang Lang) {
	for _, ext := range DefaultExts(lang) {
		if existing, ok := extLang[ext]; ok && existing != lang {
			continue
		}
		extLang[ext] = lang
	}
}

// MapExt associates a file extension (with dot) to a language id.
func MapExt(ext string, lang Lang) {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	mu.Lock()
	defer mu.Unlock()
	extLang[ext] = lang
}

// LangFromPath returns the language for a file path, if mapped.
func LangFromPath(path string) (Lang, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	mu.RLock()
	defer mu.RUnlock()
	lang, ok := extLang[ext]
	return lang, ok
}

// HasDriver reports whether lang has a built-in or WASM driver.
func HasDriver(lang Lang) bool {
	mu.RLock()
	defer mu.RUnlock()
	if _, ok := drivers[lang]; ok {
		return true
	}
	_, ok := wasmPath[lang]
	return ok
}

// Transform runs the language driver (built-in preferred over WASM).
func Transform(lang Lang, src []byte, opts Opts) ([]byte, Report, error) {
	lang = Lang(strings.ToLower(strings.TrimSpace(string(lang))))
	mu.RLock()
	d, ok := drivers[lang]
	wasm := wasmPath[lang]
	mu.RUnlock()
	if ok {
		return d.Transform(src, opts)
	}
	if wasm != "" {
		return runWASM(wasm, lang, src, opts)
	}
	return src, Report{Lang: lang}, fmt.Errorf("ast: no driver for %q", lang)
}
