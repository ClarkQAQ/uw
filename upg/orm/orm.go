package orm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"uw/upg/internal"
	"uw/upg/types"
)

var (
	errModelNil = errors.New("pg: Model(nil)")
	timeType    = reflect.TypeOf((*time.Time)(nil)).Elem()
)

type QueryOp string

const (
	SelectOp QueryOp = "SELECT"
	InsertOp QueryOp = "INSERT"
	UpdateOp QueryOp = "UPDATE"
	DeleteOp QueryOp = "DELETE"
)

// ColumnScanner is used to scan column values.
type ColumnScanner interface {
	// Scan assigns a column value from a row.
	//
	// An error should be returned if the value can not be stored
	// without loss of information.
	ScanColumn(col types.ColumnInfo, rd types.Reader, n int) error
}

type QueryAppender interface {
	AppendQuery(fmter QueryFormatter, b []byte) ([]byte, error)
}

type TemplateAppender interface {
	AppendTemplate(b []byte) ([]byte, error)
}

type QueryCommand interface {
	QueryAppender
	TemplateAppender
	String() string
	Operation() QueryOp
	Clone() QueryCommand
	Query() *Query
}

// DB is a common interface for pg.DB and pg.Tx types.
type DB interface {
	Table(tableName string, alias ...string) *Query
	TableContext(ctx context.Context, tableName string, alias ...string) *Query

	Exec(query interface{}, params ...interface{}) (Result, error)
	ExecContext(c context.Context, query interface{}, params ...interface{}) (Result, error)
	Query(model, query interface{}, params ...interface{}) (Result, error)
	QueryContext(c context.Context, model, query interface{}, params ...interface{}) (Result, error)

	CopyFrom(r io.Reader, query interface{}, params ...interface{}) (Result, error)
	CopyTo(w io.Writer, query interface{}, params ...interface{}) (Result, error)

	Context() context.Context
	Formatter() QueryFormatter
}

// Result summarizes an executed SQL command.
type Result interface {
	// RowsAffected returns the number of rows affected by SELECT, INSERT, UPDATE,
	// or DELETE queries. It returns -1 if query can't possibly affect any rows,
	// e.g. in case of CREATE or SHOW queries.
	RowsAffected() int

	// RowsReturned returns the number of rows returned by the query.
	RowsReturned() int
}

type withQuery struct {
	name  string
	query QueryAppender
}

//nolint:unused
type columnValue struct {
	column string
	value  *SafeQueryAppender
}

type union struct {
	expr  string
	query *Query
}

type Query struct {
	ctx       context.Context
	db        DB
	stickyErr error

	model Model

	with         []withQuery
	tables       []QueryAppender
	distinctOn   []*SafeQueryAppender
	columns      []QueryAppender
	set          []QueryAppender
	where        []queryWithSepAppender
	updWhere     []queryWithSepAppender
	group        []QueryAppender
	having       []*SafeQueryAppender
	union        []*union
	joins        []QueryAppender
	joinAppendOn func(app *condAppender)
	order        []QueryAppender
	limit        int
	offset       int
	selFor       *SafeQueryAppender

	onConflict *SafeQueryAppender
	returning  []*SafeQueryAppender
}

func NewQuery(db DB, model ...interface{}) *Query {
	ctx := context.Background()
	if db != nil {
		ctx = db.Context()
	}
	q := &Query{ctx: ctx}
	return q.DB(db).Model(model...)
}

func NewQueryContext(ctx context.Context, db DB, model ...interface{}) *Query {
	return NewQuery(db, model...).Context(ctx)
}

// New returns new zero Query bound to the current db.
func (q *Query) New() *Query {
	return &Query{
		ctx: q.ctx,
		db:  q.db,

		model: q.model,
	}
}

