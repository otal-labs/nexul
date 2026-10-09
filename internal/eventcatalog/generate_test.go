package eventcatalog

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

const draft = "https://json-schema.org/draft/2020-12/schema"

var (
	marshalerType     = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

// generateSchemas returns each topic's schema text, generated from the payload types the domains declare: jsonschema-go
// infers the shape, the json tags say what is left out when empty, and the enum, minimum, type and deprecated tags
// add what reflection cannot see.
func generateSchemas() (map[string]string, error) {
	generated, err := generate(declared())
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(generated))
	for name, s := range generated {
		text, err := json.Marshal(s)
		if err != nil {
			return nil, fmt.Errorf("encode schema of %s: %w", name, err)
		}
		out[name] = string(text)
	}
	return out, nil
}

func generate(topics []eventbus.Topic) (map[string]*jsonschema.Schema, error) {
	g := generator{overrides: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[time.Time]():       {Type: "string", Format: "date-time"},
		reflect.TypeFor[json.RawMessage](): {},
	}}
	out := make(map[string]*jsonschema.Schema, len(topics))
	for _, t := range topics {
		s, err := g.schema(reflect.TypeOf(t.Payload))
		if err != nil {
			return nil, fmt.Errorf("schema of %s: %w", t.Name, err)
		}
		s.Schema = draft
		s.Description = t.Description
		prev, ok := out[t.Name]
		if !ok {
			out[t.Name] = s
			continue
		}
		if err := merge(prev, s); err != nil {
			return nil, fmt.Errorf("schema of %s: %w", t.Name, err)
		}
	}
	return out, nil
}

// generator turns payload types into schemas: jsonschema-go infers the shape, then shape fits it to the wire.
type generator struct {
	overrides map[reflect.Type]*jsonschema.Schema
}

func (g *generator) schema(t reflect.Type) (*jsonschema.Schema, error) {
	if err := g.wireShapes(t, map[reflect.Type]bool{}); err != nil {
		return nil, err
	}
	s, err := jsonschema.ForType(t, &jsonschema.ForOptions{TypeSchemas: g.overrides})
	if err != nil {
		return nil, err
	}
	g.shape(t, s)
	return s, nil
}

// wireShapes gives every type with its own JSON encoding the schema of what it encodes as, which reflection cannot see.
func (g *generator) wireShapes(t reflect.Type, seen map[reflect.Type]bool) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if seen[t] || g.overrides[t] != nil {
		return nil
	}
	seen[t] = true
	if implements(t, marshalerType) {
		shaper, ok := reflect.Zero(t).Interface().(eventbus.WireShaper)
		if !ok {
			return fmt.Errorf("%s has its own MarshalJSON; give it a WireShape method", t)
		}
		s, err := g.schema(reflect.TypeOf(shaper.WireShape()))
		if err != nil {
			return fmt.Errorf("wire shape of %s: %w", t, err)
		}
		g.overrides[t] = s
		return nil
	}
	if implements(t, textMarshalerType) {
		g.overrides[t] = &jsonschema.Schema{Type: "string"}
		return nil
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return g.wireShapes(t.Elem(), seen)
	case reflect.Struct:
		for _, f := range reflect.VisibleFields(t) {
			if !f.IsExported() {
				continue
			}
			if err := g.wireShapes(f.Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func implements(t, iface reflect.Type) bool {
	return t.Implements(iface) || reflect.PointerTo(t).Implements(iface)
}

// shape fits an inferred schema to what encoding/json writes and the contract promises.
func (g *generator) shape(t reflect.Type, s *jsonschema.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if s == nil || g.overrides[t] != nil {
		return
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		g.shape(t.Elem(), s.Items)
	case reflect.Map:
		g.shape(t.Elem(), s.AdditionalProperties)
	case reflect.Struct:
		g.shapeStruct(t, s)
	}
}

func (g *generator) shapeStruct(t reflect.Type, s *jsonschema.Schema) {
	// A consumer must accept fields added later (ADR 0044), so objects stay open.
	s.AdditionalProperties = nil
	for _, f := range reflect.VisibleFields(t) {
		name, omitted := jsonField(f)
		p := s.Properties[name]
		if p == nil {
			continue
		}
		if omitted {
			dropNull(p)
		}
		if typ := f.Tag.Get("type"); typ != "" {
			p.Type, p.Types = typ, nil
		}
		if enum := f.Tag.Get("enum"); enum != "" {
			values := p
			if p.Items != nil {
				values = p.Items
			}
			for _, v := range strings.Split(enum, ",") {
				values.Enum = append(values.Enum, v)
			}
		}
		if minimum, err := strconv.ParseFloat(f.Tag.Get("minimum"), 64); err == nil {
			p.Minimum = &minimum
		}
		p.Deprecated = f.Tag.Get("deprecated") == "true"
		g.shape(f.Type, p)
	}
}

// jsonField returns a field's JSON name, empty when encoding/json skips it, and whether a zero value is left out.
func jsonField(f reflect.StructField) (string, bool) {
	if f.Anonymous || !f.IsExported() {
		return "", false
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	name, opts, _ := strings.Cut(tag, ",")
	if name == "" {
		name = f.Name
	}
	settings := strings.Split(opts, ",")
	return name, slices.Contains(settings, "omitempty") || slices.Contains(settings, "omitzero")
}

// dropNull removes null from a left-out-when-empty field's types: encoding/json omits a nil value, never writes null.
func dropNull(s *jsonschema.Schema) {
	s.Types = slices.DeleteFunc(s.Types, func(t string) bool { return t == "null" })
	if len(s.Types) == 1 {
		s.Type, s.Types = s.Types[0], nil
	}
}

// merge folds a second payload type of one topic into the first: either may arrive, so only what both always
// carry stays required, and a property both carry must mean the same in each.
func merge(into, other *jsonschema.Schema) error {
	if other.Description != "" && into.Description != "" && other.Description != into.Description {
		return fmt.Errorf("two descriptions: %q and %q", into.Description, other.Description)
	}
	if into.Description == "" {
		into.Description = other.Description
	}
	for _, name := range other.PropertyOrder {
		p := other.Properties[name]
		existing, ok := into.Properties[name]
		if !ok {
			into.Properties[name] = p
			into.PropertyOrder = append(into.PropertyOrder, name)
			continue
		}
		a, errA := json.Marshal(existing)
		b, errB := json.Marshal(p)
		if errA != nil || errB != nil || string(a) != string(b) {
			return fmt.Errorf("property %s differs between payload types: %s and %s", name, a, b)
		}
	}
	into.Required = slices.DeleteFunc(into.Required, func(name string) bool { return !slices.Contains(other.Required, name) })
	return nil
}
