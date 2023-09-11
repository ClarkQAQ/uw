package orm

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"uw/upg/types"
)

type UpdateQuery struct {
	q           *Query
	omitZero    bool
	placeholder bool
}

var (
	_ QueryAppender = (*UpdateQuery)(nil)
	_ QueryCommand  = (*UpdateQuery)(nil)
)

func NewUpdateQuery(q *Query, omitZero bool) *UpdateQuery {
	return &UpdateQuery{
		q:        q,
		omitZero: omitZero,
	}
}

func (q *UpdateQuery) String() string {
	b, err := q.AppendQuery(defaultFmter, nil)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (q *UpdateQuery) Operation() QueryOp {
	return UpdateOp
}

func (q *UpdateQuery) Clone() QueryCommand {
	return &UpdateQuery{
		q:           q.q.Clone(),
		omitZero:    q.omitZero,
		placeholder: q.placeholder,
	}
}

func (q *UpdateQuery) Query() *Query {
	return q.q
}

func (q *UpdateQuery) AppendTemplate(b []byte) ([]byte, error) {
	cp := q.Clone().(*UpdateQuery)
	cp.placeholder = true
	return cp.AppendQuery(dummyFormatter{}, b)
}

func (q *UpdateQuery) AppendQuery(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	if q.q.stickyErr != nil {
		return nil, q.q.stickyErr
	}

	if len(q.q.with) > 0 {
		b, err = q.q.appendWith(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	b = append(b, "UPDATE "...)

	b, err = q.q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	b, err = q.mustAppendSet(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.q.hasMultiTables() {
		b = append(b, " FROM "...)
		b, err = q.q.appendOtherTables(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	b, err = q.mustAppendWhere(fmter, b, q.q.hasMultiTables())
	if err != nil {
		return nil, err
	}

	if len(q.q.returning) > 0 {
		b, err = q.q.appendReturning(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, q.q.stickyErr
}

func (q *UpdateQuery) mustAppendWhere(
	fmter QueryFormatter, b []byte, isSliceModelWithData bool,
) (_ []byte, err error) {
	b = append(b, " WHERE "...)

	if !isSliceModelWithData {
		return q.q.mustAppendWhere(fmter, b)
	}

	if len(q.q.where) > 0 {
		return q.q.appendWhere(fmter, b, q.q.where)
	}

	return b, nil
}

func (q *UpdateQuery) mustAppendSet(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	if len(q.q.set) > 0 {
		return q.q.appendSet(fmter, b)
	}

	b = append(b, " SET "...)

	switch val := q.q.model.(type) {
	case *mapModel:
		return q.appendMapSet(b, val.m), nil
	case *structTableModel:
		return q.appendSetStruct(fmter, b, val.value)
	default:
		return b, fmt.Errorf("pg: can't append set for %T", q.q.model)
	}
}

func (q *UpdateQuery) appendMapSet(b []byte, m map[string]interface{}) []byte {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}

		b = types.AppendIdent(b, k, 1)
		b = append(b, " = "...)
		if q.placeholder {
			b = append(b, '?')
		} else {
			b = types.Append(b, m[k], 1)
		}
	}

	return b
}

func (q *UpdateQuery) appendSetStruct(fmter QueryFormatter, b []byte, strct reflect.Value) (_ []byte, e error) {
	strctp, pos := strct.Type(), len(b)
	for i := 0; i < strctp.NumField(); i++ {
		val, tp := strct.Field(i), strctp.Field(i)

		if (q.omitZero && val.IsNil()) || !val.CanInterface() {
			continue
		}

		if tp.Type.Kind() == reflect.Struct {
			if b, e = q.appendSetStruct(fmter, b, val); e != nil {
				return b, e
			}

			continue
		}

		if tag := tp.Tag.Get("db"); tag != "-" {
			if len(b) != pos {
				b = append(b, ", "...)
				pos = len(b)
			}

			if len(tag) < 1 {
				tag = strings.ToLower(tp.Name)
			}

			b = append(b, tag...)
			b = append(b, " = "...)

			if q.placeholder {
				b = append(b, '?')
				continue
			}

			b = types.Append(b, val.Interface(), 1)
		}
	}

	return b, nil
}
