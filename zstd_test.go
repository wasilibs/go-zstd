package zstd

import (
	"testing"
)

// Test our version of compress bound vs C implementation
func TestCompressBound(t *testing.T) {
	tests := []int{0, 1, 2, 10, 456, 15468, 1313, 512, 2147483632}
	for _, test := range tests {
		if CompressBound(test) != cCompressBound(test) {
			t.Fatalf("For %v, results are different: %v (actual) != %v (expected)", test,
				CompressBound(test), cCompressBound(test))
		}
	}
}

// Test compression
func TestCompressDecompress(t *testing.T) {
	input := []byte("Hello World!")
	out, err := Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	out2 := make([]byte, 1000)
	out2, err = Compress(out2, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	t.Logf("Compressed: %v", out)
	rein, err := Decompress(nil, out)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}
	rein2 := make([]byte, 10)
	rein2, err = Decompress(rein2, out2)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}

	if string(input) != string(rein) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
	if string(input) != string(rein2) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
}