// Clone clones the Query.
func (q *Query) Clone() *Query {
	clone := &Query{
		ctx:       q.ctx,
		db:        q.db,
		stickyErr: q.stickyErr,

		model: q.model,

		with:       q.with[:len(q.with):len(q.with)],
		tables:     q.tables[:len(q.tables):len(q.tables)],
		distinctOn: q.distinctOn[:len(q.distinctOn):len(q.distinctOn)],
		columns:    q.columns[:len(q.columns):len(q.columns)],
		set:        q.set[:len(q.set):len(q.set)],
		where:      q.where[:len(q.where):len(q.where)],
		updWhere:   q.updWhere[:len(q.updWhere):len(q.updWhere)],
		joins:      q.joins[:len(q.joins):len(q.joins)],
		group:      q.group[:len(q.group):len(q.group)],
		having:     q.having[:len(q.having):len(q.having)],
		union:      q.union[:len(q.union):len(q.union)],
		order:      q.order[:len(q.order):len(q.order)],
		limit:      q.limit,
		offset:     q.offset,
		selFor:     q.selFor,

		onConflict: q.onConflict,
		returning:  q.returning[:len(q.returning):len(q.returning)],
	}

	return clone
}

func (q *Query) err(err error) *Query {
	if q.stickyErr == nil {
		q.stickyErr = err
	}
	return q
}

func (q *Query) Context(c context.Context) *Query {
	q.ctx = c
	return q
}

func (q *Query) DB(db DB) *Query {
	q.db = db
	return q
}

func (q *Query) Model(model ...interface{}) *Query {
	var err error
	switch l := len(model); {
	case l == 0:
		q.model = nil
	case l == 1:
		q.model, err = NewModel(model[0])
	case l > 1:
		q.model, err = NewModel(&model)
	default:
		panic("not reached")
	}
	if err != nil {
		q = q.err(err)
	}

	return q
}

// With adds subq as common table expression with the given name.
func (q *Query) With(name string, subq *Query) *Query {
	return q._with(name, NewSelectQuery(subq))
}

func (q *Query) WithInsert(name string, subq *Query) *Query {
	return q._with(name, NewInsertQuery(subq))
}

func (q *Query) WithUpdate(name string, subq *Query) *Query {
	return q._with(name, NewUpdateQuery(subq, false))
}

func (q *Query) WithDelete(name string, subq *Query) *Query {
	return q._with(name, NewDeleteQuery(subq))
}

func (q *Query) _with(name string, subq QueryAppender) *Query {
	q.with = append(q.with, withQuery{
		name:  name,
		query: subq,
	})
	return q
}

// WrapWith creates new Query and adds to it current query as
// common table expression with the given name.
func (q *Query) WrapWith(name string) *Query {
	wrapper := q.New()
	wrapper.with = q.with
	q.with = nil
	wrapper = wrapper.With(name, q)
	return wrapper
}

func (q *Query) Table(table string, alias ...string) *Query {
	t := tableAppender{table: table}
	if len(alias) > 0 {
		t.alias = alias[0]
	}
	q.tables = append(q.tables, t)
	return q
}

func (q *Query) TableExpr(expr string, params ...interface{}) *Query {
	q.tables = append(q.tables, SafeQuery(expr, params...))
	return q
}

func (q *Query) Distinct() *Query {
	q.distinctOn = make([]*SafeQueryAppender, 0)
	return q
}

func (q *Query) DistinctOn(expr string, params ...interface{}) *Query {
	q.distinctOn = append(q.distinctOn, SafeQuery(expr, params...))
	return q
}

// Column adds a column to the Query quoting it according to PostgreSQL rules.
// Does not expand params like ?TableAlias etc.
// ColumnExpr can be used to bypass quoting restriction or for params expansion.
// Column name can be:
//   - column_name,
//   - table_alias.column_name,
//   - table_alias.*.
func (q *Query) Column(columns ...string) *Query {
	for _, column := range columns {
		if column == "_" {
			if q.columns == nil {
				q.columns = make([]QueryAppender, 0)
			}
			continue
		}

		q.columns = append(q.columns, fieldAppender{column})
	}
	return q
}

// ColumnExpr adds column expression to the Query.
func (q *Query) ColumnExpr(expr string, params ...interface{}) *Query {
	q.columns = append(q.columns, SafeQuery(expr, params...))
	return q
}

// ExcludeColumn excludes a column from the list of to be selected columns.
func (q *Query) ExcludeColumn(columns ...string) *Query {
	for _, col := range columns {
		if !q.excludeColumn(col) {
			return q.err(fmt.Errorf("pg: can't find column=%q", col))
		}
	}
	return q
}

