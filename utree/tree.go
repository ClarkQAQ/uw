package utree

import "unsafe"

const (
	asterisk = '*' // 通配符
	colon    = ':' // 冒号 (路径参数)
	slash    = '/' // 斜杠
)

// 路由树
type Tree[T any] struct {
	children [256]*Tree[T] // 子路由节点
	value    T             // 路由方法 (methodTyp 类型)
	vpath    []uint8       // 注册的路由路径
}

func New[T any]() *Tree[T] {
	return &Tree[T]{}
}

// 斜杠读取器
// 读取到下一个斜杠或者结束, 然后返回下标位置
func slashReader(a []uint8, i int) int {
	for ; i < len(a) && a[i] != slash; i++ {
	}

	return i
}

// 设置路由
func (t *Tree[T]) Set(route string, value T) {
	a := toByte(route)

	for i := 0; i < len(a); i++ {
		if t.children[a[i]] == nil {
			t.children[a[i]] = &Tree[T]{}
		}

		t = t.children[a[i]]

		// 判断是否为路径参数 (例如: :name *name)
		if a[i] == colon {
			i = slashReader(a, i)
		} else if a[i] == asterisk {
			break
		}
	}

	t.value = value
	t.vpath = a
}

func (t *Tree[T]) get(route string) *Tree[T] {
	a := toByte(route)

	for i := 0; i < len(a); i++ {
		switch {
		case t.children[asterisk] != nil:
			return t.children[asterisk]
		case t.children[colon] != nil:
			t = t.children[colon]
			i = slashReader(a, i)
		case t.children[a[i]] == nil:
			return nil
		default:
			t = t.children[a[i]]
		}
	}

	return t
}

// 获取路由
func (t *Tree[T]) Get(route string) (T, []uint8) {
	if t = t.get(route); t != nil {
		return t.value, t.vpath
	}

	var empty T
	return empty, nil
}

func (t *Tree[T]) Delete(route string) {
	t = t.get(route)
	if t != nil {
		var empty T
		t.value = empty
		t.vpath = nil
	}
}

func (t *Tree[T]) Jump(route string) *Tree[T] {
	return t.get(route)
}

// 转换为字节切片
func toByte(s string) []uint8 {
	return *(*[]uint8)(unsafe.Pointer(&s))
}

// 以下为转换为字节切片的另一种方式, 但是效率较低, 使用他将会比前者慢 4 ns/op
// func toByte(s string) []uint8 {
// 	return []uint8(s)
// }

type DumpValue[T any] struct {
	Value T
	Path  string
}

// 递归遍历路由树并写入 slice
func (t *Tree[T]) Dump() []*DumpValue[T] {
	m := []*DumpValue[T]{}

	for i := 0; i < 256; i++ {
		if t.children[i] == nil {
			continue
		}

		if t.children[i].vpath != nil {
			m = append(m, &DumpValue[T]{
				Path:  string(t.children[i].vpath),
				Value: t.children[i].value,
			})
		}

		m = append(m, t.children[i].Dump()...)
	}

	return m
}
