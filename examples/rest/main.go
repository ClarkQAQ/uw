package main

import (
	"context"
	"fmt"

	"uw/urest"
)

type Services struct{}

func (*Services) AdminService() urest.Groupor {
	return urest.Group("/admin",
		new(AdminService), // 管理接口
	).Tags("管理员接口")
}

type AdminService struct{}

func (*AdminService) BaseService() urest.Groupor {
	return urest.Group("/base",
		new(BaseService), // 基础服务
	).Tags("基础服务")
}

func (*AdminService) AdminMiddleware() urest.Middlewareor {
	return urest.Middleware(func(c *urest.Context) {
		fmt.Println("QAQ")
	})
}

type BaseService struct{}

func (*BaseService) UserService() urest.Groupor {
	return urest.Group("/user",
		new(UserService), // 用户管理
	).Tags("用户管理")
}

type UserService struct{}

func (*UserService) Middleware() urest.Middlewareor {
	return urest.Middleware(func(c *urest.Context) {
	})
}

type PostReq struct{}

func (*UserService) Post() urest.Methodor {
	return urest.Method(func(ctx context.Context, req *struct {
		Name string `json:"name" form:"name" description:"名称"`
	},
	) (bool, error) {
		fmt.Println(req.Name)

		return req.Name == "admin", nil
	})
}

type JsonExit[T any] struct {
	Code    int
	Message string
	Data    T
}

type GetReq struct {
	Id    int64  `json:"id" form:"id" query:"id" validate:"required" description:"编号"`
	Limit int64  `json:"limit" form:"limit" query:"limit" description:"单页数量"`
	Page  int64  `json:"page" form:"page" query:"page" description:"页码"`
	Sort  string `json:"sort" form:"sort" query:"sort" default:"id" enum:"id,date" description:"排序字段"`
}

type GetResp struct {
	Id   int64  `json:"id" description:"编号"`
	Name string `json:"name" description:"名称"`
}

func (*UserService) Get() urest.Methodor {
	return urest.Method(func(ctx context.Context, req *GetReq) (*JsonExit[[]*GetResp], error) {
		return &JsonExit[[]*GetResp]{
			Code:    0,
			Message: "success",
			Data:    []*GetResp{{Id: 1, Name: "admin"}},
		}, nil
	}).Summary("获取信息").Description("获取用户信息")
}

func main() {
	mps, e := urest.Invoke(new(Services))
	if e != nil {
		panic(e)
	}

	for _, v := range mps {
		fmt.Printf("path: %s, value: %#v\n", v.Path, v)
	}

	// t := uweb.New()

	// urest.Bind(t.Group, new(Services))

	// 导出路由
	// for _, v := range t.DumpRoute() {
	//     fmt.Printf("path: %s, method: %#v\n", v.Path, v.Value)
	// }
}