func (q *Query) excludeColumn(column string) bool {
	for i := 0; i < len(q.columns); i++ {
		app, ok := q.columns[i].(fieldAppender)
		if ok && app.field == column {
			q.columns = append(q.columns[:i], q.columns[i+1:]...)
			return true
		}
	}
	return false
}

func (q *Query) Set(set string, params ...interface{}) *Query {
	q.set = append(q.set, SafeQuery(set, params...))
	return q
}

func (q *Query) Where(condition string, params ...interface{}) *Query {
	q.addWhere(&condAppender{
		sep:    " AND ",
		cond:   condition,
		params: params,
	})
	return q
}

func (q *Query) WhereOr(condition string, params ...interface{}) *Query {
	q.addWhere(&condAppender{
		sep:    " OR ",
		cond:   condition,
		params: params,
	})
	return q
}

// WhereGroup encloses conditions added in the function in parentheses.
//
//	q.Where("TRUE").
//		WhereGroup(func(q *pg.Query) (*pg.Query, error) {
//			q = q.WhereOr("FALSE").WhereOr("TRUE").
//			return q, nil
//		})
//
// generates
//
//	WHERE TRUE AND (FALSE OR TRUE)
func (q *Query) WhereGroup(fn func(*Query) (*Query, error)) *Query {
	return q.whereGroup(" AND ", fn)
}

// WhereGroup encloses conditions added in the function in parentheses.
//
//	q.Where("TRUE").
//		WhereNotGroup(func(q *pg.Query) (*pg.Query, error) {
//			q = q.WhereOr("FALSE").WhereOr("TRUE").
//			return q, nil
//		})
//
// generates
//
//	WHERE TRUE AND NOT (FALSE OR TRUE)
func (q *Query) WhereNotGroup(fn func(*Query) (*Query, error)) *Query {
	return q.whereGroup(" AND NOT ", fn)
}

// WhereOrGroup encloses conditions added in the function in parentheses.
//
//	q.Where("TRUE").
//		WhereOrGroup(func(q *pg.Query) (*pg.Query, error) {
//			q = q.Where("FALSE").Where("TRUE").
//			return q, nil
//		})
//
// generates
//
//	WHERE TRUE OR (FALSE AND TRUE)
func (q *Query) WhereOrGroup(fn func(*Query) (*Query, error)) *Query {
	return q.whereGroup(" OR ", fn)
}

// WhereOrGroup encloses conditions added in the function in parentheses.
//
//	q.Where("TRUE").
//		WhereOrGroup(func(q *pg.Query) (*pg.Query, error) {
//			q = q.Where("FALSE").Where("TRUE").
//			return q, nil
//		})
//
// generates
//
//	WHERE TRUE OR NOT (FALSE AND TRUE)
func (q *Query) WhereOrNotGroup(fn func(*Query) (*Query, error)) *Query {
	return q.whereGroup(" OR NOT ", fn)
}

func (q *Query) whereGroup(conj string, fn func(*Query) (*Query, error)) *Query {
	saved := q.where
	q.where = nil

	newq, err := fn(q)
	if err != nil {
		q.err(err)
		return q
	}

	if len(newq.where) == 0 {
		newq.where = saved
		return newq
	}

	f := &condGroupAppender{
		sep:  conj,
		cond: newq.where,
	}
	newq.where = saved
	newq.addWhere(f)

	return newq
}

// WhereIn is a shortcut for Where and pg.In.
func (q *Query) WhereIn(where string, slice interface{}) *Query {
	return q.Where(where, types.In(slice))
}

// WhereInOr is a shortcut for WhereOr and pg.In.
func (q *Query) WhereInOr(where string, slice interface{}) *Query {
	return q.WhereOr(where, types.In(slice))
}

// WhereInMulti is a shortcut for Where and pg.InMulti.
func (q *Query) WhereInMulti(where string, values ...interface{}) *Query {
	return q.Where(where, types.InMulti(values...))
}

func (q *Query) addWhere(f queryWithSepAppender) {
	if q.onConflictDoUpdate() {
		q.updWhere = append(q.updWhere, f)
	} else {
		q.where = append(q.where, f)
	}
}

func (q *Query) Join(join string, params ...interface{}) *Query {
	j := &joinQuery{
		join: SafeQuery(join, params...),
	}
	q.joins = append(q.joins, j)
	q.joinAppendOn = j.AppendOn
	return q
}

