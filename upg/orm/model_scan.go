package orm

import (
	"fmt"
	"reflect"
	"time"

	"uw/pkg/cast"
	"uw/upg/internal"
	"uw/upg/types"
)

type scanValuesModel struct {
	ModelDiscard
	values []interface{}
}

var _ Model = scanValuesModel{}

func Scan(values ...interface{}) scanValuesModel {
	return scanValuesModel{
		values: values,
	}
}

func (m scanValuesModel) NextColumnScanner() ColumnScanner {
	return m
}

func (m scanValuesModel) ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error {
	if int(col.Index) >= len(m.values) {
		return fmt.Errorf("pg: no Scan var for column index=%d name=%q",
			col.Index, col.Name)
	}
	return types.Scan(m.values[col.Index], rd, n)
}

//------------------------------------------------------------------------------

type scanReflectValuesModel struct {
	ModelDiscard
	values []reflect.Value
}

var _ Model = scanReflectValuesModel{}

func scanReflectValues(values []reflect.Value) scanReflectValuesModel {
	return scanReflectValuesModel{
		values: values,
	}
}

func (m scanReflectValuesModel) NextColumnScanner() ColumnScanner {
	return m
}

func (m scanReflectValuesModel) ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error {
	if int(col.Index) >= len(m.values) {
		return fmt.Errorf("pg: no Scan var for column index=%d name=%q",
			col.Index, col.Name)
	}
	return types.ScanValue(m.values[col.Index], rd, n)
}

//------------------------------------------------------------------------------

type sliceModel struct {
	ModelDiscard
	slice    reflect.Value
	nextElem func() reflect.Value
	scan     func(reflect.Value, types.Reader, int) error
}

var _ Model = (*sliceModel)(nil)

func newSliceModel(slice reflect.Value, elemType reflect.Type) *sliceModel {
	return &sliceModel{
		slice: slice,
		scan:  types.Scanner(elemType),
	}
}

func (m *sliceModel) Init() error {
	if m.slice.IsValid() && m.slice.Len() > 0 {
		m.slice.Set(m.slice.Slice(0, 0))
	}
	return nil
}

func (m *sliceModel) NextColumnScanner() ColumnScanner {
	return m
}

func (m *sliceModel) ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error {
	if m.nextElem == nil {
		m.nextElem = internal.MakeSliceNextElemFunc(m.slice)
	}
	v := m.nextElem()
	return m.scan(v, rd, n)
}

//------------------------------------------------------------------------------

type mapModel struct {
	ptr *map[string]interface{}
	m   map[string]interface{}
}

var _ Model = (*mapModel)(nil)

func newMapModel(ptr *map[string]interface{}) *mapModel {
	model := &mapModel{
		ptr: ptr,
	}
	if ptr != nil {
		model.m = *ptr
	}
	return model
}

func (m *mapModel) Init() error {
	return nil
}

func (m *mapModel) NextColumnScanner() ColumnScanner {
	if m.m == nil {
		m.m = make(map[string]interface{})
		*m.ptr = m.m
	}
	return m
}

func (m mapModel) AddColumnScanner(ColumnScanner) error {
	return nil
}

func (m *mapModel) ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error {
	val, err := types.ReadColumnValue(col, rd, n)
	if err != nil {
		return err
	}

	m.m[col.Name] = val
	return nil
}

//------------------------------------------------------------------------------

type mapSliceModel struct {
	mapModel
	slice *[]map[string]interface{}
}

var _ Model = (*mapSliceModel)(nil)

func newMapSliceModel(ptr *[]map[string]interface{}) *mapSliceModel {
	return &mapSliceModel{
		slice: ptr,
	}
}

func (m *mapSliceModel) Init() error {
	slice := *m.slice
	if len(slice) > 0 {
		*m.slice = slice[:0]
	}
	return nil
}

func (m *mapSliceModel) NextColumnScanner() ColumnScanner {
	slice := *m.slice
	if len(slice) == cap(slice) {
		m.mapModel.m = make(map[string]interface{})
		*m.slice = append(slice, m.mapModel.m) //nolint:gocritic
		return m
	}

	slice = slice[:len(slice)+1]
	el := slice[len(slice)-1]
	if el != nil {
		m.mapModel.m = el
	} else {
		el = make(map[string]interface{})
		slice[len(slice)-1] = el
		m.mapModel.m = el
	}
	*m.slice = slice
	return m
}

//------------------------------------------------------------------------------

type mapValue struct {
	m map[string]interface{}
}

func newMapValue() *mapValue {
	return &mapValue{
		m: make(map[string]interface{}),
	}
}

func (v *mapValue) Bool(column string, def ...bool) bool {
	val, e := cast.ToBoolE(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Int(column string, def ...int) int {
	val, e := cast.ToIntE(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Uint(column string, def ...uint) uint {
	val, e := cast.ToUintE(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Int64(column string, def ...int64) int64 {
	val, e := cast.ToInt64E(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Uint64(column string, def ...uint64) uint64 {
	val, e := cast.ToUint64E(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Float64(column string, def ...float64) float64 {
	val, e := cast.ToFloat64E(v.m[column])
	if e != nil && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) String(column string, def ...string) string {
	val, e := cast.ToStringE(v.m[column])
	if (e != nil || val == "") && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Time(column string, def ...time.Time) time.Time {
	val, e := cast.ToTimeE(v.m[column])
	if (e != nil || val.IsZero()) && len(def) > 0 {
		return def[0]
	}

	return val
}

func (v *mapValue) Value(column string, def ...interface{}) interface{} {
	val, ok := v.m[column]
	if (!ok || val == nil) && len(def) > 0 {
		return def[0]
	}

	return val
}
