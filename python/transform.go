package python

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Opts configures Python source transform.
type Opts struct {
	RenameLocals   bool
	ReorderFields  bool
	ReorderImports bool
	Format         bool
	Seed           int64
}

// Result summarizes Python transform.
type Result struct {
	Renamed   int
	Reordered int
	Changed   bool
}

var (
	importLine = regexp.MustCompile(`^(?:from\s+\S+\s+import\s+.+|import\s+.+)$`)
	defLine    = regexp.MustCompile(`^(\s*)def\s+([A-Za-z_][\w]*)\s*\((.*)\)\s*:`)
	assignRe   = regexp.MustCompile(`^(\s*)([a-z_][\w]*)\s*=`)
	identRe    = regexp.MustCompile(`\b([A-Za-z_][\w]*)\b`)
)

// Transform applies best-effort semantic-preserving transforms to Python source.
func Transform(src []byte, opts Opts) ([]byte, Result, error) {
	rep := Result{}
	if len(src) == 0 {
		return src, rep, nil
	}
	if bytes.Contains(src, []byte{0}) {
		return src, rep, nil
	}
	text := string(src)
	lines := splitKeepNL(text)
	if opts.ReorderImports {
		n, next := sortImports(lines)
		lines = next
		rep.Reordered += n
	}
	if opts.RenameLocals {
		n, next := renameLocals(lines, opts.Seed)
		lines = next
		rep.Renamed = n
	}
	if opts.ReorderFields {
		n, next := reorderClassAssigns(lines)
		lines = next
		rep.Reordered += n
	}
	out := []byte(strings.Join(lines, ""))
	if opts.Format {
		out = normalizeNewlines(out)
	}
	rep.Changed = !bytes.Equal(out, src)
	return out, rep, nil
}

func splitKeepNL(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func sortImports(lines []string) (int, []string) {
	// Find leading import block (after optional coding/comments/docstring skip simple: from start).
	i := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "#") {
			i++
			continue
		}
		break
	}
	start := i
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			break
		}
		if !importLine.MatchString(t) {
			break
		}
		i++
	}
	if i-start < 2 {
		return 0, lines
	}
	block := append([]string(nil), lines[start:i]...)
	sorted := append([]string(nil), block...)
	sort.SliceStable(sorted, func(a, b int) bool {
		return strings.TrimSpace(sorted[a]) < strings.TrimSpace(sorted[b])
	})
	same := true
	for j := range block {
		if block[j] != sorted[j] {
			same = false
			break
		}
	}
	if same {
		return 0, lines
	}
	out := append([]string(nil), lines...)
	copy(out[start:i], sorted)
	return 1, out
}

func renameLocals(lines []string, seed int64) (int, []string) {
	// Per-function: collect assigned names / params that look private; rename within that indent block.
	out := append([]string(nil), lines...)
	total := 0
	for i := 0; i < len(out); i++ {
		m := defLine.FindStringSubmatch(strings.TrimRight(out[i], "\n"))
		if m == nil {
			continue
		}
		baseIndent := len(m[1])
		params := splitParams(m[3])
		bodyStart := i + 1
		bodyEnd := bodyStart
		for bodyEnd < len(out) {
			t := strings.TrimRight(out[bodyEnd], "\n")
			if strings.TrimSpace(t) == "" {
				bodyEnd++
				continue
			}
			ind := leadingSpaces(t)
			if ind <= baseIndent && !strings.HasPrefix(strings.TrimSpace(t), "#") {
				break
			}
			bodyEnd++
		}
		locals := map[string]string{}
		seq := 0
		for _, p := range params {
			if !isPrivateLocal(p) {
				continue
			}
			locals[p] = synthetic(seed, total+seq)
			seq++
		}
		for j := bodyStart; j < bodyEnd; j++ {
			if am := assignRe.FindStringSubmatch(strings.TrimRight(out[j], "\n")); am != nil {
				name := am[2]
				if isPrivateLocal(name) {
					if _, ok := locals[name]; !ok {
						locals[name] = synthetic(seed, total+seq)
						seq++
					}
				}
			}
		}
		if len(locals) == 0 {
			continue
		}
		total += len(locals)
		// rewrite def line params
		out[i] = replaceIdentsLine(out[i], locals)
		for j := bodyStart; j < bodyEnd; j++ {
			out[j] = replaceIdentsLine(out[j], locals)
		}
		i = bodyEnd - 1
	}
	return total, out
}

func reorderClassAssigns(lines []string) (int, []string) {
	out := append([]string(nil), lines...)
	n := 0
	classRe := regexp.MustCompile(`^(\s*)class\s+`)
	for i := 0; i < len(out); i++ {
		m := classRe.FindStringSubmatch(strings.TrimRight(out[i], "\n"))
		if m == nil {
			continue
		}
		base := len(m[1])
		j := i + 1
		var assigns []struct {
			idx  int
			name string
			line string
		}
		for j < len(out) {
			t := strings.TrimRight(out[j], "\n")
			trim := strings.TrimSpace(t)
			if trim == "" || strings.HasPrefix(trim, "#") {
				j++
				continue
			}
			ind := leadingSpaces(t)
			if ind <= base {
				break
			}
			if ind == base+4 || (base == 0 && ind == 4) {
				if am := assignRe.FindStringSubmatch(t); am != nil && !strings.HasPrefix(trim, "def ") {
					assigns = append(assigns, struct {
						idx  int
						name string
						line string
					}{idx: j, name: am[2], line: out[j]})
				} else {
					// methods / other break contiguous assign block handling: only sort contiguous run
					if len(assigns) >= 2 {
						n += sortAssignRun(out, assigns)
					}
					assigns = nil
				}
			}
			j++
		}
		if len(assigns) >= 2 {
			n += sortAssignRun(out, assigns)
		}
		i = j - 1
	}
	return n, out
}

func sortAssignRun(out []string, assigns []struct {
	idx  int
	name string
	line string
}) int {
	sorted := append([]struct {
		idx  int
		name string
		line string
	}(nil), assigns...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].name < sorted[b].name })
	same := true
	for i := range assigns {
		if assigns[i].name != sorted[i].name {
			same = false
			break
		}
	}
	if same {
		return 0
	}
	for i := range assigns {
		out[assigns[i].idx] = sorted[i].line
	}
	return 1
}

func splitParams(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "self" || p == "cls" || strings.HasPrefix(p, "*") {
			continue
		}
		if i := strings.IndexAny(p, ":="); i >= 0 {
			p = strings.TrimSpace(p[:i])
		}
		out = append(out, p)
	}
	return out
}

func isPrivateLocal(name string) bool {
	if name == "" || name == "_" || name == "self" || name == "cls" {
		return false
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return false
	}
	return true
}

func synthetic(seed int64, seq int) string {
	if seed == 0 {
		seed = 1
	}
	return fmt.Sprintf("_p%x_%d", uint64(seed)&0xffff, seq)
}

func replaceIdentsLine(line string, locals map[string]string) string {
	return identRe.ReplaceAllStringFunc(line, func(id string) string {
		if n, ok := locals[id]; ok {
			return n
		}
		return id
	})
}

func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}

func normalizeNewlines(b []byte) []byte {
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasSuffix(s, "\n") && len(s) > 0 {
		s += "\n"
	}
	return []byte(s)
}
