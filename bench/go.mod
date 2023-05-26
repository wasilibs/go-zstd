module bench

go 1.18

require (
	github.com/DataDog/zstd v1.5.2
	github.com/klauspost/compress v1.16.5
	github.com/wasilibs/go-zstd v0.0.0-00010101000000-000000000000
)

require (
	github.com/magefile/mage v1.15.0 // indirect
	github.com/tetratelabs/wazero v1.1.0 // indirect
)

replace github.com/wasilibs/go-zstd => ../