// JoinOn appends join condition to the last join.
func (q *Query) JoinOn(condition string, params ...interface{}) *Query {
	if q.joinAppendOn == nil {
		q.err(errors.New("pg: no joins to apply JoinOn"))
		return q
	}
	q.joinAppendOn(&condAppender{
		sep:    " AND ",
		cond:   condition,
		params: params,
	})
	return q
}

func (q *Query) JoinOnOr(condition string, params ...interface{}) *Query {
	if q.joinAppendOn == nil {
		q.err(errors.New("pg: no joins to apply JoinOn"))
		return q
	}
	q.joinAppendOn(&condAppender{
		sep:    " OR ",
		cond:   condition,
		params: params,
	})
	return q
}

func (q *Query) Group(columns ...string) *Query {
	for _, column := range columns {
		q.group = append(q.group, fieldAppender{column})
	}
	return q
}

func (q *Query) GroupExpr(group string, params ...interface{}) *Query {
	q.group = append(q.group, SafeQuery(group, params...))
	return q
}

func (q *Query) Having(having string, params ...interface{}) *Query {
	q.having = append(q.having, SafeQuery(having, params...))
	return q
}

func (q *Query) Union(other *Query) *Query {
	return q.addUnion(" UNION ", other)
}

func (q *Query) UnionAll(other *Query) *Query {
	return q.addUnion(" UNION ALL ", other)
}

func (q *Query) Intersect(other *Query) *Query {
	return q.addUnion(" INTERSECT ", other)
}

func (q *Query) IntersectAll(other *Query) *Query {
	return q.addUnion(" INTERSECT ALL ", other)
}

func (q *Query) Except(other *Query) *Query {
	return q.addUnion(" EXCEPT ", other)
}

func (q *Query) ExceptAll(other *Query) *Query {
	return q.addUnion(" EXCEPT ALL ", other)
}

func (q *Query) addUnion(expr string, other *Query) *Query {
	q.union = append(q.union, &union{
		expr:  expr,
		query: other,
	})
	return q
}

// Order adds sort order to the Query quoting column name. Does not expand params like ?TableAlias etc.
// OrderExpr can be used to bypass quoting restriction or for params expansion.
func (q *Query) Order(orders ...string) *Query {
loop:
	for _, order := range orders {
		if order == "" {
			continue
		}
		ind := strings.Index(order, " ")
		if ind != -1 {
			field := order[:ind]
			sort := order[ind+1:]
			switch internal.UpperString(sort) {
			case "ASC", "DESC", "ASC NULLS FIRST", "DESC NULLS FIRST",
				"ASC NULLS LAST", "DESC NULLS LAST":
				q = q.OrderExpr("? ?", types.Ident(field), types.Safe(sort))
				continue loop
			}
		}

		q.order = append(q.order, fieldAppender{order})
	}
	return q
}

// Order adds sort order to the Query.
func (q *Query) OrderExpr(order string, params ...interface{}) *Query {
	if order != "" {
		q.order = append(q.order, SafeQuery(order, params...))
	}
	return q
}

func (q *Query) Limit(n int) *Query {
	q.limit = n
	return q
}

func (q *Query) Offset(n int) *Query {
	q.offset = n
	return q
}

func (q *Query) OnConflict(s string, params ...interface{}) *Query {
	q.onConflict = SafeQuery(s, params...)
	return q
}

func (q *Query) onConflictDoUpdate() bool {
	return q.onConflict != nil &&
		strings.HasSuffix(internal.UpperString(q.onConflict.query), "DO UPDATE")
}

// Returning adds a RETURNING clause to the query.
//
// `Returning("NULL")` can be used to suppress default returning clause
// generated by go-pg for INSERT queries to get values for null columns.
func (q *Query) Returning(s string, params ...interface{}) *Query {
	q.returning = append(q.returning, SafeQuery(s, params...))
	return q
}

func (q *Query) For(s string, params ...interface{}) *Query {
	q.selFor = SafeQuery(s, params...)
	return q
}

// Apply calls the fn passing the Query as an argument.
func (q *Query) Apply(fn func(*Query) (*Query, error)) *Query {
	qq, err := fn(q)
	if err != nil {
		q.err(err)
		return q
	}
	return qq
}

