package utree

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	defaultSeparator           = "/" // 默认路径分隔符
	globalParams, globalValues = generateKeyValuePath()
)

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

func generateKeyValuePath() ([1024]string, [1024]string) {
	segmentLen := 2
	r := Numeric

	params, values := [1024]string{}, [1024]string{}

	for i := 0; i < 1024; i++ {
		idx := r.Intn(4)
		var list []string
		for j := 0; j < 4; j++ {
			ele := string(r.Generate(segmentLen))
			if j == idx {
				ele = ":" + ele
			}
			list = append(list, ele)
		}

		params[i] = strings.Join(list, defaultSeparator)
	}

	for i := 0; i < 1024; i++ {
		path := r.Generate(12)
		path[0], path[3], path[6], path[9] = defaultSeparator[0], defaultSeparator[0], defaultSeparator[0], defaultSeparator[0]
		values[i] = string(path)
	}

	return params, values
}

type Value struct{}

func TestTree_Get_Set(t *testing.T) {
	tree := New[*Value]()

	testGetHandler := func(path, target string, has bool) {
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
	testGetHandler("/�/:qaq/ÿ", "/�/123/ÿ", true)
	testGetHandler("/�/:qaq/ÿ", "/�/123/�", false)
	testGetHandler("/api/qaq/1", "/api/qaq/1", true)
	testGetHandler("/api/qaq/1/qwq", "/api/qaq/1/qwq", true)
	testGetHandler("GET@/api/qaq/1/qwq/2", "GET@/api/qaq/1/qwq/2", true)
	testGetHandler("/api/colon/:qwq/oxo", "/api/colon/1/oxo", true)
	testGetHandler("/api/asterisk/*qaq/qwq", "/api/asterisk/1/2/3/qwq", true)
	testGetHandler("/api/qqqqqq", "/api/aaaaaa", false)
}

func TestTree_Delete(t *testing.T) {
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

func TestTree_Move(t *testing.T) {
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

func TestTree_Dump(t *testing.T) {
	tree := New[*Value]()

	for i := 0; i < 256; i++ {
		tree.Set(string([]byte{byte(i)}), &Value{})
	}

	m := tree.Dump()
	if len(m) < 1 {
		t.Fatal("export must not be empty")
	}

	list := make(map[string]bool)
	for i := 0; i < len(m); i++ {
		list[m[i].Path] = true
	}

	for i := 0; i < 256; i++ {
		if !list[string([]byte{byte(i)})] {
			t.Fatal("export must not be empty")
		}
	}
}

// fork by: https://github.com/lxzan/uRouter/blob/main/trie_test.go#L68
func BenchmarkTree_Get(b *testing.B) {
	tree := &Tree[*Value]{}
	for i := 0; i < len(globalParams); i++ {
		tree.Set(globalParams[i], &Value{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalValues[i&(1023)]
		tree.Get(path)
	}
}

func BenchmarkStaticMap_Get(b *testing.B) {
	m := map[string]*Value{}
	for i := 0; i < len(globalParams); i++ {
		m[globalParams[i]] = &Value{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalValues[i&(1023)]
		_ = m[path]
	}
}

func BenchmarkTree_Set(b *testing.B) {
	tree, val := &Tree[*Value]{}, &Value{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalParams[i&(1023)]
		tree.Set(path, val)
	}
}

func BenchmarkStaticMap_Set(b *testing.B) {
	m, val := map[string]*Value{}, &Value{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalValues[i&(1023)]
		m[path] = val
	}
}

func BenchmarkTree_Delete(b *testing.B) {
	tree := &Tree[*Value]{}
	for i := 0; i < len(globalParams); i++ {
		tree.Set(globalParams[i], &Value{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalParams[i&(1023)]
		tree.Delete(path)
	}
}

func BenchmarkStaticMap_Delete(b *testing.B) {
	m := map[string]*Value{}
	for i := 0; i < len(globalParams); i++ {
		m[globalParams[i]] = &Value{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := globalValues[i&(1023)]
		delete(m, path)
	}
}
