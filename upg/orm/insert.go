package orm

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"uw/upg/types"
)

type InsertQuery struct {
	q           *Query
	placeholder bool
}

var _ QueryCommand = (*InsertQuery)(nil)

func NewInsertQuery(q *Query) *InsertQuery {
	return &InsertQuery{
		q: q,
	}
}

func (q *InsertQuery) String() string {
	b, err := q.AppendQuery(defaultFmter, nil)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (q *InsertQuery) Operation() QueryOp {
	return InsertOp
}

func (q *InsertQuery) Clone() QueryCommand {
	return &InsertQuery{
		q:           q.q.Clone(),
		placeholder: q.placeholder,
	}
}

func (q *InsertQuery) Query() *Query {
	return q.q
}

var _ TemplateAppender = (*InsertQuery)(nil)

func (q *InsertQuery) AppendTemplate(b []byte) ([]byte, error) {
	cp := q.Clone().(*InsertQuery)
	cp.placeholder = true
	return cp.AppendQuery(dummyFormatter{}, b)
}

var _ QueryAppender = (*InsertQuery)(nil)

func (q *InsertQuery) AppendQuery(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	if q.q.stickyErr != nil {
		return nil, q.q.stickyErr
	}

	if len(q.q.with) > 0 {
		b, err = q.q.appendWith(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	b = append(b, "INSERT INTO "...)
	if q.q.onConflict != nil {
		b, err = q.q.appendFirstTable(fmter, b)
	} else {
		b, err = q.q.appendFirstTable(fmter, b)
	}
	if err != nil {
		return nil, err
	}

	b, err = q.appendColumnsValues(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.q.onConflict != nil {
		b = append(b, " ON CONFLICT "...)
		b, err = q.q.onConflict.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}

		if q.q.onConflictDoUpdate() {
			if len(q.q.set) > 0 {
				b, err = q.q.appendSet(fmter, b)
				if err != nil {
					return nil, err
				}
			}

			if len(q.q.updWhere) > 0 {
				b = append(b, " WHERE "...)
				b, err = q.q.appendUpdWhere(fmter, b)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if len(q.q.returning) > 0 {
		b, err = q.q.appendReturning(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, q.q.stickyErr
}

func (q *InsertQuery) appendColumnsValues(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	if q.q.hasMultiTables() {
		if q.q.columns != nil {
			b = append(b, " ("...)
			b, err = q.q.appendColumns(fmter, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ")"...)
		}

		b = append(b, " SELECT * FROM "...)
		b, err = q.q.appendOtherTables(fmter, b)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	switch val := q.q.model.(type) {
	case *mapModel:
		return q.appendMapColumnsValues(b, val.m), nil
	case *structTableModel:
		return q.appendStructColumnsValues(b, val.value)
	default:
		return b, fmt.Errorf("pg: can't append set for %T", q.q.model)
	}
}

func (q *InsertQuery) appendStructColumnsValues(b []byte, strct reflect.Value) (_ []byte, e error) {
	strctp := strct.Type()
	index := make([]int, 0, strctp.NumField())

	b = append(b, " ("...)

	for i := 0; i < strctp.NumField(); i++ {
		tp := strctp.Field(i)
		if tag := tp.Tag.Get("db"); tag != "-" {
			if len(tag) < 1 {
				tag = strings.ToLower(tp.Name)
			}
			if len(index) > 0 {
				b = append(b, ", "...)
			}
			b = types.AppendIdent(b, tag, 1)
			index = append(index, i)
		}
	}

	b = append(b, ") VALUES ("...)

	for i := 0; i < len(index); i++ {
		if i > 0 {
			b = append(b, ", "...)
		}
		if q.placeholder {
			b = append(b, '?')

			continue
		}

		if f := strct.Field(index[i]); f.CanInterface() {
			b = types.Append(b, f.Interface(), 1)
		} else {
			b = types.Append(b, nil, 1)
		}
	}

	b = append(b, ")"...)

	return b, nil
}

func (q *InsertQuery) appendMapColumnsValues(b []byte, m map[string]interface{}) []byte {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b = append(b, " ("...)

	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = types.AppendIdent(b, k, 1)
	}

	b = append(b, ") VALUES ("...)

	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		if q.placeholder {
			b = append(b, '?')
		} else {
			b = types.Append(b, m[k], 1)
		}
	}

	b = append(b, ")"...)

	return b
}
