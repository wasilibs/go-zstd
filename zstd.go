package zstd

import "C"
import (
	"context"
	_ "embed"
	"errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/wasilibs/go-zstd/internal/memory"
	"github.com/wasilibs/go-zstd/internal/wasmabi"
	"runtime"
	"sync"
)

const DefaultCompression = 5

const (
	// decompressSizeBufferLimit is the limit we set on creating a decompression buffer for the Decompress API
	// This is made to prevent DOS from maliciously-created payloads (aka zipbomb).
	// For large payloads with a compression ratio > 10, you can do your own allocation and pass it to the method:
	// dst := make([]byte, 1GB)
	// decompressed, err := zstd.Decompress(dst, src)
	decompressSizeBufferLimit = 1000 * 1000

	zstdFrameHeaderSizeMin = 2 // From zstd.h. Since it's experimental API, hardcoding it
)

//go:embed internal/wasm/libzstd.wasm
var libZSTD []byte

var (
	wasmRT       wazero.Runtime
	wasmCompiled wazero.CompiledModule

	errFailedRead = errors.New("failed to read from wasm memory")
)

func init() {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	code, err := rt.CompileModule(ctx, libZSTD)
	if err != nil {
		panic(err)
	}
	wasmCompiled = code
	wasmRT = rt
}

type libzstdABI struct {
	memory.MallocABI

	zstdCompress            api.Function
	zstdCompressBound       api.Function
	zstdGetFrameContentSize api.Function
	zstdDecompress          api.Function
}

func newABI() *libzstdABI {
	ctx := context.Background()
	mod, err := wasmRT.InstantiateModule(ctx, wasmCompiled, wazero.NewModuleConfig())
	if err != nil {
		panic(err)
	}

	abi := &libzstdABI{
		MallocABI: memory.MallocABI{
			ABIBase: wasmabi.ABIBase{
				CallStack: make([]uint64, 5),
				Mod:       mod,
				Memory:    mod.Memory(),
			},
			Malloc: mod.ExportedFunction("malloc"),
			Free:   mod.ExportedFunction("free"),
		},
		zstdCompress:            mod.ExportedFunction("ZSTD_compress"),
		zstdCompressBound:       mod.ExportedFunction("ZSTD_compressBound"),
		zstdGetFrameContentSize: mod.ExportedFunction("ZSTD_getFrameContentSize"),
		zstdDecompress:          mod.ExportedFunction("ZSTD_decompress"),
	}

	runtime.SetFinalizer(abi, func(abi *libzstdABI) {
		_ = abi.Mod.Close(context.Background())
	})
	return abi
}

var abiPool = sync.Pool{
	New: func() interface{} {
		return newABI()
	},
}

// Compress src into dst.  If you have a buffer to use, you can pass it to
// prevent allocation.  If it is too small, or if nil is passed, a new buffer
// will be allocated and returned.
func Compress(dst, src []byte) ([]byte, error) {
	return CompressLevel(dst, src, DefaultCompression)
}

// CompressLevel is the same as Compress but you can pass a compression level
func CompressLevel(dst, src []byte, level int) ([]byte, error) {
	ctx := context.Background()

	abi := abiPool.Get().(*libzstdABI)
	defer abiPool.Put(abi)

	callStack := abi.CallStack

	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound]
	} else {
		dst = make([]byte, bound)
	}

	alloc := abi.Reserve(ctx, uint32(len(dst)+len(src)))
	defer alloc.Free()

	dstWasm := alloc.Allocate(uint32(len(dst)))
	srcWasm := alloc.Write(src)

	callStack[0] = uint64(dstWasm)
	callStack[1] = uint64(len(dst))
	callStack[2] = uint64(srcWasm)
	callStack[3] = uint64(len(src))
	callStack[4] = uint64(level)

	if err := abi.zstdCompress.CallWithStack(ctx, callStack); err != nil {
		panic(err)
	}

	written := int(callStack[0])
	if written < 0 {
		return nil, errors.New("zstd: compress failed")
	}

	dstWasmBuf, ok := abi.Memory.Read(uint32(dstWasm), uint32(len(dst)))
	if !ok {
		panic(errFailedRead)
	}
	copy(dst, dstWasmBuf)
	return dst[:written], nil
}

