package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/protobuf-orm/protobuf-orm/graph"
	"github.com/protobuf-orm/protobuf-orm/ormpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/pluginpb"
)

// The storage planner consumes resolved ORM properties. In particular, presence
// on a message edge means something different from presence on a scalar field.
type model struct {
	Version  int           `json:"version"`
	Entities []modelEntity `json:"entities"`
}
type modelEntity struct {
	Name    string       `json:"name"`
	Key     string       `json:"key"`
	Fields  []modelField `json:"fields"`
	Indexes []modelIndex `json:"indexes"`
}
type modelField struct {
	Name       string `json:"name"`
	JSON       string `json:"json"`
	Number     int32  `json:"number"`
	Kind       string `json:"kind"`
	Collection string `json:"collection,omitempty"`
	Target     string `json:"target,omitempty"`
	Inverse    string `json:"inverse,omitempty"`
	// OnDelete is what becomes of this row when the row an edge names is
	// deleted: RESTRICT, CASCADE or SET_NULL, and absent where the schema
	// does not say, which leaves it to the storage generator's own default.
	OnDelete  string  `json:"on_delete,omitempty"`
	Nullable  bool    `json:"nullable,omitempty"`
	Immutable bool    `json:"immutable,omitempty"`
	Unique    bool    `json:"unique,omitempty"`
	Default   *string `json:"default,omitempty"`
	Version   bool    `json:"version,omitempty"`
	Erased    bool    `json:"erased,omitempty"`
}
type modelIndex struct {
	Name       string   `json:"name"`
	Fields     []string `json:"fields"`
	Unique     bool     `json:"unique,omitempty"`
	ActiveOnly bool     `json:"active_only,omitempty"`
	Hidden     bool     `json:"hidden,omitempty"`
}

func storageModel(files *protoregistry.Files, names []string, resp *pluginpb.CodeGeneratorResponse) error {
	g := graph.NewGraph()
	for _, name := range names {
		file, err := files.FindFileByPath(name)
		if err != nil {
			return err
		}
		if err := graph.Parse(context.Background(), g, file); err != nil {
			return err
		}
	}
	m := model{Version: 1, Entities: []modelEntity{}}
	for _, ent := range g.Entities {
		if ent.Key() == nil {
			return fmt.Errorf("%s: missing ORM primary key", ent.FullName())
		}
		e := modelEntity{Name: string(ent.FullName()), Key: ent.Key().Name(), Indexes: []modelIndex{}}
		for p := range ent.Props() {
			d := p.Descriptor()
			field := modelField{Name: p.Name(), JSON: d.JSONName(), Number: int32(p.Number()), Kind: strings.TrimPrefix(p.Type().String(), "TYPE_"), Nullable: p.IsNullable(), Immutable: p.IsImmutable(), Unique: p.IsUnique()}
			if d.IsMap() {
				field.Collection = "map"
			} else if d.IsList() {
				field.Collection = "list"
			}
			switch prop := p.(type) {
			case graph.Edge:
				field.Target = string(prop.Target().FullName())
				if from := prop.Inverse(); from != nil {
					field.Inverse = from.Name()
				}
				if v := prop.OnDelete(); v != ormpb.OnDelete_ON_DELETE_UNSPECIFIED {
					field.OnDelete = strings.TrimPrefix(v.String(), "ON_DELETE_")
				}
				o := proto.GetExtension(d.Options(), ormpb.E_Edge).(*ormpb.EdgeOptions)
				if o.HasDefault() {
					field.Default = proto.String(o.GetDefault())
				}
			case graph.Field:
				field.Erased, field.Version = prop.IsErased(), prop.IsVersion()
				o := proto.GetExtension(d.Options(), ormpb.E_Field).(*ormpb.FieldOptions)
				if o.HasDefault() {
					field.Default = proto.String(o.GetDefault())
				}
			}
			if field.Name == e.Key {
				field.Nullable = false
			}
			e.Fields = append(e.Fields, field)
		}
		sort.Slice(e.Fields, func(i, j int) bool { return e.Fields[i].Number < e.Fields[j].Number })
		for index := range ent.Indexes() {
			idx := modelIndex{Name: index.Name(), Unique: index.IsUnique(), ActiveOnly: index.ExcludesErased(), Hidden: index.IsHidden()}
			for prop := range index.Props() {
				idx.Fields = append(idx.Fields, prop.Name())
			}
			e.Indexes = append(e.Indexes, idx)
		}
		slices.SortFunc(e.Indexes, func(a, b modelIndex) int { return strings.Compare(a.Name, b.Name) })
		m.Entities = append(m.Entities, e)
	}
	slices.SortFunc(m.Entities, func(a, b modelEntity) int { return strings.Compare(a.Name, b.Name) })
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{Name: proto.String("model.json"), Content: proto.String(string(bytes) + "\n")})
	return nil
}
