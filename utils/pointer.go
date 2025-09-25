package utils

func Ptr[T any](v T) *T {
	return &v
}

func MapPtr[TIn any, TOut any](v *TIn, mapping func(TIn) TOut) *TOut {
	if v == nil {
		return nil
	}

	return Ptr(mapping(*v))
}

func NilIfZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
