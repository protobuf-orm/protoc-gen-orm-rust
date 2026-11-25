package main

import (
	"strings"
	"testing"

	"github.com/protobuf-orm/protobuf-orm/ormpb"
	"google.golang.org/protobuf/proto"
	d "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// An edge's on_delete reaches the model, so that a storage generator can write
// the foreign key's action, and an edge that says nothing says nothing there.
func TestStorageModelOnDelete(t *testing.T) {
	key := func() *d.FieldOptions {
		o := &d.FieldOptions{}
		proto.SetExtension(o, ormpb.E_Field, ormpb.FieldOptions_builder{Key: proto.Bool(true)}.Build())
		return o
	}
	edge := func(v ormpb.OnDelete) *d.FieldOptions {
		o := &d.FieldOptions{}
		proto.SetExtension(o, ormpb.E_Edge, ormpb.EdgeOptions_builder{Nullable: proto.Bool(true), OnDelete: v.Enum()}.Build())
		return o
	}
	entity := func() *d.MessageOptions {
		o := &d.MessageOptions{}
		proto.SetExtension(o, ormpb.E_Message, &ormpb.MessageOptions{})
		return o
	}
	msg := d.FieldDescriptorProto_TYPE_MESSAGE.Enum()
	f := &d.FileDescriptorProto{
		Name:    proto.String("row.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("row.v1"),
		MessageType: []*d.DescriptorProto{
			{Name: proto.String("Owner"), Options: entity(), Field: []*d.FieldDescriptorProto{
				{Name: proto.String("id"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_STRING.Enum(), Options: key()},
			}},
			{Name: proto.String("Row"), Options: entity(), Field: []*d.FieldDescriptorProto{
				{Name: proto.String("id"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_STRING.Enum(), Options: key()},
				{Name: proto.String("restricted"), Number: proto.Int32(2), Type: msg, TypeName: proto.String(".row.v1.Owner"), Options: edge(ormpb.OnDelete_ON_DELETE_RESTRICT)},
				{Name: proto.String("cleared"), Number: proto.Int32(3), Type: msg, TypeName: proto.String(".row.v1.Owner"), Options: edge(ormpb.OnDelete_ON_DELETE_SET_NULL)},
				{Name: proto.String("unsaid"), Number: proto.Int32(4), Type: msg, TypeName: proto.String(".row.v1.Owner"), Options: edge(ormpb.OnDelete_ON_DELETE_UNSPECIFIED)},
			}},
		},
	}
	input := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{f.GetName()},
		Parameter:      proto.String("storage_model"),
		ProtoFile:      []*d.FileDescriptorProto{f},
	}
	output := new(pluginpb.CodeGeneratorResponse)
	if err := generate(input, output); err != nil {
		t.Fatal(err)
	}
	if len(output.File) != 1 {
		t.Fatalf("files: %d", len(output.File))
	}
	got := output.File[0].GetContent()
	for _, want := range []string{`"on_delete": "RESTRICT"`, `"on_delete": "SET_NULL"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s:\n%s", want, got)
		}
	}
	if n := strings.Count(got, `"on_delete"`); n != 2 {
		t.Errorf("on_delete appears %d times, and an edge that says nothing says nothing:\n%s", n, got)
	}
}