// CompressBound returns the worst case size needed for a destination buffer,
// which can be used to preallocate a destination buffer or select a previously
// allocated buffer from a pool.
// https://github.com/facebook/zstd/blob/23a0643ef1e9b375fd0acb62175791670ad49936/lib/zstd.h#L232
func CompressBound(srcSize int) int {
	lowLimit := 128 << 10 // 128 kB
	var margin int
	if srcSize < lowLimit {
		margin = (lowLimit - srcSize) >> 11
	}
	return srcSize + (srcSize >> 8) + margin
}

// cCompressBound is a wasm call to check the go implementation above against the c code.
func cCompressBound(srcSize int) int {
	ctx := context.Background()

	abi := abiPool.Get().(*libzstdABI)
	defer abiPool.Put(abi)

	callStack := abi.CallStack

	callStack[0] = uint64(srcSize)
	if err := abi.zstdCompressBound.CallWithStack(ctx, callStack); err != nil {
		panic(err)
	}
	return int(callStack[0])
}

// Decompress src into dst.  If you have a buffer to use, you can pass it to
// prevent allocation.  If it is too small, or if nil is passed, a new buffer
// will be allocated and returned.
func Decompress(dst, src []byte) ([]byte, error) {
	ctx := context.Background()

	abi := abiPool.Get().(*libzstdABI)
	defer abiPool.Put(abi)

	callStack := abi.CallStack

	// We need to copy the source to Wasm before determining the dst size, so we can't
	// fit it into one alloc

	alloc1 := abi.Reserve(ctx, uint32(len(src)))
	defer alloc1.Free()

	srcWasm := alloc1.Write(src)
	bound := decompressSizeHint(ctx, abi, uint32(srcWasm), len(src))
	if cap(dst) >= bound {
		dst = dst[0:cap(dst)]
	} else {
		dst = make([]byte, bound)
	}

	alloc2 := abi.Reserve(ctx, uint32(len(dst)))
	defer alloc2.Free()
	dstWasm := alloc2.Allocate(uint32(len(dst)))

	callStack[0] = uint64(dstWasm)
	callStack[1] = uint64(len(dst))
	callStack[2] = uint64(srcWasm)
	callStack[3] = uint64(len(src))

	if err := abi.zstdDecompress.CallWithStack(ctx, callStack); err != nil {
		panic(err)
	}

	written := int(callStack[0])
	if written < 0 {
		return nil, errors.New("zstd: decompress failed")
	}

	dstWasmBuf, ok := abi.Memory.Read(uint32(dstWasm), uint32(len(dst)))
	if !ok {
		panic(errFailedRead)
	}
	copy(dst, dstWasmBuf)
	return dst[:written], nil
}

// decompressSizeHint tries to give a hint on how much of the output buffer size we should have
// based on zstd frame descriptors. To prevent DOS from maliciously-created payloads, limit the size
func decompressSizeHint(ctx context.Context, abi *libzstdABI, srcWasm uint32, srcLen int) int {
	callStack := abi.CallStack

	// 1 MB or 10x input size
	upperBound := 10 * srcLen
	if upperBound < decompressSizeBufferLimit {
		upperBound = decompressSizeBufferLimit
	}

	hint := upperBound
	if srcLen >= zstdFrameHeaderSizeMin {
		callStack[0] = uint64(srcWasm)
		callStack[1] = uint64(srcLen)
		if err := abi.zstdGetFrameContentSize.CallWithStack(ctx, callStack); err != nil {
			panic(err)
		}
		hint = int(callStack[0])
		if hint < 0 { // On error, just use upperBound
			hint = upperBound
		}
		if hint == 0 { // When compressing the empty slice, we need an output of at least 1 to pass down to the C lib
			hint = 1
		}
	}

	// Take the minimum of both
	if hint > upperBound {
		return upperBound
	}
	return hint
}
