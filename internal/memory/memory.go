package memory

import (
	"context"
	"errors"

	"github.com/tetratelabs/wazero/api"

	"github.com/wasilibs/go-zstd/internal/wasmabi"
)

var errFailedWrite = errors.New("failed to write to wasm memory")

type MallocABI struct {
	wasmabi.ABIBase

	Malloc api.Function
	Free   api.Function
}

func (abi *MallocABI) Reserve(ctx context.Context, size uint32) Allocation {
	stack := abi.CallStack
	stack[0] = uint64(size)
	if err := abi.Malloc.CallWithStack(ctx, stack); err != nil {
		panic(err)
	}
	ptr := uint32(stack[0])
	return Allocation{
		size:   size,
		bufPtr: ptr,
		abi:    abi,
	}
}

type Allocation struct {
	size    uint32
	bufPtr  uint32
	nextIdx uint32
	abi     *MallocABI
}

func (a *Allocation) Free() {
	callStack := a.abi.CallStack
	callStack[0] = uint64(a.bufPtr)
	if err := a.abi.Free.CallWithStack(context.Background(), callStack); err != nil {
		panic(err)
	}
}

func (a *Allocation) Allocate(size uint32) uintptr {
	if a.nextIdx+size > a.size {
		panic("not enough reserved memory in allocation")
	}

	ptr := a.bufPtr + a.nextIdx
	a.nextIdx += size
	return uintptr(ptr)
}

func (a *Allocation) Write(b []byte) uintptr {
	ptr := a.Allocate(uint32(len(b)))
	if !a.abi.Memory.Write(uint32(ptr), b) {
		panic(errFailedWrite)
	}
	return ptr
}
