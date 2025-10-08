package utils

func MapSlice[T any, R any](input []T, fn func(T) R) []R {
	result := make([]R, len(input))
	for i, v := range input {
		result[i] = fn(v)
	}
	return result
}

func EmptyIfNil[T any](input []T) []T {
	if input == nil {
		return make([]T, 0)
	}
	return input
}
