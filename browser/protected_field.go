package browser

import (
	"fmt"
	"reflect"
	"slices"
)

type ProtectedMarker interface {
	Mark()
	FrozenDependencies() bool
	AddInvalidation(ProtectedMarker)
}

type ProtectedField[T any] struct {
	Value               T
	Dirty               bool
	invalidations       map[ProtectedMarker]bool
	obj                 any
	name                string
	parent              any
	frozen_dependencies bool
}

func NewProtectedField[T any](obj any, name string, parent any, dependencies *[]ProtectedMarker) *ProtectedField[T] {
	field := &ProtectedField[T]{
		Dirty:         true,
		invalidations: make(map[ProtectedMarker]bool),
		obj:           obj,
		name:          name,
		parent:        parent,
	}
	field.frozen_dependencies = dependencies != nil
	if dependencies != nil {
		for _, dependency := range *dependencies {
			dependency.AddInvalidation(field)
		}
	} else if !(slices.Contains([]string{"height", "ascent", "descent", "children"}, name) || CSS_PROPERTIES[name] != "") {
		panic("invalid dependencies")
	}
	return field
}

func (f *ProtectedField[T]) Mark() {
	if f.Dirty {
		return
	}
	f.Dirty = true
	f.SetAncestorDirtyBits()
}

func (f *ProtectedField[T]) Get() T {
	if f.Dirty {
		panic("protected field is dirty")
	}
	return f.Value
}

func (f *ProtectedField[T]) Set(value T) {
	// if !reflect.DeepEqual(f.Value, *new(T)) {
	// 	fmt.Println("Change", f.String())
	// }
	if !reflect.DeepEqual(value, f.Value) {
		f.Notify()
	}
	f.Value = value
	f.Dirty = false
}

func (f *ProtectedField[T]) Notify() {
	for field := range f.invalidations {
		field.Mark()
	}
	f.SetAncestorDirtyBits()
}

func (f *ProtectedField[T]) Read(notify ProtectedMarker) T {
	if notify.FrozenDependencies() {
		if _, found := f.invalidations[notify]; !found {
			panic("notify not found")
		}
	} else {
		f.invalidations[notify] = true
	}
	return f.Get()
}

func (f *ProtectedField[T]) Copy(field *ProtectedField[T]) {
	f.Set(field.Read(f))
}

func (f *ProtectedField[T]) String() string {
	switch t := f.obj.(type) {
	case *LayoutNode:
		return fmt.Sprintf("ProtectedField(%v, %s)", t.Node, f.name)
	case ElementToken:
		return fmt.Sprintf("ProtectedField(%v, %s)", t, f.name)
	case TextToken:
		return fmt.Sprintf("ProtectedField(%v, %s)", t, f.name)
	}
	// if f.obj.Node != nil {
	// 	return fmt.Sprintf("ProtectedField(%v, %s)", f.obj.Node, f.name)
	// }
	return fmt.Sprintf("ProtectedField(%v, %s)", f.obj, f.name)
}

func (f *ProtectedField[T]) SetAncestorDirtyBits() {
	switch t := f.parent.(type) {
	case *LayoutNode:
		parent := t
		for parent != nil && !parent.has_dirty_descendants {
			parent.has_dirty_descendants = true
			parent = parent.Parent
		}
	}
}

func (f *ProtectedField[T]) FrozenDependencies() bool {
	return f.frozen_dependencies
}

func (f *ProtectedField[T]) SetDependencies(dependencies []ProtectedMarker) {
	if !(slices.Contains([]string{"height", "ascent", "descent"}, f.name) || CSS_PROPERTIES[f.name] != "") {
		panic("invalid dependencies")
	}
	if !(f.name == "height" || !f.frozen_dependencies) {
		panic("invalid frozen dependencies")
	}

	for _, dependency := range dependencies {
		dependency.AddInvalidation(f)
	}
	f.frozen_dependencies = true
}

func (f *ProtectedField[T]) AddInvalidation(field ProtectedMarker) {
	f.invalidations[field] = true
}
