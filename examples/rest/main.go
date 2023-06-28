package main

import (
	"encoding/json"
	"net/http"
	"time"

	"uw/ulog"
	"uw/urest"
	"uw/uweb"
)

type Services struct{}

func (*Services) AdminService() urest.Groupor {
	return urest.Group("/admin",
		new(AdminService), // 管理接口
	).Tags("管理员接口").Recover(func(c *urest.Context, e error) {
		c.Writer.Header().Set("Content-Type", "application/json")
		if e := json.NewEncoder(c.Writer).Encode(&JsonExit[interface{}]{
			Code:    1,
			Message: e.Error(),
			Data:    nil,
		}); e != nil {
			panic(e)
		}

		c.End()
	})
}

type AdminService struct{}

func (*AdminService) BaseService() urest.Groupor {
	return urest.Group("/base",
		new(BaseService), // 基础服务
	).Tags("基础服务")
}

func (*AdminService) AdminMiddleware() urest.Middlewareor {
	return urest.Middleware(func(c *urest.Context) {
		ulog.Debug("admin middleware: %v", c)
	})
}

type BaseService struct{}

func (*BaseService) UserService() urest.Groupor {
	return urest.Group("/user",
		new(UserService), // 用户管理
	).Tags("用户管理").Response(func(w http.ResponseWriter, resp any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if e := json.NewEncoder(w).Encode(&JsonExit[any]{
			Code:    0,
			Message: "success",
			Data:    resp,
		}); e != nil {
			panic(e)
		}
	}, func(fl []*urest.Field) []*urest.Field {
		ulog.Info("response fields: %v", fl)

		return []*urest.Field{{
			Name: "code",
			Type: urest.FieldTypeNumber,
		}, {
			Name: "message",
			Type: urest.FieldTypeString,
		}, {
			Name:     "data",
			Type:     urest.FieldTypeAny,
			Children: fl,
		}}
	})
}

type UserService struct{}

func (*UserService) Middleware() urest.Middlewareor {
	return urest.Middleware(func(c *urest.Context) {
	})
}

// type PostReq struct{}

// func (*UserService) Post() urest.Methodor {
// 	return urest.Method(func(c *urest.Context, req *struct {
// 		Name string `json:"name" form:"name" description:"名称"`
// 	},
// 	) (bool, error) {
// 		fmt.Println(req.Name)

// 		return req.Name == "admin", nil
// 	})
// }

type JsonExit[T any] struct {
	Code    int    `json:"code" name:"状态码"`
	Message string `json:"message" name:"消息"`
	Data    T      `json:"data" name:"数据"`
}

type GetReq struct {
	Id    int64   `json:"id" key:"id" name:"编号"`
	Limit int64   `json:"limit" key:"limit" name:"单页数量"`
	Page  int64   `json:"page" key:"page" name:"页码"`
	Sort  *string `json:"sort" key:"sort" default:"id" enum:"id:编号,date:日期" name:"排序字段"`
	Data  *GetReqData
}

type GetReqData struct {
	Id string `json:"id" header:"Authorization" name:"编号"`
}

type GetResp struct {
	Id   int64          `json:"id" name:"编号"`
	Name string         `json:"name" name:"名称"`
	Data []*GetRespData `json:"data" name:"数据"`
}

type GetRespData struct {
	Id int64 `json:"id" name:"编号"`
}

func (*UserService) Post() urest.Methodor {
	return urest.Method(func(c *urest.Context, req *GetReq) (*GetResp, error) {
		return &GetResp{}, nil
	}).Summary("获取信息").Detail("获取用户信息")
}

func main() {
	tr := ulog.Timer()
	r, e := urest.NewRest(new(Services))
	if e != nil {
		ulog.Fatal("rest error: %s", e)
	}
	tr.End("rest")

	for _, v := range r.Invoke() {
		ulog.Debug("path: %s, value: %#v\n", v.Path(), v)
	}

	t := uweb.New()

	t.Use(func(c *uweb.Context) {
		v := time.Now()
		c.Next()

		ulog.Printf("status: %d, url: %s, time: %s", c.Status(), c.Req.RequestURI, time.Since(v))
	})

	r.BindUweb(t.Group)

	t.Get("/api", func(c *uweb.Context) {
		c.JSON(200, &JsonExit[any]{
			Code:    0,
			Message: "success",
			Data:    r.Invoke()[0].HandlerField(),
		})
	})

	for _, v := range t.DumpRoute() {
		ulog.Debug("path: %s, method: %#v", v.Path, v.Value)
	}

	if _, e := t.ServeAddr(":8080"); e != nil {
		ulog.Fatal("serve addr error: %s", e)
	}
}
