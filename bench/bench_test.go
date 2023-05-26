package bench

import (
	"os"
	"path/filepath"
	"testing"

	ddzstd "github.com/DataDog/zstd"
	kpzstd "github.com/klauspost/compress/zstd"

	"github.com/wasilibs/go-zstd"
)

var benches = []string{
	"dickens",
	"mozilla",
	"mr",
	"ooffice",
	"osdb",
	"reymont",
	"samba",
	"sao",
	"webster",
	"x-ray",
	"xml",
}

func BenchmarkCompress(b *testing.B) {
	kpEncoder, _ := kpzstd.NewWriter(nil)

	modes := []struct {
		name string
		fn   func(dst, src []byte) ([]byte, error)
	}{
		{
			name: "wasilibs",
			fn:   zstd.Compress,
		},
		{
			name: "datadog",
			fn:   ddzstd.Compress,
		},
		{
			name: "klauspost",
			fn: func(dst, src []byte) ([]byte, error) {
				return kpEncoder.EncodeAll(src, dst), nil
			},
		},
	}

	for _, bench := range benches {
		b.Run(bench, func(b *testing.B) {
			raw, err := os.ReadFile(filepath.Join("silesia", bench))
			if err != nil {
				b.Fatal(err)
			}

			dst := make([]byte, zstd.CompressBound(len(raw)))
			b.SetBytes(int64(len(raw)))

			for _, tc := range modes {
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
		})
	}
}

func BenchmarkDecompress(b *testing.B) {
	kpDecoder, _ := kpzstd.NewReader(nil)

	modes := []struct {
		name string
		fn   func(dst, src []byte) ([]byte, error)
	}{
		{
			name: "wasilibs",
			fn:   zstd.Decompress,
		},
		{
			name: "datadog",
			fn:   ddzstd.Decompress,
		},
		{
			name: "klauspost",
			fn: func(dst, src []byte) ([]byte, error) {
				return kpDecoder.DecodeAll(src, dst)
			},
		},
	}

	for _, bench := range benches {
		b.Run(bench, func(b *testing.B) {
			raw, err := os.ReadFile(filepath.Join("silesia", bench))
			if err != nil {
				b.Fatal(err)
			}

			dst := make([]byte, zstd.CompressBound(len(raw)))

			src, err := zstd.Compress(nil, raw)
			if err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(len(src)))

			for _, tc := range modes {
				tt := tc
				b.Run(tt.name, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, err := tt.fn(dst, src)
						if err != nil {
							b.Fatalf("Failed decompressing: %s", err)
						}
					}
				})
			}
		})
	}
}
