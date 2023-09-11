package orm

import (
	"fmt"
	"reflect"
	"strings"

	"uw/upg/internal"
	"uw/upg/types"
)

type structTableModel struct {
	ModelDiscard
	fields        map[string][]int
	value         reflect.Value
	structInited  bool
	structInitErr error
}

var _ Model = (*structTableModel)(nil)

func newStructTableModel(v reflect.Value) *structTableModel {
	return &structTableModel{
		value: v,
	}
}

func (m *structTableModel) initStruct() error {
	if m.structInited {
		return m.structInitErr
	}
	m.structInited = true

	switch m.value.Kind() {
	case reflect.Invalid:
		m.structInitErr = errModelNil
		return m.structInitErr
	case reflect.Interface:
		m.value = m.value.Elem()
	}

	if m.value.Kind() == reflect.Ptr {
		if m.value.IsNil() {
			m.value.Set(reflect.New(m.value.Type().Elem()))
			m.value = m.value.Elem()
		} else {
			m.value = m.value.Elem()
		}
	}

	m.fields = make(map[string][]int)
	if e := m.parserField(m.value.Type()); e != nil {
		m.structInitErr = e
		return e
	}

	return nil
}

func (m *structTableModel) parserField(val reflect.Type) error {
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)

		if f.Type.Kind() == reflect.Struct {
			if e := m.parserField(f.Type); e != nil {
				return e
			}
			continue
		}

		if tag := f.Tag.Get("db"); tag != "-" {
			if len(tag) < 1 {
				tag = strings.ToLower(f.Name)
			}

			if len(m.fields[tag]) > 0 {
				return fmt.Errorf("duplicate db tag: %s", tag)
			}

			m.fields[tag] = f.Index
		}
	}

	return nil
}

func (m *structTableModel) ScanColumn(
	col types.ColumnInfo, rd types.Reader, n int,
) error {
	ok, err := m.scanColumn(col, rd, n)
	if ok {
		return err
	}
	if col.Name[0] == '_' {
		return nil
	}
	return fmt.Errorf(
		"pg: can't find column=%s"+
			"(prefix the column with underscore or use discard_unknown_columns)",
		col.Name,
	)
}

func (m *structTableModel) scanColumn(col types.ColumnInfo, rd types.Reader, n int) (bool, error) {
	// Don't init nil struct if value is NULL.
	if n == -1 &&
		!m.structInited &&
		m.value.Kind() == reflect.Ptr &&
		m.value.IsNil() {
		return true, nil
	}

	if err := m.initStruct(); err != nil {
		return false, err
	}

	if fields, ok := m.fields[col.Name]; ok && len(fields) > 0 {
		for i := 0; i < len(fields); i++ {
			if e := types.ScanValue(m.value.Field(fields[i]), rd, n); e != nil {
				return false, e
			}
		}
	}

	return true, nil
}

func (m *structTableModel) NextColumnScanner() ColumnScanner {
	return m
}

type sliceTableModel struct {
	structTableModel

	slice      reflect.Value
	sliceLen   int
	sliceOfPtr bool
	nextElem   func() reflect.Value
}

var _ Model = (*sliceTableModel)(nil)

func newSliceTableModel(slice reflect.Value) *sliceTableModel {
	m := &sliceTableModel{
		structTableModel: structTableModel{
			value: slice,
		},
		slice:    slice,
		sliceLen: slice.Len(),
		nextElem: internal.MakeSliceNextElemFunc(slice),
	}
	m.init(slice.Type())
	return m
}

func (m *sliceTableModel) init(sliceType reflect.Type) {
	switch sliceType.Elem().Kind() {
	case reflect.Ptr, reflect.Interface:
		m.sliceOfPtr = true
	}
}

func (m *sliceTableModel) Init() error {
	if m.slice.IsValid() && m.slice.Len() > 0 {
		m.slice.Set(m.slice.Slice(0, 0))
	}
	return nil
}

func (m *sliceTableModel) NextColumnScanner() ColumnScanner {
	m.value = m.nextElem()
	m.structInited = false
	return m
}

func (m *sliceTableModel) AddColumnScanner(_ ColumnScanner) error {
	return nil
}
