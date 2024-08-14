package uweb

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
)

const (
	EndIndex        = -10
	OutlineEndIndex = -20
	CloseIndex      = -30
)

type HandlerFunc func(*Context)

type HandlerList []HandlerFunc

type HandlerWriter struct {
	status int           // 状态码
	header http.Header   // headers
	body   *bytes.Buffer // body
}

// Handler 上下文
type Context struct {
	uweb *Uweb // Uweb

	writer http.ResponseWriter // 原始的 writer
	Req    *http.Request       // 公开请求
	Writer *HandlerWriter      // 公开响应

	ctxStore     map[string]interface{} // 上下文存储
	ctxStoreLock *sync.RWMutex          // 上下文存储锁

	vpath       []uint8           // 魔法路径
	parmas      map[string]string // 路径参数
	index       int               // 当前执行的处理函数索引
	handlerList HandlerList       // 处理函数列表
}

func newWriterBufferContext() *HandlerWriter {
	return &HandlerWriter{
		status: 200,
		body:   &bytes.Buffer{},
	}
}

func (w *HandlerWriter) reset() {
	w.status = defaultStatusCode
	w.header = nil
	w.body.Reset()
}

func (w *HandlerWriter) Header() http.Header {
	return w.header
}

func (w *HandlerWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *HandlerWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

func (w *HandlerWriter) WriteJSON(v interface{}) error {
	w.header.Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w.body).Encode(v)
}

func (w *HandlerWriter) WriteHeader(code int) {
	w.status = code
}

func (uweb *Uweb) newContext() *Context {
	c := &Context{
		uweb: uweb,

		ctxStore:     nil,
		ctxStoreLock: &sync.RWMutex{},
		parmas:       nil,
		index:        0,
	}

	c.Writer = newWriterBufferContext()
	return c
}

func (c *Context) reset() {
	c.writer = nil
	c.Req = nil
	c.Writer.reset()

	c.vpath = nil
	c.parmas = nil
	c.ctxStore = nil
	c.index = 0
	c.handlerList = nil
}

func (c *Context) use(writer http.ResponseWriter, req *http.Request) {
	c.writer = writer
	c.Req = req

	c.Writer.header = writer.Header()
}

func (c *Context) Set(key string, value interface{}) {
	c.ctxStoreLock.Lock()
	defer c.ctxStoreLock.Unlock()

	if c.ctxStore == nil {
		c.ctxStore = make(map[string]interface{})
	}

	c.ctxStore[key] = value
}

func (c *Context) Get(key string) (value interface{}, ok bool) {
	c.ctxStoreLock.RLock()
	defer c.ctxStoreLock.RUnlock()

	if c.ctxStore == nil {
		return nil, false
	}

	value, ok = c.ctxStore[key]
	return
}

func (c *Context) Delete(key string) {
	c.ctxStoreLock.Lock()
	defer c.ctxStoreLock.Unlock()

	if c.ctxStore == nil {
		return
	}

	delete(c.ctxStore, key)
}

func (c *Context) Param(key string) string {
	if c.parmas != nil {
		return c.parmas[key]
	}

	c.parmas = make(map[string]string)
	vpath, path := strings.Split(string(c.vpath), "/"),
		strings.Split(c.Req.URL.Path, "/")

	if vpl, pl := len(vpath), len(path); vpl > pl {
		vpath = vpath[:pl]
	}

	for i := 0; i < len(vpath); i++ {
		if len(vpath[i]) < 1 {
			continue
		} else if vpath[i][0] == ':' {
			c.parmas[vpath[i][1:]] = path[i]
		} else if vpath[i][0] == '*' {
			c.parmas[vpath[i][1:]] = strings.Join(path[i:], "/")
			break
		}
	}

	return c.parmas[key]
}

// URL Query
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// POST Form Value
func (c *Context) PostForm(key string) string {
	_ = c.Req.ParseForm()
	return c.Req.FormValue(key)
}

// 获取当前请求的Cookit
func (c *Context) Cookie(key string) string {
	if v, _ := c.Req.Cookie(key); v != nil {
		return v.Value
	}

	return ""
}

