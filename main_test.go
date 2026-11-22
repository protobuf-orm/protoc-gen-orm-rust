package main

import (
	"github.com/protobuf-orm/protobuf-orm/ormpb"
	"google.golang.org/protobuf/proto"
	d "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
	"strings"
	"testing"
)

// A consuming application can have other services named Get. Only its ORM
// entity contracts should acquire constructors.
func TestAnnotatedService(t *testing.T) {
	f := &d.FileDescriptorProto{Name: proto.String("lookup.proto"), Syntax: proto.String("proto3"), Package: proto.String("type.v1")}
	entity := &d.DescriptorProto{Name: proto.String("Row"), Options: &d.MessageOptions{}}
	ref := &d.DescriptorProto{Name: proto.String("RowRef"), OneofDecl: []*d.OneofDescriptorProto{{Name: proto.String("key")}}, Field: []*d.FieldDescriptorProto{
		{Name: proto.String("id"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_BYTES.Enum(), OneofIndex: proto.Int32(0)},
	}}
	req := &d.DescriptorProto{Name: proto.String("RowGetRequest"), Field: []*d.FieldDescriptorProto{
		{Name: proto.String("ref"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".type.v1.RowRef")},
	}}
	f.MessageType = []*d.DescriptorProto{entity, ref, req}
	f.Service = []*d.ServiceDescriptorProto{{Name: proto.String("RowService"), Method: []*d.MethodDescriptorProto{
		{Name: proto.String("Get"), InputType: proto.String(".type.v1.RowGetRequest"), OutputType: proto.String(".type.v1.Row")},
	}}}
	input := &pluginpb.CodeGeneratorRequest{FileToGenerate: []string{f.GetName()}, ProtoFile: []*d.FileDescriptorProto{f}}
	output := new(pluginpb.CodeGeneratorResponse)
	if err := generate(input, output); err != nil || len(output.File) != 0 {
		t.Fatalf("ordinary service: files=%d err=%v", len(output.File), err)
	}
	proto.SetExtension(entity.Options, ormpb.E_Message, &ormpb.MessageOptions{})
	if err := generate(input, output); err != nil || len(output.File) != 1 {
		t.Fatalf("entity service: files=%d err=%v", len(output.File), err)
	}
	for _, want := range []string{"pub fn by_id(value: Vec<u8>)", "crate::proto::r#type::v1::RowRef", "pub fn register_row_service"} {
		if !strings.Contains(output.File[0].GetContent(), want) {
			t.Errorf("missing %q", want)
		}
	}
	ref.OneofDecl = nil
	ref.Field[0].OneofIndex = nil
	if err := generate(input, new(pluginpb.CodeGeneratorResponse)); err == nil {
		t.Error("expected missing key oneof to be rejected")
	}
}
