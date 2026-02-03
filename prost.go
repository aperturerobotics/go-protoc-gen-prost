package prost

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// ProtocGenProst wraps a protoc-gen-prost WASI module providing a high-level API
// for executing the Prost protobuf code generator.
type ProtocGenProst struct {
	runtime wazero.Runtime
	mod     api.Module

	// Memory management
	malloc api.Function
	free   api.Function

	// Prost plugin functions
	prostExecute      api.Function
	prostGetOutputPtr api.Function
	prostGetOutputLen api.Function
	prostClearOutput  api.Function

	// Mutex for thread-safe Execute calls (WASI is single-threaded)
	mu sync.Mutex
}

// CompileProtocGenProst compiles the embedded protoc-gen-prost WASM module.
// The compiled module can be reused across multiple ProtocGenProst instances.
func CompileProtocGenProst(ctx context.Context, r wazero.Runtime) (wazero.CompiledModule, error) {
	return r.CompileModule(ctx, ProtocGenProstWASM)
}

// NewProtocGenProst creates a new ProtocGenProst instance using the embedded WASM.
// Call Close() when done to release resources.
func NewProtocGenProst(ctx context.Context, r wazero.Runtime) (*ProtocGenProst, error) {
	compiled, err := CompileProtocGenProst(ctx, r)
	if err != nil {
		return nil, err
	}
	return NewProtocGenProstWithModule(ctx, r, compiled)
}

// NewProtocGenProstWithModule creates a new ProtocGenProst instance using a pre-compiled module.
func NewProtocGenProstWithModule(ctx context.Context, r wazero.Runtime, compiled wazero.CompiledModule) (*ProtocGenProst, error) {
	// Instantiate WASI
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	// Build module config
	modCfg := wazero.NewModuleConfig().WithName(ProtocGenProstWASMFilename)

	// Instantiate the module
	mod, err := r.InstantiateModule(ctx, compiled, modCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	// Call _initialize if present (reactor mode)
	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			mod.Close(ctx)
			return nil, fmt.Errorf("_initialize failed: %w", err)
		}
	}

	p := &ProtocGenProst{
		runtime:           r,
		mod:               mod,
		malloc:            mod.ExportedFunction(ExportProstMalloc),
		free:              mod.ExportedFunction(ExportProstFree),
		prostExecute:      mod.ExportedFunction(ExportProstExecute),
		prostGetOutputPtr: mod.ExportedFunction(ExportProstGetOutputPtr),
		prostGetOutputLen: mod.ExportedFunction(ExportProstGetOutputLen),
		prostClearOutput:  mod.ExportedFunction(ExportProstClearOutput),
	}

	// Validate required exports
	if p.malloc == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstMalloc)
	}
	if p.free == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstFree)
	}
	if p.prostExecute == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstExecute)
	}
	if p.prostGetOutputPtr == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstGetOutputPtr)
	}
	if p.prostGetOutputLen == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstGetOutputLen)
	}
	if p.prostClearOutput == nil {
		mod.Close(ctx)
		return nil, errors.New("missing export: " + ExportProstClearOutput)
	}

	return p, nil
}

// Execute runs the protoc-gen-prost plugin with the given CodeGeneratorRequest.
// The input should be a serialized google.protobuf.compiler.CodeGeneratorRequest.
// Returns a serialized google.protobuf.compiler.CodeGeneratorResponse.
func (p *ProtocGenProst) Execute(ctx context.Context, input []byte) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Allocate memory for input
	inputPtr, err := p.allocBytes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate input: %w", err)
	}
	defer p.freePtr(ctx, inputPtr, uint32(len(input)))

	// Call prost_execute
	results, err := p.prostExecute.Call(ctx, uint64(inputPtr), uint64(len(input)))
	if err != nil {
		return nil, fmt.Errorf("prost_execute failed: %w", err)
	}
	outputLen := uint32(results[0])

	// Get output pointer
	results, err = p.prostGetOutputPtr.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("prost_get_output_ptr failed: %w", err)
	}
	outputPtr := uint32(results[0])

	// Read output from WASM memory
	output, ok := p.mod.Memory().Read(outputPtr, outputLen)
	if !ok {
		return nil, errors.New("failed to read output from memory")
	}

	// Make a copy since we're about to clear the buffer
	result := make([]byte, len(output))
	copy(result, output)

	// Clear output buffer
	if _, err := p.prostClearOutput.Call(ctx); err != nil {
		return nil, fmt.Errorf("prost_clear_output failed: %w", err)
	}

	return result, nil
}

// Close releases resources associated with the ProtocGenProst instance.
func (p *ProtocGenProst) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mod != nil {
		return p.mod.Close(ctx)
	}
	return nil
}

// Memory helpers

func (p *ProtocGenProst) allocBytes(ctx context.Context, data []byte) (uint32, error) {
	if len(data) == 0 {
		return 0, nil
	}
	results, err := p.malloc.Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, err
	}
	ptr := uint32(results[0])
	if ptr == 0 {
		return 0, errors.New("malloc returned null")
	}
	if !p.mod.Memory().Write(ptr, data) {
		p.free.Call(ctx, uint64(ptr), uint64(len(data)))
		return 0, errors.New("failed to write to memory")
	}
	return ptr, nil
}

func (p *ProtocGenProst) freePtr(ctx context.Context, ptr, size uint32) {
	if ptr != 0 {
		p.free.Call(ctx, uint64(ptr), uint64(size))
	}
}
