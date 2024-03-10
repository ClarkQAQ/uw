package umap

import "sync"

type Mmap[K Hashable, V any] struct {
	m map[K]V
	l *sync.RWMutex
}

func NewMmap[K Hashable, V any]() *Mmap[K, V] {
	return &Mmap[K, V]{
		m: make(map[K]V),
		l: &sync.RWMutex{},
	}
}

func (h *Mmap[K, V]) Load(key K) (V, bool) {
	h.l.RLock()
	defer h.l.RUnlock()

	v, ok := h.m[key]
	return v, ok
}

func (h *Mmap[K, V]) Get(key K) V {
	v, _ := h.Load(key)
	return v
}

func (h *Mmap[K, V]) GetOrSet(key K, val V) (V, bool) {
	if val, ok := h.Load(key); ok {
		return val, !ok
	}

	h.l.Lock()
	defer h.l.Unlock()
	h.m[key] = val

	return val, true
}

func (s *Mmap[K, V]) Set(key K, val V) V {
	s.l.Lock()
	defer s.l.Unlock()

	s.m[key] = val
	return val
}

func (s *Mmap[K, V]) Range(f func(k K, v V) bool) {
	s.l.Lock()
	defer s.l.Unlock()

	for k, v := range s.m {
		if !f(k, v) {
			break
		}
	}
}

func (s *Mmap[K, V]) Delete(key K) {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.m, key)
}

func (s *Mmap[K, V]) Len() int {
	s.l.RLock()
	defer s.l.RUnlock()
	return len(s.m)
}

func (s *Mmap[K, V]) Clean() {
	s.l.Lock()
	defer s.l.Unlock()
	clear(s.m)
}
