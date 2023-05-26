package bench

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	ddzstd "github.com/DataDog/zstd"
	"github.com/wasilibs/go-zstd"
)

var raw []byte
var (
	ErrNoPayloadEnv = errors.New("PAYLOAD env was not set")
)

func init() {
	var err error
	payload := os.Getenv("PAYLOAD")
	if len(payload) > 0 {
		raw, err = ioutil.ReadFile(payload)
		if err != nil {
			fmt.Printf("Error opening payload: %s\n", err)
		}
	}
}

func BenchmarkCompression(b *testing.B) {
	tests := []struct {
		name string
		fn   func([]byte, []byte) ([]byte, error)
	}{
		{
			name: "wasilibs",
			fn:   zstd.Compress,
		},
		{
			name: "datadog",
			fn:   ddzstd.Compress,
		},
	}

	if raw == nil {
		b.Fatal(ErrNoPayloadEnv)
	}
	dst := make([]byte, zstd.CompressBound(len(raw)))
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for _, tc := range tests {
		tt := tc
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := tt.fn(dst, raw)
				if err != nil {
					b.Fatalf("Failed compressing: %s", err)
				}
			}
		})
	}
}
