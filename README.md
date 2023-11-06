# go-zstd

A wrapper of the official zstd library, using wazero
to allow usage in pure go applications.

This originally started before realizing a pure go implementation
exists in `compress`. It performs better and should be used, this
repository serves as a historical example of wrapping libraries.
