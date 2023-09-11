package orm

import (
	"bytes"
	"strconv"
	"strings"
)

type SelectQuery struct {
	q     *Query
	count string
}

var (
	_ QueryAppender = (*SelectQuery)(nil)
	_ QueryCommand  = (*SelectQuery)(nil)
)

func NewSelectQuery(q *Query) *SelectQuery {
	return &SelectQuery{
		q: q,
	}
}

func (q *SelectQuery) String() string {
	b, err := q.AppendQuery(defaultFmter, nil)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (q *SelectQuery) Operation() QueryOp {
	return SelectOp
}

func (q *SelectQuery) Clone() QueryCommand {
	return &SelectQuery{
		q:     q.q.Clone(),
		count: q.count,
	}
}

func (q *SelectQuery) Query() *Query {
	return q.q
}

func (q *SelectQuery) AppendTemplate(b []byte) ([]byte, error) {
	return q.AppendQuery(dummyFormatter{}, b)
}

func (q *SelectQuery) AppendQuery(fmter QueryFormatter, b []byte) (_ []byte, err error) { //nolint:gocyclo
	if q.q.stickyErr != nil {
		return nil, q.q.stickyErr
	}

	cteCount := q.count != "" && (len(q.q.group) > 0 || q.isDistinct())
	if cteCount {
		b = append(b, `WITH "_count_wrapper" AS (`...)
	}

	if len(q.q.with) > 0 {
		b, err = q.q.appendWith(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if len(q.q.union) > 0 {
		b = append(b, '(')
	}

	b = append(b, "SELECT "...)

	if len(q.q.distinctOn) > 0 {
		b = append(b, "DISTINCT ON ("...)
		for i, app := range q.q.distinctOn {
			if i > 0 {
				b = append(b, ", "...)
			}

			b, err = app.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, ") "...)
	} else if q.q.distinctOn != nil {
		b = append(b, "DISTINCT "...)
	}

	if q.count != "" && !cteCount {
		b = append(b, q.count...)
	} else {
		b, err = q.appendColumns(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if q.q.hasTables() {
		b = append(b, " FROM "...)
		b, err = q.q.appendTables(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	for _, j := range q.q.joins {
		b, err = j.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if len(q.q.where) > 0 {
		b = append(b, " WHERE "...)
		b, err = q.q.appendWhere(fmter, b, q.q.where)
		if err != nil {
			return nil, err
		}
	}

	if len(q.q.group) > 0 {
		b = append(b, " GROUP BY "...)
		for i, f := range q.q.group {
			if i > 0 {
				b = append(b, ", "...)
			}
			b, err = f.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(q.q.having) > 0 {
		b = append(b, " HAVING "...)
		for i, f := range q.q.having {
			if i > 0 {
				b = append(b, " AND "...)
			}
			b = append(b, '(')
			b, err = f.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ')')
		}
	}

	if q.count == "" {
		if len(q.q.order) > 0 {
			b = append(b, " ORDER BY "...)
			for i, f := range q.q.order {
				if i > 0 {
					b = append(b, ", "...)
				}
				b, err = f.AppendQuery(fmter, b)
				if err != nil {
					return nil, err
				}
			}
		}

		if q.q.limit != 0 {
			b = append(b, " LIMIT "...)
			b = strconv.AppendInt(b, int64(q.q.limit), 10)
		}

		if q.q.offset != 0 {
			b = append(b, " OFFSET "...)
			b = strconv.AppendInt(b, int64(q.q.offset), 10)
		}

		if q.q.selFor != nil {
			b = append(b, " FOR "...)
			b, err = q.q.selFor.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		}
	} else if cteCount {
		b = append(b, `) SELECT `...)
		b = append(b, q.count...)
		b = append(b, ` FROM "_count_wrapper"`...)
	}

	if len(q.q.union) > 0 {
		b = append(b, ")"...)

		for _, u := range q.q.union {
			b = append(b, u.expr...)
			b = append(b, '(')
			b, err = u.query.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ")"...)
		}
	}

	return b, q.q.stickyErr
}

func (q SelectQuery) appendColumns(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	switch {
	case q.q.columns != nil:
		b, err = q.q.appendColumns(fmter, b)
		if err != nil {
			return nil, err
		}
	default:
		b = append(b, '*')
	}

	b = bytes.TrimSuffix(b, []byte(", "))

	return b, nil
}

func (q *SelectQuery) isDistinct() bool {
	if q.q.distinctOn != nil {
		return true
	}
	for _, column := range q.q.columns {
		column, ok := column.(*SafeQueryAppender)
		if ok {
			if strings.Contains(column.query, "DISTINCT") ||
				strings.Contains(column.query, "distinct") {
				return true
			}
		}
	}
	return false
}

//------------------------------------------------------------------------------

type joinQuery struct {
	join *SafeQueryAppender
	on   []*condAppender
}

func (j *joinQuery) AppendOn(app *condAppender) {
	j.on = append(j.on, app)
}

func (j *joinQuery) AppendQuery(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	b = append(b, ' ')

	b, err = j.join.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	if len(j.on) > 0 {
		b = append(b, " ON "...)
		for i, on := range j.on {
			if i > 0 {
				b = on.AppendSep(b)
			}
			b, err = on.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		}
	}

	return b, nil
}