// Count returns number of rows matching the query using count aggregate function.
func (q *Query) Count() (int, error) {
	if q.stickyErr != nil {
		return 0, q.stickyErr
	}

	var count int
	_, err := q.db.QueryContext(
		q.ctx, Scan(&count), q.countSelectQuery("count(*)"), q.model)
	return count, err
}

func (q *Query) countSelectQuery(column string) *SelectQuery {
	return &SelectQuery{
		q:     q,
		count: column,
	}
}

// First sorts rows by primary key and selects the first row.
// It is a shortcut for:
//
//	q.OrderExpr("id ASC").Limit(1)
func (q *Query) First(column string) (Result, error) {
	return q.OrderExpr("? ASC", column).Limit(1).Scan()
}

// Last sorts rows by primary key and selects the last row.
// It is a shortcut for:
//
//	q.OrderExpr("id DESC").Limit(1)
func (q *Query) Last(column string) (Result, error) {
	return q.OrderExpr("? DESC", column).Limit(1).Scan()
}

func (q *Query) Scan(values ...interface{}) (_ Result, e error) {
	if q.stickyErr != nil {
		return nil, q.stickyErr
	}

	q.model, e = q.newModel(values)
	if e != nil {
		return nil, e
	}

	return q.query(q.ctx, q.model, NewSelectQuery(q))
}

func (q *Query) newModel(values []interface{}) (Model, error) {
	if len(values) > 0 {
		return newScanModel(values)
	}
	return q.model, nil
}

func (q *Query) query(ctx context.Context, model interface{}, query interface{}) (Result, error) {
	return q.db.QueryContext(ctx, model, query, q.model)
}

// SelectAndCount runs Select and Count in two goroutines,
// waits for them to finish and returns the result. If query limit is -1
// it does not select any data and only counts the results.
func (q *Query) SelectAndCount(values ...interface{}) (count int, firstErr error) {
	if q.stickyErr != nil {
		return 0, q.stickyErr
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	if q.limit >= 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := q.Scan(values...); e != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				mu.Unlock()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		count, e = q.Count()
		if e != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = e
			}
			mu.Unlock()
		}
	}()

	wg.Wait()
	return count, firstErr
}

// ForEach calls the function for each row returned by the query
// without loading all rows into the memory.
//
// Function can accept a struct, a pointer to a struct, an orm.Model,
// or values for the columns in a row. Function must return an error.
func (q *Query) ForEach(fn interface{}) (Result, error) {
	m := newFuncModel(fn)
	return q.Scan(m)
}

