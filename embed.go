// Package prost provides a Go wrapper for running protoc-gen-prost via WASI/wazero.
package prost

import _ "embed"

// ProtocGenProstWASM contains the binary contents of the protoc-gen-prost WASI build.
//
// This is a WASM binary that exports functions for executing the Prost protobuf
// code generator. The module uses the standard WASI preview1 interface.
//
//go:embed protoc-gen-prost.wasm
var ProtocGenProstWASM []byte

// ProtocGenProstWASMFilename is the filename for ProtocGenProstWASM.
const ProtocGenProstWASMFilename = "protoc-gen-prost.wasm"

// Prost plugin exports
const (
	// ExportProstExecute executes the prost plugin.
	// Signature: prost_execute(input_ptr: i32, input_len: i32) -> i32 (output_len)
	ExportProstExecute = "prost_execute"

	// ExportProstGetOutputPtr returns the pointer to the output buffer.
	// Signature: prost_get_output_ptr() -> i32 (ptr)
	ExportProstGetOutputPtr = "prost_get_output_ptr"

	// ExportProstGetOutputLen returns the length of the output buffer.
	// Signature: prost_get_output_len() -> i32 (len)
	ExportProstGetOutputLen = "prost_get_output_len"

	// ExportProstClearOutput clears the output buffer.
	// Signature: prost_clear_output() -> void
	ExportProstClearOutput = "prost_clear_output"
)

// Memory management exports
const (
	// ExportProstMalloc allocates memory in WASM linear memory.
	// Signature: prost_malloc(size: i32) -> i32 (pointer)
	ExportProstMalloc = "prost_malloc"

	// ExportProstFree frees memory in WASM linear memory.
	// Signature: prost_free(ptr: i32, size: i32) -> void
	ExportProstFree = "prost_free"
)
