// Package jsonx is the JSON encoder every adapter and event payload shares, so clients always get arrays.
package jsonx

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"
	"unsafe"
)

// Marshal is json.Marshal with every nil slice written as []. A byte slice keeps null (an empty json.RawMessage
// fails to encode) and a nil slice under an omitzero field stays omitted. v is never written to: handlers pass
// values other goroutines still hold, so only the path down to a nil slice is copied.
func Marshal(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || !planOf(rv.Type()).walk {
		return json.Marshal(v)
	}
	fixed, changed := fill(rv)
	if !changed {
		return json.Marshal(v)
	}
	return json.Marshal(fixed.Interface())
}

// plan is what a type needs from the walk, built once per type.
type plan struct {
	walk   bool          // a value of this type can hold a nil slice the encoder would write
	empty  reflect.Value // slices: the shared empty value a nil one becomes
	fields []field       // structs: the fields worth walking
	unsafe bool          // structs: a promoted field sits in an unexported embedded struct, reached only through a copy
}

type field struct {
	index    int
	promoted bool // an unexported embedded struct whose exported fields JSON promotes
}

var plans sync.Map // reflect.Type -> *plan

func planOf(t reflect.Type) *plan {
	if p, ok := plans.Load(t); ok {
		return p.(*plan)
	}
	return build(t, map[reflect.Type]bool{})
}

// build treats a type already being built as walkable, so a recursive type is walked rather than guessed.
func build(t reflect.Type, building map[reflect.Type]bool) *plan {
	if p, ok := plans.Load(t); ok {
		return p.(*plan)
	}
	if building[t] {
		return &plan{walk: true}
	}
	building[t] = true
	p := &plan{}
	switch t.Kind() {
	case reflect.Slice:
		p.walk = t.Elem().Kind() != reflect.Uint8
		p.empty = reflect.MakeSlice(t, 0, 0)
	case reflect.Pointer, reflect.Map:
		p.walk = build(t.Elem(), building).walk
	case reflect.Interface:
		p.walk = true
	case reflect.Struct:
		p.fields, p.unsafe = structFields(t, building)
		p.walk = len(p.fields) > 0
	}
	delete(building, t)
	actual, _ := plans.LoadOrStore(t, p)
	return actual.(*plan)
}

func structFields(t reflect.Type, building map[reflect.Type]bool) ([]field, bool) {
	var fields []field
	promotedAny := false
	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		_, opts, _ := strings.Cut(tag, ",")
		if tag == "-" || slices.Contains(strings.Split(opts, ","), "omitzero") {
			continue
		}
		promoted := f.Anonymous && !f.IsExported() && f.Type.Kind() == reflect.Struct
		if !f.IsExported() && !promoted {
			continue
		}
		if !build(f.Type, building).walk {
			continue
		}
		fields = append(fields, field{index: i, promoted: promoted})
		promotedAny = promotedAny || promoted
	}
	return fields, promotedAny
}

// fill returns v with its nil slices made empty, and whether anything changed; an unchanged v is returned as is.
func fill(v reflect.Value) (reflect.Value, bool) {
	switch v.Kind() {
	case reflect.Slice:
		return fillSlice(v)
	case reflect.Struct:
		return fillStruct(v)
	case reflect.Pointer:
		return fillPointer(v)
	case reflect.Map:
		return fillMap(v)
	case reflect.Interface:
		return fillInterface(v)
	default:
		return v, false
	}
}

func fillSlice(v reflect.Value) (reflect.Value, bool) {
	p := planOf(v.Type())
	if !p.walk {
		return v, false
	}
	if v.IsNil() {
		return p.empty, true
	}
	if !planOf(v.Type().Elem()).walk {
		return v, false
	}
	var out reflect.Value
	for i := range v.Len() {
		e, changed := fill(v.Index(i))
		if !changed {
			continue
		}
		if !out.IsValid() {
			out = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
			reflect.Copy(out, v)
		}
		out.Index(i).Set(e)
	}
	if !out.IsValid() {
		return v, false
	}
	return out, true
}

func fillStruct(v reflect.Value) (reflect.Value, bool) {
	p := planOf(v.Type())
	if !p.walk {
		return v, false
	}
	src := v
	if p.unsafe {
		src = copyOf(v)
	}
	var out reflect.Value
	for _, f := range p.fields {
		nv, changed := fill(fieldOf(src, f))
		if !changed {
			continue
		}
		if !out.IsValid() {
			out = src
			if !p.unsafe {
				out = copyOf(v)
			}
		}
		fieldOf(out, f).Set(nv)
	}
	if !out.IsValid() {
		return v, false
	}
	return out, true
}

// fieldOf reads a struct field; a promoted one is reached through its address, since reflect refuses unexported
// fields and the struct is always the walk's own copy.
func fieldOf(s reflect.Value, f field) reflect.Value {
	fv := s.Field(f.index)
	if !f.promoted {
		return fv
	}
	return reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
}

func copyOf(v reflect.Value) reflect.Value {
	c := reflect.New(v.Type()).Elem()
	c.Set(v)
	return c
}

func fillPointer(v reflect.Value) (reflect.Value, bool) {
	if v.IsNil() {
		return v, false
	}
	e, changed := fill(v.Elem())
	if !changed {
		return v, false
	}
	if e.CanAddr() {
		return e.Addr(), true // a changed struct is already the walk's own copy
	}
	return copyOf(e).Addr(), true
}

func fillMap(v reflect.Value) (reflect.Value, bool) {
	if v.IsNil() || !planOf(v.Type().Elem()).walk {
		return v, false
	}
	var keys, vals []reflect.Value
	for it := v.MapRange(); it.Next(); {
		nv, changed := fill(it.Value())
		if changed {
			keys, vals = append(keys, it.Key()), append(vals, nv)
		}
	}
	if keys == nil {
		return v, false
	}
	out := reflect.MakeMapWithSize(v.Type(), v.Len())
	for it := v.MapRange(); it.Next(); {
		out.SetMapIndex(it.Key(), it.Value())
	}
	for i, k := range keys {
		out.SetMapIndex(k, vals[i])
	}
	return out, true
}

func fillInterface(v reflect.Value) (reflect.Value, bool) {
	if v.IsNil() {
		return v, false
	}
	e, changed := fill(v.Elem())
	if !changed {
		return v, false
	}
	out := reflect.New(v.Type()).Elem()
	out.Set(e)
	return out, true
}
