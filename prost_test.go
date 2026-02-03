package prost

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestProtocGenProst_LoadModule(t *testing.T) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	p, err := NewProtocGenProst(ctx, r)
	if err != nil {
		t.Fatalf("NewProtocGenProst failed: %v", err)
	}
	defer p.Close(ctx)

	// Module should be loaded successfully
	if p.mod == nil {
		t.Fatal("module is nil")
	}
}

func TestProtocGenProst_ExecuteMinimalRequest(t *testing.T) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	p, err := NewProtocGenProst(ctx, r)
	if err != nil {
		t.Fatalf("NewProtocGenProst failed: %v", err)
	}
	defer p.Close(ctx)

	// Create a minimal CodeGeneratorRequest with a simple proto file
	protoFileName := "test.proto"
	packageName := "test"
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    &protoFileName,
				Package: &packageName,
				Syntax:  proto.String("proto3"),
			},
		},
	}

	input, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	output, err := p.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Parse the response
	resp := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(output, resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// The response should be valid
	t.Logf("Response: error=%q, files=%d", resp.GetError(), len(resp.GetFile()))

	// Should have generated at least one file
	if len(resp.GetFile()) == 0 && resp.GetError() == "" {
		t.Fatal("expected at least one generated file or an error")
	}
}

func TestProtocGenProst_RepeatedExecutions(t *testing.T) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	p, err := NewProtocGenProst(ctx, r)
	if err != nil {
		t.Fatalf("NewProtocGenProst failed: %v", err)
	}
	defer p.Close(ctx)

	// Create a minimal CodeGeneratorRequest
	protoFileName := "test.proto"
	packageName := "test"
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    &protoFileName,
				Package: &packageName,
				Syntax:  proto.String("proto3"),
			},
		},
	}

	input, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Execute multiple times to check for memory leaks
	for i := 0; i < 10; i++ {
		output, err := p.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}

		resp := &pluginpb.CodeGeneratorResponse{}
		if err := proto.Unmarshal(output, resp); err != nil {
			t.Fatalf("failed to unmarshal response %d: %v", i, err)
		}
	}
}

func TestProtocGenProst_CompiledModule(t *testing.T) {
	ctx := context.Background()

	// Test that we can compile and use the module multiple times with the same runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Compile once
	compiled, err := CompileProtocGenProst(ctx, r)
	if err != nil {
		t.Fatalf("CompileProtocGenProst failed: %v", err)
	}

	// Use the compiled module with the same runtime
	p, err := NewProtocGenProstWithModule(ctx, r, compiled)
	if err != nil {
		t.Fatalf("NewProtocGenProstWithModule failed: %v", err)
	}
	defer p.Close(ctx)

	protoFileName := "test.proto"
	packageName := "test"
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    &protoFileName,
				Package: &packageName,
				Syntax:  proto.String("proto3"),
			},
		},
	}
	input, _ := proto.Marshal(req)

	_, err = p.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}
