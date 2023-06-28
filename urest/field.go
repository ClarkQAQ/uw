package urest

import (
	"errors"
	"reflect"
	"strings"

	"uw/pkg/tagparser"
)

var ErrUnsupportedArrayType = errors.New("unsupported allow array type")

type FieldType uint8

const (
	FieldTypeInvalid FieldType = iota
	FieldTypeString
	FieldTypeNumber
	FieldTypeBool
	FieldTypeFile
	FieldTypeObject
	FieldTypeArray
	FieldTypeAny
)

type HandlerField struct {
	Request  []*Field `json:"request"`
	Response []*Field `json:"response"`
}

type Field struct {
	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	Type     FieldType `json:"type"`
	Key      string    `json:"key"`
	Enum     []string  `json:"enum"`
	Default  string    `json:"default"`
	Children []*Field  `json:"children"`
}

func (ft FieldType) String() string {
	switch ft {
	case FieldTypeString:
		return "string"
	case FieldTypeNumber:
		return "number"
	case FieldTypeBool:
		return "bool"
	case FieldTypeFile:
		return "file"
	case FieldTypeObject:
		return "object"
	case FieldTypeArray:
		return "allowArray"
	case FieldTypeAny:
		return "any"
	default:
		return "invalid"
	}
}

func parseValueToType(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.String:
		return FieldTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return FieldTypeNumber
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FieldTypeNumber
	case reflect.Float32, reflect.Float64:
		return FieldTypeNumber
	case reflect.Bool:
		return FieldTypeBool
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return FieldTypeFile
		}

		return FieldTypeArray
	case reflect.Pointer:
		return parseValueToType(t.Elem())
	case reflect.Struct:
		return FieldTypeObject
	default:
		return FieldTypeAny
	}
}

type FieldFn func(fl []*Field) []*Field

func reflectField(allowArray bool, t reflect.Type) ([]*Field, error) {
	switch t.Kind() {
	case reflect.Struct:
		list := []*Field{}

		for i := 0; i < t.NumField(); i++ {
			tag := tagparser.Parse(string(t.Field(i).Tag))
			field := &Field{
				Name:   strings.ToLower(t.Field(i).Name),
				Detail: "N/A",
				Type:   parseValueToType(t.Field(i).Type),
			}

			if n, ok := tag["name"]; ok && n != "" {
				field.Name = n
			}

			if n, ok := tag["detail"]; ok && n != "" {
				field.Detail = n
			}

			if n, ok := tag["key"]; ok && n != "" {
				field.Key = n
			}
			if n, ok := tag["json"]; ok && n != "" {
				field.Key = n
			}

			if n, ok := tag["enum"]; ok && n != "" {
				field.Enum = strings.Split(n, ",")
			}

			if n, ok := tag["default"]; ok && n != "" {
				field.Default = n
			}

			kind := t.Field(i).Type.Kind()
			switch kind {
			case reflect.Slice, reflect.Array, reflect.Map, reflect.Pointer:
				if kind == reflect.Pointer &&
					t.Field(i).Type.Elem().Kind() != reflect.Struct {
					break
				}

				if kind != reflect.Pointer && !allowArray {
					return nil, ErrUnsupportedArrayType
				}

				cfl, e := reflectField(allowArray, t.Field(i).Type.Elem())
				if e != nil {
					return nil, e
				}

				field.Children = cfl
			}

			list = append(list, field)
		}

		return list, nil
	case reflect.Pointer:
		return reflectField(allowArray, t.Elem())

	default:
		return []*Field{{
			Name:   strings.ToLower(t.Name()),
			Type:   parseValueToType(t),
			Detail: "N/A",
		}}, nil
	}
}