func (c *Context) ReqBody() (body []byte) {
	if c.Req.Body != nil {
		body, _ = io.ReadAll(c.Req.Body)
	}
	c.Req.Body = io.NopCloser(bytes.NewBuffer(body))
	return body
}

// 设置状态码
// 也可以获取当前设置的状态码
// 加了魔法, 可以重复设置状态码
func (c *Context) Status(code ...int) int {
	if len(code) > 0 {
		c.Writer.WriteHeader(code[0])
	}

	return c.Writer.status
}

// 设置 header 的简单封装
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 占位符填充输出
// 内部封装了fmt.Sprintf
// 默认content-type为text/html
func (c *Context) Sprintf(code int, format string, values ...any) {
	c.Status(code)
	if c.Writer.Header().Get(HeaderContentType) == "" {
		c.SetHeader(HeaderContentType, "text/plain; charset=utf-8")
	}

	fmt.Fprintf(c.Writer, format, values...)
}

// 输出字符串
// 默认content-type为text/html
func (c *Context) String(code int, format string) {
	c.Status(code)
	if c.Writer.Header().Get(HeaderContentType) == "" {
		c.SetHeader(HeaderContentType, "text/plain; charset=utf-8")
	}

	_, _ = c.Writer.WriteString(format)
}

func (c *Context) Html(code int, html string) {
	c.Status(code)
	if c.Writer.Header().Get(HeaderContentType) == "" {
		c.SetHeader(HeaderContentType, "text/html; charset=utf-8")
	}

	_, _ = c.Writer.WriteString(html)
}

func (c *Context) JSON(code int, obj any) {
	c.SetHeader(HeaderContentType, "application/json; charset=utf-8")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) XML(code int, obj any) {
	c.SetHeader(HeaderContentType, "application/xml; charset=utf-8")
	c.Status(code)
	encoder := xml.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 输出字节数据
// 默认content-type为application/octet-stream
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	if c.Writer.Header().Get(HeaderContentType) == "" {
		c.SetHeader(HeaderContentType, "application/octet-stream")
	}

	_, _ = c.Writer.Write(data)
}

func (c *Context) File(code int, ffs fs.FS, filename string) {
	f, e := ffs.Open(filename)
	if e != nil && !errors.Is(e, fs.ErrNotExist) {
		http.Error(c.Writer, e.Error(), http.StatusBadRequest)
		return
	} else if e != nil && errors.Is(e, fs.ErrNotExist) {
		http.Error(c.Writer, e.Error(), http.StatusNotFound)
	}

	defer f.Close()

	c.Status(code)

	if c.Writer.Header().Get(HeaderContentType) == "" {
		c.SetHeader(HeaderContentType, "text/html;  charset=utf-8")
	}

	if _, e := io.Copy(c.Writer, f); e != nil {
		http.Error(c.Writer, e.Error(), http.StatusBadRequest)
	}
}

func (c *Context) WriteTo(w io.Writer) (int64, error) {
	return c.Writer.body.WriteTo(w)
}

func (c *Context) ResponseWriter(f func(w http.ResponseWriter)) {
	defer c.end(OutlineEndIndex)

	f(c.writer)
}

// 循环执行下一个HandlerFunc
func (c *Context) Next() {
	for c.index > -1 && c.index < len(c.handlerList) {
		func() {
			defer func() {
				if r := recover(); r != nil && c.index > -1 {
					panic(r)
				}
			}()

			c.index++
			c.handlerList[c.index-1](c)
		}()
	}
}

func (c *Context) Index() int {
	return c.index
}

func (c *Context) Clean() {
	c.Writer.reset()
	c.Writer.header = c.writer.Header()
}

func (c *Context) End() {
	c.end(EndIndex)
}

// 重置请求, 并跳过后续的HandlerFunc
// 将无任何输出, 并且浏览器显示连接已重置
// 但是仍然有响应头 "HTTP 1.1 400 Bad Request\r\nConnection: close"
func (c *Context) Close() {
	c.end(CloseIndex)
}

func (c *Context) end(index int) {
	c.index = index
	panic(http.ErrAbortHandler)
}
