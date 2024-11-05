package orm

import (
	"database/sql"
	"fmt"
	"reflect"

	"uw/upg/types"
)

type Model interface {
	Init() error
	NextColumnScanner() ColumnScanner
	AddColumnScanner(ColumnScanner) error
	ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error
}

var _ = Model((*ModelDiscard)(nil))

type ModelDiscard struct{}

func (ModelDiscard) Init() error {
	return nil
}

func (m ModelDiscard) NextColumnScanner() ColumnScanner {
	return m
}

func (m ModelDiscard) AddColumnScanner(ColumnScanner) error {
	return nil
}

func (m ModelDiscard) ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error {
	return nil
}

func NewModel(value interface{}, scan bool) (Model, error) {
	return newModel(value, scan)
}

func newScanModel(values []interface{}) (Model, error) {
	if len(values) > 1 {
		return Scan(values...), nil
	}
	return newModel(values[0], true)
}

func newModel(value interface{}, scan bool) (Model, error) {
	switch value := value.(type) {
	case Model:
		return value, nil
	case types.ValueScanner, sql.Scanner:
		if !scan {
			return nil, fmt.Errorf("pg: Model(unsupported %T)", value)
		}
		return Scan(value), nil
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil, errModelNil
	}
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("pg: Model(non-pointer %T)", value)
	}

	if v.IsNil() {
		typ := v.Type().Elem()
		if typ.Kind() == reflect.Struct {
			return newStructTableModel(reflect.New(typ)), nil
		}
		return nil, errModelNil
	}

	v = v.Elem()

	if v.Kind() == reflect.Interface {
		if !v.IsNil() {
			v = v.Elem()
			if v.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("pg: Model(non-pointer %s)", v.Type().String())
			}
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		if v.Type() != timeType {
			return newStructTableModel(v), nil
		}
	case reflect.Slice:
		elemType := sliceElemType(v)
		switch elemType.Kind() {
		case reflect.Struct:
			if elemType != timeType {
				return newSliceTableModel(v), nil
			}
		case reflect.Map:
			if err := validMap(elemType); err != nil {
				return nil, err
			}
			slicePtr := v.Addr().Interface().(*[]map[string]interface{})
			return newMapSliceModel(slicePtr), nil
		}
		return newSliceModel(v, elemType), nil
	case reflect.Map:
		typ := v.Type()
		if err := validMap(typ); err != nil {
			return nil, err
		}
		mapPtr := v.Addr().Interface().(*map[string]interface{})
		return newMapModel(mapPtr), nil
	}

	if !scan {
		return nil, fmt.Errorf("pg: Model(unsupported %T)", value)
	}
	return Scan(value), nil
}

func validMap(typ reflect.Type) error {
	if typ.Key().Kind() != reflect.String || typ.Elem().Kind() != reflect.Interface {
		return fmt.Errorf("pg: Model(unsupported %s, expected *map[string]interface{})",
			typ.String())
	}
	return nil
}
