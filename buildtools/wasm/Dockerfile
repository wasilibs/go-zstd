FROM ghcr.io/webassembly/wasi-sdk:wasi-sdk-20

ENV CFLAGS -O3 ${CFLAGS}
# global-base=1024 same as emcc and allows further wasm-opt optimizations
ENV LDFLAGS  ${LDFLAGS} -Wl,--global-base=1024 -mexec-model=reactor \
  -Wl,--export=ZSTD_compress \
  -Wl,--export=ZSTD_decompress \
  -Wl,--export=ZSTD_compressBound \
  -Wl,--export=ZSTD_getFrameContentSize \
  -Wl,--export=ZSTD_getErrorName \
  -Wl,--export=ZSTD_isError \
  -Wl,--export=malloc \
  -Wl,--export=free

RUN apt-get update && apt-get install -y binaryen curl

RUN mkdir -p /workspace && curl -L https://github.com/facebook/zstd/archive/refs/tags/v1.5.5.tar.gz | tar -xz --strip-components 1 -C /workspace
WORKDIR /workspace

# UNAME=AIX works around makefile always adding soname, which isn't supported by wasm-ld.
# It seems to otherwise not cause problems.
RUN make lib-nomt VERBOSE=1 UNAME=AIX

RUN wasm-opt -o lib/libzstd.wasm --low-memory-unused --flatten --rereloop --converge -O3 lib/libzstd.so.1.5.5

CMD ["cp", "lib/libzstd.wasm", "/out/libzstd.wasm"]
