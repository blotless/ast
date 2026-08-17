package ast

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const abiVersion = 1

type wasmSession struct {
	src    []byte
	out    []byte
	opts   Opts
	lang   Lang
	report Report
	failed string
}

var (
	wasmOnce sync.Once
	wasmRT   wazero.Runtime
	wasmCtx  = context.Background()
)

func runtime() wazero.Runtime {
	wasmOnce.Do(func() {
		wasmRT = wazero.NewRuntime(wasmCtx)
	})
	return wasmRT
}

func runWASM(path string, lang Lang, src []byte, opts Opts) ([]byte, Report, error) {
	bin, err := os.ReadFile(path)
	if err != nil {
		return src, Report{Lang: lang}, fmt.Errorf("ast: read wasm: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rt := runtime()
	sess := &wasmSession{src: src, opts: opts, lang: lang, report: Report{Lang: lang}}

	host, err := rt.NewHostModuleBuilder("blotless").
		NewFunctionBuilder().WithFunc(func() uint32 { return abiVersion }).Export("abi_version").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, maxlen uint32) uint32 {
		return hostWriteJSON(m, ptr, maxlen, sess.opts)
	}).Export("get_opts").
		NewFunctionBuilder().WithFunc(func() uint32 { return uint32(len(sess.src)) }).Export("src_len").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, dest, off, n uint32) uint32 {
		return hostSrcRead(m, sess, dest, off, n)
	}).Export("src_read").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, n uint32) uint32 {
		return hostSrcWrite(m, sess, ptr, n)
	}).Export("src_write").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, n uint32) uint32 {
		return hostReport(m, sess, ptr, n)
	}).Export("report").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, level, ptr, n uint32) {
		// intentionally no-op beyond bounds-checked read (logs discarded in MVP)
		_, _ = hostRead(m, ptr, n)
		_ = level
	}).Export("log").
		NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, n uint32) {
		b, err := hostRead(m, ptr, n)
		if err == nil {
			sess.failed = string(b)
		}
	}).Export("fail").
		Instantiate(ctx)
	if err != nil {
		return src, sess.report, fmt.Errorf("ast: host module: %w", err)
	}
	defer host.Close(ctx)

	mod, err := rt.InstantiateWithConfig(ctx, bin, wazero.NewModuleConfig().WithName("transform-guest"))
	if err != nil {
		return src, sess.report, fmt.Errorf("ast: instantiate wasm: %w", err)
	}
	defer mod.Close(ctx)

	fn := mod.ExportedFunction("transform")
	if fn == nil {
		return src, sess.report, fmt.Errorf("ast: wasm missing export transform")
	}
	results, err := fn.Call(ctx)
	if err != nil {
		return src, sess.report, fmt.Errorf("ast: wasm transform: %w", err)
	}
	if sess.failed != "" {
		return src, sess.report, fmt.Errorf("ast: wasm fail: %s", sess.failed)
	}
	if len(results) > 0 && results[0] != 0 {
		return src, sess.report, fmt.Errorf("ast: wasm transform status %d", results[0])
	}
	if sess.out == nil {
		return src, sess.report, nil
	}
	sess.report.Changed = string(sess.out) != string(src)
	return sess.out, sess.report, nil
}

func hostRead(m api.Module, ptr, n uint32) ([]byte, error) {
	mem := m.Memory()
	if mem == nil {
		return nil, fmt.Errorf("no memory")
	}
	b, ok := mem.Read(ptr, n)
	if !ok {
		return nil, fmt.Errorf("oob read")
	}
	return append([]byte(nil), b...), nil
}

func hostWrite(m api.Module, ptr uint32, data []byte) bool {
	mem := m.Memory()
	if mem == nil {
		return false
	}
	return mem.Write(ptr, data)
}

func hostWriteJSON(m api.Module, ptr, maxlen uint32, v any) uint32 {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	if uint32(len(b)) > maxlen {
		b = b[:maxlen]
	}
	if !hostWrite(m, ptr, b) {
		return 0
	}
	return uint32(len(b))
}

func hostSrcRead(m api.Module, s *wasmSession, dest, off, n uint32) uint32 {
	if int(off) >= len(s.src) {
		return 0
	}
	end := int(off) + int(n)
	if end > len(s.src) {
		end = len(s.src)
	}
	chunk := s.src[off:end]
	if !hostWrite(m, dest, chunk) {
		return 0
	}
	return uint32(len(chunk))
}

func hostSrcWrite(m api.Module, s *wasmSession, ptr, n uint32) uint32 {
	b, err := hostRead(m, ptr, n)
	if err != nil {
		return 1
	}
	s.out = b
	return 0
}

func hostReport(m api.Module, s *wasmSession, ptr, n uint32) uint32 {
	b, err := hostRead(m, ptr, n)
	if err != nil {
		return 1
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return 1
	}
	if r.Lang == "" {
		r.Lang = s.lang
	}
	s.report = r
	return 0
}
