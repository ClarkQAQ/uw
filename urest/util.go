package urest

import "reflect"

type reflectValue struct {
	Value  reflect.Value
	Method reflect.Method
}

func structReflectValue(val any) ([]*reflectValue, error) {
	v := reflect.ValueOf(val)
	t := v.Type()

	if v.Kind() != reflect.Pointer && v.Kind() != reflect.Struct {
		return nil, ErrValueNotStructOrPointer
	}

	value := []*reflectValue{}

	for i := 0; i < v.NumMethod(); i++ {
		vt := v.Method(i)
		mt := t.Method(i)

		if !vt.IsValid() || !vt.CanInterface() {
			continue
		}

		value = append(value, &reflectValue{vt, mt})
	}

	return value, nil
}
