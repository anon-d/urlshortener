package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/anon-d/urlshortener/internal/model"
	"go.uber.org/zap"
)

// mockProfileCacheService — мок для кэша, используемый при профилировании
type mockProfileCacheService struct {
	data map[string]string
}

func (m *mockProfileCacheService) Set(data *model.Data) {
	m.data[data.ID] = data.OriginalURL
}

func (m *mockProfileCacheService) Get(id string) (string, bool) {
	val, ok := m.data[id]
	return val, ok
}

func (m *mockProfileCacheService) Self() []model.Data {
	return nil
}

// TestProfileMemory выполняет типичную нагрузку и сохраняет профиль памяти.
// Путь к выходному файлу задаётся через переменную окружения PPROF_OUT
// (по умолчанию profiles/base.pprof).
func TestProfileMemory(t *testing.T) {
	outPath := os.Getenv("PPROF_OUT")
	if outPath == "" {
		outPath = "../../profiles/base.pprof"
	}

	logger := zap.NewNop().Sugar()
	cache := &mockProfileCacheService{data: make(map[string]string)}
	svc := New(cache, nil, logger)
	ctx := context.Background()

	// 1. Массовое сокращение URL — основная нагрузка на аллокации
	for i := 0; i < 10000; i++ {
		url := fmt.Sprintf("https://example.com/path/%d", i)
		_, _ = svc.ShortenURL(ctx, url, fmt.Sprintf("user_%d", i%100))
	}

	// 2. Пакетное сокращение
	for i := 0; i < 100; i++ {
		batch := make(map[string]string, 50)
		for j := 0; j < 50; j++ {
			batch[fmt.Sprintf("corr_%d_%d", i, j)] = fmt.Sprintf("https://example.com/batch/%d/%d", i, j)
		}
		_, _ = svc.ShortenBatchURL(ctx, batch, "batch_user")
	}

	// 3. Чтение из кэша
	for i := 0; i < 5000; i++ {
		for key := range cache.data {
			_, _ = svc.GetURL(ctx, key)
			break
		}
	}

	// Сохраняем профиль
	runtime.GC()
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("не удалось создать файл профиля %s: %v", outPath, err)
	}
	defer func() { _ = f.Close() }()

	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatalf("не удалось записать профиль: %v", err)
	}

	t.Logf("Профиль памяти сохранён в %s", outPath)
}
