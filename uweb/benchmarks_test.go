// Copyright 2017 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// This file is part of the gin, modified by uweb.
// source: https://github.com/gin-gonic/gin/blob/master/benchmarks_test.go

package uweb_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"uw/uweb"
)

func BenchmarkOneRoute(B *testing.B) {
	router := uweb.New()
	router.Get("/ping", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/ping")
}

func BenchmarkRecoveryMiddleware(B *testing.B) {
	router := uweb.New()
	router.Use(recovery())
	router.Get("/", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/")
}

func BenchmarkLoggerMiddleware(B *testing.B) {
	router := uweb.New()
	router.Use(timer(newMockWriter().WriteString))
	router.Get("/", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/")
}

func BenchmarkManyHandlers(B *testing.B) {
	router := uweb.New()
	router.Use(recovery(), timer(newMockWriter().WriteString))
	router.Use(func(c *uweb.Context) {})
	router.Use(func(c *uweb.Context) {})
	router.Get("/ping", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/ping")
}

func Benchmark5Params(B *testing.B) {
	router := uweb.New()
	router.Use(func(c *uweb.Context) {})
	router.Get("/param/:param1/:params2/:param3/:param4/:param5", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/param/path/to/parameter/john/12345")
}

func BenchmarkOneRouteJSON(B *testing.B) {
	router := uweb.New()
	data := struct {
		Status string `json:"status"`
	}{"ok"}
	router.Get("/json", func(c *uweb.Context) {
		c.JSON(http.StatusOK, data)
	})
	runRequest(B, router, "GET", "/json")
}

func BenchmarkOneRoutePrintf(B *testing.B) {
	router := uweb.New()
	router.Get("/html", func(c *uweb.Context) {
		c.Sprintf(http.StatusOK, "<html><body><h1>%s</h1></body></html>", "hola")
	})
	runRequest(B, router, "GET", "/html")
}

func BenchmarkOneRouteSet(B *testing.B) {
	router := uweb.New()
	router.Get("/ping", func(c *uweb.Context) {
		c.Set("key", "value")
	})
	runRequest(B, router, "GET", "/ping")
}

func BenchmarkOneRouteString(B *testing.B) {
	router := uweb.New()
	router.Get("/text", func(c *uweb.Context) {
		c.String(http.StatusOK, "this is a plain text")
	})
	runRequest(B, router, "GET", "/text")
}

func BenchmarkManyRoutesFist(B *testing.B) {
	router := uweb.New()
	router.Any("/ping", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/ping")
}

func BenchmarkManyRoutesLast(B *testing.B) {
	router := uweb.New()
	router.Any("/ping", func(c *uweb.Context) {})
	runRequest(B, router, "OPTIONS", "/ping")
}

func Benchmark404(B *testing.B) {
	router := uweb.New()
	router.Any("/something", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/ping")
}

func Benchmark404Many(B *testing.B) {
	router := uweb.New()
	router.Get("/", func(c *uweb.Context) {})
	router.Get("/path/to/something", func(c *uweb.Context) {})
	router.Get("/post/:id", func(c *uweb.Context) {})
	router.Get("/view/:id", func(c *uweb.Context) {})
	router.Get("/favicon.ico", func(c *uweb.Context) {})
	router.Get("/robots.txt", func(c *uweb.Context) {})
	router.Get("/delete/:id", func(c *uweb.Context) {})
	router.Get("/user/:id/:mode", func(c *uweb.Context) {})
	runRequest(B, router, "GET", "/viewfake")
}

type mockWriter struct {
	headers http.Header
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		http.Header{},
	}
}

func (m *mockWriter) Header() (h http.Header) {
	return m.headers
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockWriter) WriteHeader(int) {}

func runRequest(B *testing.B, r *uweb.Uweb, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	w := newMockWriter()
	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func recovery() uweb.HandlerFunc {
	return func(c *uweb.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.String(http.StatusInternalServerError, "Internal Server Error")
			}

			if c.Index() < -1 {
				panic(nil)
			}
		}()

		c.Next()
	}
}

func timer(w func(s string) (int, error)) uweb.HandlerFunc {
	return func(c *uweb.Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()

		w(fmt.Sprintf("[%d] [%s] %s %s\r\n", c.Status(),
			c.Req.Method, c.Req.RequestURI, time.Since(t)))
	}
}
