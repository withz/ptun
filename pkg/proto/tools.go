package proto

import (
	"fmt"
	"reflect"
)

func typeOf(d any) reflect.Type {
	t := reflect.TypeOf(d)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func CheckType[T any](data any) error {
	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	v := reflect.TypeFor[T]()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if t != v {
		return fmt.Errorf("require type %s but get type %s", v.Name(), t.Name())
	}
	return nil
}
