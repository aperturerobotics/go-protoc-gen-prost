# go-protoc-gen-prost

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

> A Go module that embeds protoc-gen-prost as a WebAssembly module for pure-Go Protocol Buffer code generation.

[GoDoc]: https://godoc.org/github.com/aperturerobotics/go-protoc-gen-prost
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/go-protoc-gen-prost?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/go-protoc-gen-prost
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/go-protoc-gen-prost

## Related Projects

- [aperturerobotics/protoc-gen-prost](https://github.com/aperturerobotics/protoc-gen-prost) - Fork with WASI build support
- [neoeinstein/protoc-gen-prost](https://github.com/neoeinstein/protoc-gen-prost) - Original protoc-gen-prost repository
- [tetratelabs/wazero](https://github.com/tetratelabs/wazero) - Zero-dependency WebAssembly runtime for Go

## About

This module provides the [protoc-gen-prost](https://github.com/neoeinstein/protoc-gen-prost) Protocol Buffers code generator compiled to WebAssembly with WASI support. The WASM binary is embedded directly in the Go module, enabling Rust/Prost code generation in Go applications without external dependencies or native binaries.

### Exported Functions

The WASM module exports the following functions:

- `prost_malloc(size)` - Allocate memory for input data
- `prost_free(ptr, size)` - Free allocated memory
- `prost_execute(input_ptr, input_len)` - Execute the plugin, returns output length
- `prost_get_output_ptr()` - Get pointer to output buffer
- `prost_get_output_len()` - Get output buffer length
- `prost_clear_output()` - Clear the output buffer

### How It Works

1. The host allocates memory in WASM using `prost_malloc`
2. The host writes the serialized `CodeGeneratorRequest` to that memory
3. The host calls `prost_execute` with the pointer and length
4. The plugin processes the request and stores the `CodeGeneratorResponse` internally
5. The host reads the response using `prost_get_output_ptr` and `prost_get_output_len`
6. The host calls `prost_clear_output` to free the internal buffer

## Features

- Embeds protoc-gen-prost as a ~600KB WASI WebAssembly binary
- Pure Go execution via wazero (no CGO required)
- Thread-safe with mutex protection
- Supports repeated executions without reloading

## Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/tetratelabs/wazero"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/pluginpb"
    prost "github.com/aperturerobotics/go-protoc-gen-prost"
)

func main() {
    ctx := context.Background()
    r := wazero.NewRuntime(ctx)
    defer r.Close(ctx)

    // Create the protoc-gen-prost instance
    p, err := prost.NewProtocGenProst(ctx, r)
    if err != nil {
        panic(err)
    }
    defer p.Close(ctx)

    // Create a CodeGeneratorRequest
    req := &pluginpb.CodeGeneratorRequest{
        FileToGenerate: []string{"example.proto"},
        ProtoFile: []*descriptorpb.FileDescriptorProto{
            // ... your proto file descriptors
        },
    }

    // Serialize the request
    input, err := proto.Marshal(req)
    if err != nil {
        panic(err)
    }

    // Execute the plugin
    output, err := p.Execute(ctx, input)
    if err != nil {
        panic(err)
    }

    // Parse the response
    resp := &pluginpb.CodeGeneratorResponse{}
    if err := proto.Unmarshal(output, resp); err != nil {
        panic(err)
    }

    // Process generated files
    for _, file := range resp.GetFile() {
        fmt.Printf("Generated: %s\n", file.GetName())
    }
}
```

## Updating the WASM Binary

To update to a new version of protoc-gen-prost:

```bash
./update-prost.bash
```

This script:
1. Fetches the latest release from `aperturerobotics/protoc-gen-prost`
2. Downloads the `protoc-gen-prost.wasm` artifact
3. Updates `version.go` with the new version info

## Building the WASM Binary

The WASM binary is built from [aperturerobotics/protoc-gen-prost](https://github.com/aperturerobotics/protoc-gen-prost):

```bash
# Clone the repository
git clone https://github.com/aperturerobotics/protoc-gen-prost.git
cd protoc-gen-prost

# Build for WASI
./build-wasi.sh

# Output: dist/protoc-gen-prost.wasm
```

### Build Requirements

- Rust toolchain with `wasm32-wasip1` target
- binaryen (for `wasm-opt`)
- wabt (for `wasm-strip`)

## Testing

```bash
go test -v ./...
```

## License

MIT

The embedded protoc-gen-prost WASM is covered by the [Apache 2.0 license](https://github.com/neoeinstein/protoc-gen-prost/blob/main/LICENSE).
