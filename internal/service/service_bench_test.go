package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/anon-d/urlshortener/internal/model"
	"go.uber.org/zap"
)

// mockBenchCacheService — мок для CacheService в бенчмарках
type mockBenchCacheService struct {
	data map[string]string
}

func (m *mockBenchCacheService) Set(data *model.Data) {
	m.data[data.ID] = data.OriginalURL
}

func (m *mockBenchCacheService) Get(id string) (string, bool) {
	val, ok := m.data[id]
	return val, ok
}

func (m *mockBenchCacheService) Self() []model.Data {
	return nil
}

func BenchmarkGenerateID(b *testing.B) {
	for b.Loop() {
		GenerateID()
	}
}

func BenchmarkShortenURL(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockBenchCacheService{data: make(map[string]string)}
	svc := New(cache, nil, logger)
	ctx := context.Background()
	url := "https://example.com/very/long/path/to/resource"

	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.ShortenURL(ctx, url, "user1")
	}
}

func BenchmarkGetURL(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockBenchCacheService{data: make(map[string]string)}
	svc := New(cache, nil, logger)

	// Предзаполняем кэш
	for i := 0; i < 1000; i++ {
		cache.data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("https://example.com/%d", i)
	}

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.GetURL(ctx, "key500")
	}
}

func BenchmarkShortenBatchURL(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockBenchCacheService{data: make(map[string]string)}
	svc := New(cache, nil, logger)
	ctx := context.Background()

	batchMap := make(map[string]string, 100)
	for i := 0; i < 100; i++ {
		batchMap[fmt.Sprintf("corr_%d", i)] = fmt.Sprintf("https://example.com/%d", i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.ShortenBatchURL(ctx, batchMap, "user1")
	}
}
