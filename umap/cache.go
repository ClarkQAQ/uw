package umap

import (
	"time"
)

var defaultGcInterval = 60 * time.Second

type Cache[K Hashable, V any] struct {
	h        *Hmap[K, *Item[V]]
	gcTicker *time.Ticker
}

type Item[V any] struct {
	Value  V
	Expire int64
}

func NewCache[K Hashable, V any](gcInterval time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		h: NewHmap[K, *Item[V]](),
	}

	if gcInterval < 1 {
		gcInterval = defaultGcInterval
	}

	c.gcTicker = time.NewTicker(gcInterval)
	go c.gcRound()

	return c
}

func (c *Cache[K, V]) gcRound() {
	for range c.gcTicker.C {
		unixTimestamp := time.Now().Unix()
		c.h.Range(func(k K, v *Item[V]) bool {
			if v.Expire > 0 && v.Expire <= unixTimestamp {
				c.h.Delete(k)
			}
			return true
		})
	}
}

func (c *Cache[K, V]) Set(key K, value V, expire time.Duration) V {
	item := &Item[V]{
		Value:  value,
		Expire: 0,
	}

	if expire > 0 {
		item.Expire = time.Now().Add(expire).Unix()
	}

	c.h.Set(key, item)
	return value
}

func (c *Cache[K, V]) getItem(key K) *Item[V] {
	v, ok := c.h.Load(key)
	if !ok {
		return nil
	}

	if v.Expire > 0 && v.Expire <= time.Now().Unix() {
		return nil
	}

	return v
}

func (c *Cache[K, V]) defaultValue() V {
	var v V
	return v
}

func (c *Cache[K, V]) Get(key K) V {
	if v := c.getItem(key); v != nil {
		return v.Value
	}

	return c.defaultValue()
}

func (c *Cache[K, V]) Load(key K) (V, int64, bool) {
	if v := c.getItem(key); v != nil {
		return v.Value, v.Expire, true
	}

	return c.defaultValue(), 0, false
}

func (c *Cache[K, V]) Delete(key K) {
	c.h.Delete(key)
}

func (m *Cache[K, V]) Range(f func(k K, v V) bool) {
	unixTimestamp := time.Now().Unix()
	m.h.Range(func(k K, v *Item[V]) bool {
		if v.Expire > 0 && v.Expire <= unixTimestamp {
			return true
		}

		return f(k, v.Value)
	})
}

func (m *Cache[K, V]) Clean() {
	m.h.Clean()
}

func (m *Cache[K, V]) Close() {
	m.gcTicker.Stop()
	m.h.Clean()
}
