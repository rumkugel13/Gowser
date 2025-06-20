package browser

type ProtectedMarker interface {
    Mark()
}

type ProtectedField[T any] struct {
	value         T
	dirty         bool
	invalidations map[ProtectedMarker]bool
}

func NewProtectedField[T any]() *ProtectedField[T] {
	return &ProtectedField[T]{dirty: true, invalidations: make(map[ProtectedMarker]bool)}
}

func (f *ProtectedField[T]) Mark() {
	if f.dirty {
		return
	}
	f.dirty = true
}

func (f *ProtectedField[T]) Get() T {
	if f.dirty {
		panic("protected field is dirty")
	}
	return f.value
}

func (f *ProtectedField[T]) Set(value T) {
	f.Notify()
	f.value = value
	f.dirty = false
}

func (f *ProtectedField[T]) Notify() {
	for field := range f.invalidations {
		field.Mark()
	}
}

func (f *ProtectedField[T]) Read(notify ProtectedMarker) T {
	f.invalidations[notify] = true
	return f.Get()
}

func (f *ProtectedField[T]) Copy(field *ProtectedField[T]) {
	f.Set(field.Read(f))
}
