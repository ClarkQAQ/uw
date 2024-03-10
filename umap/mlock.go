package umap

import (
	"context"
)

type MLock[K Hashable] struct {
	h Mapper[K, MLockValue]
}

type MLockValue (chan bool)

// 初始化一个新的锁实例
func NewLocker[K Hashable](mapper Mapper[K, MLockValue]) *MLock[K] {
	return &MLock[K]{NewHmap[K, MLockValue]()}
}

// 初始化一个新的锁实例, 但是使用自定义的 Mapper
func NewLockerWithMapper[K Hashable](mapper Mapper[K, MLockValue]) *MLock[K] {
	return &MLock[K]{mapper}
}

// 用于内部获取chanel
// 如果指定的key不存在，则会创建一个新的chanel
func (m *MLock[K]) getLocker(k K) MLockValue {
	if locker, ok := m.h.Load(k); ok {
		return locker
	}

	// 初始化一个新的chanel
	m.h.Set(k, make(MLockValue, 1))
	return m.getLocker(k)
}

// 普通加锁
// 没有超时时间, 如果一直没能等到解锁，则永远阻塞或者deadlock, recover也将无法捕获
func (m *MLock[K]) Lock(k K) {
	_ = m.LockWithContext(context.Background(), k)
}

// 是否已经被锁定
func (m *MLock[K]) IsLocked(k K) bool {
	return len(m.getLocker(k)) > 0
}

// 带context的加锁
// 为了解决deadlock问题
func (m *MLock[K]) LockWithContext(ctx context.Context, k K) error {
	select {
	case m.getLocker(k) <- true:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func unlock(locker MLockValue) {
	if len(locker) > 0 {
		<-locker
	}
}

// 解锁
// 可以重复调用
func (m *MLock[K]) Unlock(k K) {
	unlock(m.getLocker(k))
}

// 释放一个锁
// 如果锁被锁定, 则会被解锁
func (m *MLock[K]) Release(k K) {
	m.Unlock(k)
	m.h.Delete(k)
}

// 释放所有锁
// 释放前会解锁所有锁
func (m *MLock[K]) ReleaseAll() {
	m.h.Range(func(k K, v MLockValue) bool {
		unlock(v)
		m.h.Delete(k)
		return true
	})
}
