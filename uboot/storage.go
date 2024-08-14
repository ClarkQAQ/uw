package uboot

import (
	"context"

	"uw/umap"
)

var (
	storage       = umap.NewHmap[string, any]()
	storageSignal = make(chan struct{})
)

type Key struct {
	string
}

func NewKey(value string) Key {
	return Key{
		string: value,
	}
}

func (k Key) String() string {
	return k.string
}

func storageNotify(ctx context.Context) error {
	for {
		select {
		case <-storageSignal:
			continue
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
}

func Set(key Key, value any) {
	storage.Set(key.String(), value)
}

func SetWait(ctx context.Context, key Key, value any) error {
	storage.Set(key.String(), value)
	return storageNotify(ctx)
}

func Load[T any](key Key) (T, bool) {
	if v, ok := storage.Load(key.String()); ok && v != nil {
		if ret, ok := v.(T); ok {
			return ret, true
		}
	}

	var empty T
	return empty, false
}

func Get[T any](key Key, defaultValue ...T) T {
	v, ok := Load[T](key)
	if !ok && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return v
}

func LoadWait[T any](ctx context.Context, key Key) (T, error) {
	if v, ok := Load[T](key); ok {
		return v, nil
	}

	for {
		select {
		case <-ctx.Done():
			var empty T
			return empty, ctx.Err()
		case storageSignal <- struct{}{}:
			if v, ok := Load[T](key); ok {
				return v, nil
			}
		}
	}
}

func Range[T any](f func(key Key, value T) bool) {
	storage.Range(func(k string, v any) bool {
		val, ok := v.(T)
		if !ok {
			return true
		}

		return f(NewKey(k), val)
	})
}

func Remove(key Key) {
	storage.Delete(key.String())
}
