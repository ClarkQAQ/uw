package utree

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

var defaultSeparator = "/" // 默认路径分隔符

type Value struct{}

func TestRouteTree_Get_Set_Dump(t *testing.T) {
	tree := New[*Value]()
	handlerList := map[string]bool{}

	testGetHandler := func(path, target string, has bool) {
		handlerList[path] = true
		tree.Set(path, &Value{})
		handler, vpath := tree.Get(target)
		if handler == nil && has {
			t.Fatalf("handler must not be nil, path: %s, target: %s", path, target)
		} else if handler != nil && !has {
			t.Fatalf("handler must be nil, path: %s, target: %s", path, target)
		}

		fmt.Printf("test get target: %s, vpath: %s\n", target, vpath)
	}

	testGetHandler(string([]byte{255}), string([]byte{255}), true)
	testGetHandler("/�/ÿ", "/�/ÿ", true)
	testGetHandler("/api/qaq/1", "/api/qaq/1", true)
	testGetHandler("/api/qaq/1/qwq", "/api/qaq/1/qwq", true)
	testGetHandler("GET@/api/qaq/1/qwq/2", "GET@/api/qaq/1/qwq/2", true)
	testGetHandler("/api/colon/:qwq/oxo", "/api/colon/1/oxo", true)
	testGetHandler("/api/asterisk/*qaq/qwq", "/api/asterisk/1/2/3/qwq", true)
	testGetHandler("/api/qqqqqq", "/api/aaaaaa", false)

	{
		m := tree.Dump()
		if len(m) < 1 {
			t.Fatal("export must not be empty")
		}

		for _, v := range m {
			delete(handlerList, v.Path)
			fmt.Printf("dump path: %s, value: %#v\n", v.Path, v.Value)
		}

		if len(handlerList) > 0 {
			fmt.Println("handlerList:", handlerList)
			t.Fatal("handlerList must be empty")
		}
	}
}

func TestRouteTree_Delete(t *testing.T) {
	tree := New[*Value]()

	tree.Set("/api/qaq/1", &Value{})

	if v, _ := tree.Get("/api/qaq/1"); v == nil {
		t.Fatal("handler must not be nil")
	}

	tree.Delete("/api/qaq/1")

	if v, _ := tree.Get("/api/qaq/1"); v != nil {
		t.Fatal("handler must be nil")
	}
}

func TestRouteTree_Move(t *testing.T) {
	tree := New[*Value]()

	tree.Set("/api/qaq/1", &Value{})

	tree1 := tree.Move("/api")

	if tree1 == nil {
		t.Fatal("tree1 must not be nil")
	}

	if v, _ := tree1.Get("/qaq/1"); v == nil {
		t.Fatal("handler must not be nil")
	}
}

type RandomString struct {
	mu     sync.Mutex
	r      *rand.Rand
	layout string
}

var Numeric = &RandomString{
	layout: "0123456789",
	r:      rand.New(rand.NewSource(time.Now().UnixNano())),
	mu:     sync.Mutex{},
}

func (c *RandomString) Generate(n int) []byte {
	c.mu.Lock()
	b := make([]byte, n)
	length := len(c.layout)
	for i := 0; i < n; i++ {
		idx := c.r.Intn(length)
		b[i] = c.layout[idx]
	}
	c.mu.Unlock()
	return b
}

func (c *RandomString) Intn(n int) int {
	c.mu.Lock()
	x := c.r.Intn(n)
	c.mu.Unlock()
	return x
}

// fork by: https://github.com/lxzan/uRouter/blob/main/trie_test.go#L68
func BenchmarkRouteTree_Get(b *testing.B) {
	count := 1024
	segmentLen := 2
	tree := &Tree[*Value]{}
	r := Numeric
	for i := 0; i < count; i++ {
		idx := r.Intn(4)
		var list []string
		for j := 0; j < 4; j++ {
			ele := string(r.Generate(segmentLen))
			if j == idx {
				ele = ":" + ele
			}
			list = append(list, ele)
		}
		tree.Set(strings.Join(list, defaultSeparator), &Value{})
	}

	var paths []string
	for i := 0; i < count; i++ {
		path := r.Generate(12)
		path[0], path[3], path[6], path[9] = defaultSeparator[0], defaultSeparator[0], defaultSeparator[0], defaultSeparator[0]
		paths = append(paths, string(path))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i&(count-1)]
		tree.Get(path)
	}
}

func BenchmarkMap_Get(b *testing.B) {
	count := 1024
	segmentLen := 2
	m := map[string]*Value{}
	r := Numeric
	for i := 0; i < count; i++ {
		idx := r.Intn(4)
		var list []string
		for j := 0; j < 4; j++ {
			ele := string(r.Generate(segmentLen))
			if j == idx {
				ele = ":" + ele
			}
			list = append(list, ele)
		}
		m[strings.Join(list, defaultSeparator)] = &Value{}
	}

	var paths []string
	for i := 0; i < count; i++ {
		path := r.Generate(12)
		path[0], path[3], path[6], path[9] = defaultSeparator[0], defaultSeparator[0], defaultSeparator[0], defaultSeparator[0]
		paths = append(paths, string(path))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i&(count-1)]
		_ = m[path]
	}
}

func BenchmarkRouteTree_Set(b *testing.B) {
	count := 1024
	tree := &Tree[*Value]{}
	r := Numeric
	m := &Value{}

	var paths []string
	for i := 0; i < count; i++ {
		path := r.Generate(12)
		path[0], path[3], path[6], path[9] = defaultSeparator[0], defaultSeparator[0], defaultSeparator[0], defaultSeparator[0]
		paths = append(paths, string(path))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i&(count-1)]
		tree.Set(path, m)
	}
}

func BenchmarkRouteMap_Set(b *testing.B) {
	count := 1024
	m := map[string]*Value{}
	r := Numeric
	val := &Value{}

	var paths []string
	for i := 0; i < count; i++ {
		path := r.Generate(12)
		path[0], path[3], path[6], path[9] = defaultSeparator[0], defaultSeparator[0], defaultSeparator[0], defaultSeparator[0]
		paths = append(paths, string(path))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i&(count-1)]
		m[path] = val
	}
}
