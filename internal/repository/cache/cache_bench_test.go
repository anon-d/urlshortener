package cache

import (
	"fmt"
	"testing"
)

func BenchmarkCacheSet(b *testing.B) {
	c := &Cache{data: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("https://example.com/%d", i))
	}
}

func BenchmarkCacheGetOne(b *testing.B) {
	c := &Cache{data: make(map[string]string)}
	for i := 0; i < 10000; i++ {
		c.data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("https://example.com/%d", i)
	}

	b.ResetTimer()
	for b.Loop() {
		c.GetOne("key5000")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := &Cache{data: make(map[string]string)}
	for i := 0; i < 10000; i++ {
		c.data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("https://example.com/%d", i)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = c.Get()
	}
}
