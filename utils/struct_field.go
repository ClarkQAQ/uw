package utils

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

var (
	ErrStructFieldValueIsNotPointer = errors.New("value must be a pointer")
	ErrStructFieldValueIsNotStruct  = errors.New("value must be a struct")
	ErrStructFieldFieldRepeated     = errors.New("field repeated")
	ErrStructFieldFieldNotExist     = errors.New("field not exist")
	ErrStructFieldCannotSet         = errors.New("cannot set field")
	ErrStructFieldTypeMismatch      = errors.New("type mismatch")
)

type StructField struct {
	mu    *sync.RWMutex
	cache map[string]reflect.Value
}

func NewStructField(value interface{}) (*StructField, error) {
	sf := &StructField{
		mu:    &sync.RWMutex{},
		cache: make(map[string]reflect.Value),
	}

	if e := sf.Reset(value); e != nil {
		return nil, e
	}

	return sf, nil
}

func (sf *StructField) Reset(value interface{}) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		return ErrStructFieldValueIsNotPointer
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return ErrStructFieldValueIsNotStruct
	}

	return sf.cacheFields(v, "")
}

func (sf *StructField) cacheFields(v reflect.Value, prefix string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		path := field.Name
		if prefix != "" {
			path = prefix + "." + field.Name
		}

		if _, ok := sf.cache[path]; ok {
			return fmt.Errorf("%w: %s", ErrStructFieldFieldRepeated, path)
		}

		vf := v.Field(i)
		sf.cache[path] = vf

		if e := sf.cacheFields(vf, path); e != nil {
			return e
		}
	}

	return nil
}

func (sf *StructField) Get(key string) (reflect.Value, bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	field, ok := sf.cache[key]
	if !ok {
		return reflect.Value{}, false
	}

	return field, true
}

func (sf *StructField) Set(key string, value interface{}) error {
	field, ok := sf.Get(key)
	if !ok {
		return fmt.Errorf("%w: %s", ErrStructFieldFieldNotExist, key)
	}

	if !field.CanSet() {
		return fmt.Errorf("%w: %s", ErrStructFieldCannotSet, key)
	}

	newValue := reflect.ValueOf(value)
	if newValue.Kind() != field.Kind() {
		return fmt.Errorf("%w: expected %s, got %s",
			ErrStructFieldTypeMismatch, field.Kind(), newValue.Kind())
	}

	field.Set(newValue)
	return nil
}

func (sf *StructField) Keys() []string {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	items := make([]string, 0, len(sf.cache))
	for item := range sf.cache {
		items = append(items, item)
	}

	sort.Strings(items)

	return items
}
