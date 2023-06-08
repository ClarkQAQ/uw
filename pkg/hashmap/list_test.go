package hashmap

import (
	"testing"

	"uw/pkg/hashmap/assert"
)

func TestListNew(t *testing.T) {
	l := NewList[uintptr, uintptr]()
	node := l.First()
	assert.True(t, node == nil)

	node = l.head.Next()
	assert.True(t, node == nil)
}
