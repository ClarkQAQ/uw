package upg

import (
	"bytes"
	"strconv"

	"uw/upg/internal"
	"uw/upg/orm"
)

// Result summarizes an executed SQL command.
type Result = orm.Result

// A result summarizes an executed SQL command.
type result struct {
	model orm.Model

	affected int
	returned int
}

var _ Result = (*result)(nil)

func (res *result) parse(b []byte) error {
	res.affected = -1

	ind := bytes.LastIndexByte(b, ' ')
	if ind == -1 {
		return nil
	}

	s := internal.BytesToString(b[ind+1 : len(b)-1])

	affected, err := strconv.Atoi(s)
	if err == nil {
		res.affected = affected
	}

	return nil
}

func (res *result) Model() orm.Model {
	if res != nil {
		return res.model
	}
	return nil
}

func (res *result) RowsAffected() int {
	if res != nil {
		return res.affected
	}

	return -1
}

func (res *result) RowsReturned() int {
	if res != nil {
		return res.returned
	}
	return -1
}
