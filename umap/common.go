package umap

type Hashable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string
}

type Mapper[K Hashable, V any] interface {
	Load(key K) (V, bool)
	Get(key K) V
	Set(key K, val V) V
	Range(f func(k K, v V) bool)
	Delete(key K)
	Len() int
	Clean()
}
