package umap

import (
	"uw/pkg/hashmap"
)

type Hmap[K Hashable, V any] struct {
	m *hashmap.Map[K, V]
}

func NewHmap[K Hashable, V any]() *Hmap[K, V] {
	return &Hmap[K, V]{hashmap.New[K, V]()}
}

func (h *Hmap[K, V]) Load(key K) (V, bool) {
	return h.m.Get(key)
}

func (s *Hmap[K, V]) Get(key K) V {
	v, _ := s.Load(key)
	return v
}

func (s *Hmap[K, V]) GetOrSet(key K, val V) (V, bool) {
	return s.m.GetOrInsert(key, val)
}

func (s *Hmap[K, V]) Set(key K, val V) V {
	s.m.Set(key, val)
	return val
}

func (s *Hmap[K, V]) Range(f func(k K, v V) bool) {
	s.m.Range(func(key K, value V) bool {
		return f(key, value)
	})
}

func (s *Hmap[K, V]) Delete(key K) {
	s.m.Del(key)
}

func (s *Hmap[K, V]) Len() int {
	return s.m.Len()
}

func (s *Hmap[K, V]) Clean() {
	s.m = hashmap.New[K, V]()
}
