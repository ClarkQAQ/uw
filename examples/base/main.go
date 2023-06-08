package main

import (
	"fmt"
	"time"

	"uw/uweb"
)

func main() {
	t := uweb.New()

	t.Use(func(c *uweb.Context) {
		v := time.Now()
		c.Next()

		fmt.Printf("status: %d, url: %s, time: %s\n", c.Status(), c.Req.RequestURI, time.Since(v))
	})

	// t.Use(func(c *uweb.Context) {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			c.String(http.StatusInternalServerError, "Internal Server Error")
	// 			c.End()
	// 		}
	// 	}()

	// 	c.Next()
	// })

	// Hello World
	// t.Get("/:path/:qwq/index.html", func(c *uweb.Context) {
	// 	c.String(200, fmt.Sprintf("Hello, World! %s, %s", c.Param("path"), c.Param("qwq")))

	// 	// c.Close()
	// })

	// 恐慌测试
	t.Get("/recover", func(c *uweb.Context) {
		panic("panic")
	})

	// 阻塞测试
	t.Get("/block", func(c *uweb.Context) {
		time.Sleep(5 * time.Second)
		c.String(200, "Hello, World!")
	})

	// Close
	t.Get("/close", func(c *uweb.Context) {
		c.String(200, "Hello, World!")
		c.Close()
	})

	// End
	t.Get("/end", func(c *uweb.Context) {
		c.String(200, "Hello, World!")
		c.End()
	})

	// 方法路由测试
	t.Object("/test", &TestApi{})

	// 导出路由
	for _, v := range t.DumpRoute() {
		fmt.Printf("path: %s, method: %#v\n", v.Path, v.Value)
	}

	if _, e := t.ServeAddr(":8080"); e != nil {
		panic(fmt.Sprintf("serve error: %s", e))
	}
}

type TestApi struct{}

func (t *TestApi) Get(c *uweb.Context) {
	c.String(200, "Hello, Get!")
}

func (t *TestApi) Post(c *uweb.Context) {
	c.String(200, "Hello, Post!")
}
