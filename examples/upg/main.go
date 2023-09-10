package main

import (
	"context"
	"time"

	"uw/ulog"
	"uw/upg"
	"uw/upg/extra/upgbig"
	"uw/upg/extra/upgdebug"
)

type Account struct {
	Id        *upgbig.Int
	Name      string
	Emails    []string
	DeletedAt *time.Time `db:"deleted_at"`
}

func main() {
	db := upg.Connect(&upg.Options{
		Addr:      "127.0.0.1:15432",
		User:      "postgres",
		Password:  "clarkqaq",
		Database:  "postgres",
		TLSConfig: nil,
	})
	defer db.Close()

	db.AddQueryHook(upgdebug.NewQueryHook(
		upgdebug.WithVerbose(true),
	))

	l := db.Listen(context.Background(), "test")
	defer l.Close()
	go func(l *upg.Listener) {
		for n := range l.Channel() {
			ulog.Info("notify: %s, %s", n.Channel, n.Payload)
		}
	}(l)

	if _, e := db.Exec("NOTIFY test, ?", "hello world!"); e != nil {
		ulog.Fatal("send notify: %s", e)
	}

	if _, e := db.Exec(`
	DROP TABLE IF EXISTS "account";
	CREATE TABLE "account"(
		"id" BIGSERIAL PRIMARY KEY,
		"name" TEXT,
		"emails" JSONB,
		"deleted_at" TIMESTAMP WITHOUT TIME ZONE
	);`); e != nil {
		ulog.Fatal("create table: %s", e)
	}

	{
		mdata := map[string]interface{}{
			// Id:     upgbig.FromInt64(1),
			"name":   "admin",
			"emails": []string{"admin1@admin", "admin2@admin"},
		}

		if _, e := db.Table("account").Returning("id").Insert(&mdata); e != nil {
			ulog.Fatal("insert row: %s", e)
		}

		ulog.Info("insert data: %+v", mdata)
	}
	{
		mdata := map[string]interface{}{
			// Id:     upgbig.FromInt64(1),
			"name":   "admin",
			"emails": []string{"admin1@admin", "admin2@admin"},
		}

		if _, e := db.Table("account").Returning("id").Insert(&mdata); e != nil {
			ulog.Fatal("insert row: %s", e)
		}

		ulog.Info("insert data: %+v", mdata)
	}

	cdb := db.Table("account", "a").Where("id = ?", 1).Limit(1)

	data := &Account{}
	if _, e := cdb.Clone().ColumnExpr("a.id AS id, a.name").Scan(data); e != nil {
		ulog.Fatal("select row: %s", e)
	} else {
		ulog.Info("account struct: id: %d, name: %s, emails: %v, deleted_at: %v",
			data.Id.ToInt64(), data.Name, data.Emails, data.DeletedAt)
	}

	if _, e := cdb.Clone().Set(`"name" = ?`, "root").Update(); e != nil {
		ulog.Fatal("update row: %s", e)
	}

	if val, e := cdb.Clone().Value(); e != nil {
		ulog.Fatal("select row: %s", e)
	} else {
		ulog.Info("account value: id: %d name: %s, emails: %s, deleted_at: %v",
			val.Int64("id"), val.String("name"), val.Value("emails", "None"), val.Time("deleted_at", time.Now()))
	}

	if val, e := cdb.Clone().ColumnExpr(`"a"."id" AS "account_id"`).Value(); e != nil {
		ulog.Fatal("select row: %s", e)
	} else {
		ulog.Info("account value: id: %d name: %s, emails: %s, deleted_at: %v",
			val.Int64("account_id"), val.String("name"), val.Value("emails", "None"), val.Time("deleted_at", time.Now()))
	}
}
