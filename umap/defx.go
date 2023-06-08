package umap

import (
	"time"
)

type Defx[K Hashable, V any] struct {
	m        *Hmap[K, *DefxValue[V]]
	expire   time.Duration      // 默认过期时间
	gcTicker *time.Ticker       // gc定时器
	loader   func(K) (V, error) // 加载函数
	onExpire func(K, V)         // 过期回调
}

type DefxValue[V any] struct {
	value V
	exp   int64
}

// :defaultExpire: 默认过期时间
// :loadingDuration: 等待加载轮询时间
// :loader: 加载函数
func NewDefx[K Hashable, V any](expire, gcInterval time.Duration, loader func(K) (V, error)) *Defx[K, V] {
	if loader == nil {
		loader = func(K) (V, error) {
			var v V
			return v, nil
		}
	}

	if gcInterval < 1 {
		gcInterval = defaultGcInterval
	}

	d := &Defx[K, V]{
		m:        NewHmap[K, *DefxValue[V]](),
		expire:   expire,
		gcTicker: time.NewTicker(gcInterval),
		loader:   loader,
	}

	go d.gcRound()

	return d
}

func (d *Defx[K, V]) Close() {
	d.gcTicker.Stop()

	d.m.Range(func(k K, val *DefxValue[V]) bool {
		d.delete(k, val.value)
		return true
	})
}

func (d *Defx[K, V]) gcRound() {
	for range d.gcTicker.C {
		unixTimestamp := time.Now().Unix()
		d.m.Range(func(k K, val *DefxValue[V]) bool {
			if val.exp < unixTimestamp {
				d.delete(k, val.value)
			}

			return true
		})
	}
}

func (d *Defx[K, V]) SetOnExpire(cb func(K, V)) {
	d.onExpire = cb
}

func (d *Defx[K, V]) SetExpire(expire time.Duration) {
	d.expire = expire
}

func (d *Defx[K, V]) defaultValue() V {
	var v V
	return v
}

func (d *Defx[K, V]) load(key K) (_ *DefxValue[V], e error) {
	if val, ok := d.m.Load(key); ok {
		return val, nil
	}

	val := d.m.Set(key, &DefxValue[V]{
		value: d.defaultValue(),
		exp:   time.Now().Add(d.expire).Unix(),
	})

	val.value, e = d.loader(key)
	if e != nil {
		return nil, e
	}

	return val, nil
}

func (d *Defx[K, V]) Load(key K) (val V, e error) {
	v, e := d.load(key)
	if e != nil {
		return d.defaultValue(), e
	}

	return v.value, nil
}

func (d *Defx[K, V]) Range(cb func(K, V) bool) {
	d.m.Range(func(key K, val *DefxValue[V]) bool {
		return cb(key, val.value)
	})
}

func (d *Defx[K, V]) Delete(key K) {
	val, ok := d.m.Load(key)
	if !ok {
		return
	}

	d.delete(key, val.value)
}

func (d *Defx[K, V]) delete(key K, v V) {
	if d.onExpire != nil {
		d.onExpire(key, v)
	}

	d.m.Delete(key)
}
