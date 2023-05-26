package wasmabi

import "github.com/tetratelabs/wazero/api"

type ABIBase struct {
	CallStack []uint64

	Mod    api.Module
	Memory api.Memory
}
