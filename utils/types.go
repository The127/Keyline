package utils

import "reflect"

func TypeOf[T any]() reflect.Type {
	var tp *T
	return reflect.TypeOf(tp).Elem()
}