// Insert inserts the model.
func (q *Query) Insert(values ...interface{}) (_ Result, e error) {
	if q.stickyErr != nil {
		return nil, q.stickyErr
	}

	q.model, e = q.newModel(values)
	if e != nil {
		return nil, e
	}

	ctx := q.ctx

	query := NewInsertQuery(q)
	res, err := q.db.QueryContext(ctx, q.model, query)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Update updates the model.
func (q *Query) Update(scan ...interface{}) (Result, error) {
	return q.update(scan, false)
}

// Update updates the model omitting fields with zero values such as:
//   - empty string,
//   - 0,
//   - zero time,
//   - empty map or slice,
//   - byte array with all zeroes,
//   - nil ptr,
//   - types with method `IsZero() == true`.
func (q *Query) UpdateNotZero(scan ...interface{}) (Result, error) {
	return q.update(scan, true)
}

func (q *Query) update(values []interface{}, omitZero bool) (_ Result, e error) {
	if q.stickyErr != nil {
		return nil, q.stickyErr
	}

	q.model, e = q.newModel(values)
	if e != nil {
		return nil, e
	}

	c := q.ctx

	query := NewUpdateQuery(q, omitZero)
	res, e := q.db.QueryContext(c, q.model, query)
	if e != nil {
		return nil, e
	}

	return res, nil
}

// Delete forces delete of the model with deleted_at column.
func (q *Query) Delete() (_ Result, e error) {
	if q.stickyErr != nil {
		return nil, q.stickyErr
	}

	res, e := q.db.QueryContext(q.ctx, nil, NewDeleteQuery(q))
	if e != nil {
		return nil, e
	}

	return res, nil
}

// Exec is an alias for DB.Exec.
func (q *Query) Exec(query interface{}, params ...interface{}) (Result, error) {
	params = append(params, q.model)
	return q.db.ExecContext(q.ctx, query, params...)
}

// Query is an alias for DB.Query.
func (q *Query) Query(model, query interface{}, params ...interface{}) (Result, error) {
	params = append(params, q.model)
	return q.db.QueryContext(q.ctx, model, query, params...)
}

// CopyFrom is an alias from DB.CopyFrom.
func (q *Query) CopyFrom(r io.Reader, query interface{}, params ...interface{}) (Result, error) {
	params = append(params, q.model)
	return q.db.CopyFrom(r, query, params...)
}

// CopyTo is an alias from DB.CopyTo.
func (q *Query) CopyTo(w io.Writer, query interface{}, params ...interface{}) (Result, error) {
	params = append(params, q.model)
	return q.db.CopyTo(w, query, params...)
}

var _ QueryAppender = (*Query)(nil)

func (q *Query) AppendQuery(fmter QueryFormatter, b []byte) ([]byte, error) {
	return NewSelectQuery(q).AppendQuery(fmter, b)
}

// Exists returns true or false depending if there are any rows matching the query.
func (q *Query) Exists() (bool, error) {
	q = q.Clone() // copy to not change original query
	q.columns = []QueryAppender{SafeQuery("1")}
	q.order = nil
	q.limit = 1
	res, err := q.db.ExecContext(q.ctx, NewSelectQuery(q))
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (q *Query) Value() (*mapValue, error) {
	if q.stickyErr != nil {
		return nil, q.stickyErr
	}

	val := newMapValue()
	if _, e := q.db.QueryContext(
		q.ctx, newMapModel(&val.m), NewSelectQuery(q), q.model); e != nil {
		return nil, e
	}

	return val, nil
}

func (q *Query) hasTables() bool {
	return len(q.tables) > 0
}

func (q *Query) appendFirstTable(fmter QueryFormatter, b []byte) ([]byte, error) {
	if len(q.tables) > 0 {
		return q.tables[0].AppendQuery(fmter, b)
	}
	return b, nil
}

func (q *Query) appendTables(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	for i, f := range q.tables {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *Query) appendOtherTables(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	tables := q.tables
	if len(tables) > 0 {
		tables = tables[1:]
	}

	for i, f := range tables {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *Query) hasMultiTables() bool {
	return len(q.tables) >= 2
}

func (q *Query) appendColumns(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	for i, f := range q.columns {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *Query) mustAppendWhere(fmter QueryFormatter, b []byte) ([]byte, error) {
	if len(q.where) == 0 {
		err := errors.New(
			"pg: Update and Delete queries require Where clause")
		return nil, err
	}
	return q.appendWhere(fmter, b, q.where)
}

func (q *Query) appendUpdWhere(fmter QueryFormatter, b []byte) ([]byte, error) {
	return q.appendWhere(fmter, b, q.updWhere)
}

func (q *Query) appendWhere(
	fmter QueryFormatter, b []byte, where []queryWithSepAppender,
) (_ []byte, err error) {
	for i, f := range where {
		start := len(b)

		if i > 0 {
			b = f.AppendSep(b)
		}

		before := len(b)

		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}

		if len(b) == before {
			b = b[:start]
		}
	}
	return b, nil
}

func (q *Query) appendSet(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	b = append(b, " SET "...)
	for i, f := range q.set {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *Query) hasReturning() bool {
	if len(q.returning) == 0 {
		return false
	}
	if len(q.returning) == 1 {
		switch q.returning[0].query {
		case "null", "NULL":
			return false
		}
	}
	return true
}

func (q *Query) appendReturning(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	if !q.hasReturning() {
		return b, nil
	}

	b = append(b, " RETURNING "...)
	for i, f := range q.returning {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *Query) appendWith(fmter QueryFormatter, b []byte) (_ []byte, err error) {
	b = append(b, "WITH "...)
	for i, with := range q.with {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = types.AppendIdent(b, with.name, 1)
		b = append(b, " AS ("...)

		b, err = with.query.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}

		b = append(b, ')')
	}
	b = append(b, ' ')
	return b, nil
}
