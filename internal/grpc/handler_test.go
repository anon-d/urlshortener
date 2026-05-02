package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/anon-d/urlshortener/internal/auth"
	pb "github.com/anon-d/urlshortener/internal/grpc/pb"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/internal/service"
	"github.com/anon-d/urlshortener/internal/worker"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// --- mocks ---

type mockCacheService struct {
	data map[string]string
}

func (m *mockCacheService) Set(data *model.Data) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[data.ID] = data.OriginalURL
}

func (m *mockCacheService) Get(id string) (string, bool) {
	if m.data == nil {
		return "", false
	}
	val, ok := m.data[id]
	return val, ok
}

func (m *mockCacheService) Self() []model.Data {
	return nil
}

type mockStorage struct {
	shouldFail bool
	urls       map[string]model.Data // shortURL -> Data
	userURLs   []model.Data
}

func (m *mockStorage) Insert(ctx context.Context, data model.Data) error {
	if m.shouldFail {
		return errors.New("storage error")
	}
	return nil
}

func (m *mockStorage) InsertBatch(ctx context.Context, dataList []model.Data) error {
	return nil
}

func (m *mockStorage) Select(ctx context.Context) ([]model.Data, error) {
	return nil, nil
}

func (m *mockStorage) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	return "", errors.New("not found")
}

func (m *mockStorage) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	if m.shouldFail {
		return nil, errors.New("storage error")
	}
	return m.userURLs, nil
}

func (m *mockStorage) GetURLByShortURL(ctx context.Context, shortURL string) (model.Data, error) {
	if m.shouldFail {
		return model.Data{}, errors.New("not found")
	}
	if m.urls != nil {
		if d, ok := m.urls[shortURL]; ok {
			return d, nil
		}
	}
	return model.Data{}, errors.New("not found")
}

func (m *mockStorage) BatchMarkAsDeleted(ctx context.Context, requests []worker.DeleteRequest) error {
	return nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStorage) GetStats(ctx context.Context) (repository.Stats, error) {
	return repository.Stats{}, nil
}

// --- helpers ---

func newTestServer(cache *mockCacheService, storage repository.Storage) *ShortenerServer {
	logger := zap.NewNop().Sugar()
	svc := service.New(cache, storage, logger)
	return NewShortenerServer(svc, "http://localhost:8080", logger)
}

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), userIDKey{}, userID)
}

// --- ShortenURL tests ---

func TestShortenURL_Success(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)
	ctx := ctxWithUserID("user1")

	resp, err := srv.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetResult() == "" {
		t.Error("expected non-empty result")
	}
}

func TestShortenURL_EmptyURL(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)

	_, err := srv.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: ""})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestShortenURL_NoUserID(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)

	// Без user_id в контексте — всё равно работает (user_id опциональный для shorten)
	resp, err := srv.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetResult() == "" {
		t.Error("expected non-empty result")
	}
}

// --- ExpandURL tests ---

func TestExpandURL_Success(t *testing.T) {
	storage := &mockStorage{
		urls: map[string]model.Data{
			"abc123": {ShortURL: "abc123", OriginalURL: "https://example.com", IsDeleted: false},
		},
	}
	cache := &mockCacheService{
		data: map[string]string{"abc123": "https://example.com"},
	}
	srv := newTestServer(cache, storage)

	resp, err := srv.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetResult() != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", resp.GetResult())
	}
}

func TestExpandURL_EmptyID(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)

	_, err := srv.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: ""})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestExpandURL_NotFound(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)

	_, err := srv.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent URL")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestExpandURL_Deleted(t *testing.T) {
	storage := &mockStorage{
		urls: map[string]model.Data{
			"del123": {ShortURL: "del123", OriginalURL: "https://deleted.com", IsDeleted: true},
		},
	}
	cache := &mockCacheService{
		data: map[string]string{"del123": "https://deleted.com"},
	}
	srv := newTestServer(cache, storage)

	_, err := srv.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "del123"})
	if err == nil {
		t.Fatal("expected error for deleted URL")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

// --- ListUserURLs tests ---

func TestListUserURLs_Success(t *testing.T) {
	storage := &mockStorage{
		userURLs: []model.Data{
			{ShortURL: "abc", OriginalURL: "https://example1.com"},
			{ShortURL: "def", OriginalURL: "https://example2.com"},
		},
	}
	srv := newTestServer(&mockCacheService{}, storage)
	ctx := ctxWithUserID("user1")

	resp, err := srv.ListUserURLs(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetUrl()) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(resp.GetUrl()))
	}
}

func TestListUserURLs_NoAuth(t *testing.T) {
	srv := newTestServer(&mockCacheService{}, nil)

	_, err := srv.ListUserURLs(context.Background(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected error without auth")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestListUserURLs_Empty(t *testing.T) {
	storage := &mockStorage{userURLs: []model.Data{}}
	srv := newTestServer(&mockCacheService{}, storage)
	ctx := ctxWithUserID("user1")

	resp, err := srv.ListUserURLs(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetUrl()) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(resp.GetUrl()))
	}
}

func TestListUserURLs_StorageError(t *testing.T) {
	storage := &mockStorage{shouldFail: true}
	srv := newTestServer(&mockCacheService{}, storage)
	ctx := ctxWithUserID("user1")

	_, err := srv.ListUserURLs(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected error on storage failure")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Errorf("expected Internal, got %v", st.Code())
	}
}

// --- AuthInterceptor tests ---

func TestAuthInterceptor_ValidToken(t *testing.T) {
	secret := "test-secret"
	userID := "known-user"
	token := auth.SignValue(userID, secret)

	interceptor := AuthInterceptor(secret)

	md := metadata.Pairs("authorization", token)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}

	_, err := interceptor(ctx, nil, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotUserID, ok := UserIDFromContext(capturedCtx)
	if !ok {
		t.Fatal("expected user_id in context")
	}
	if gotUserID != userID {
		t.Errorf("expected %q, got %q", userID, gotUserID)
	}
}

func TestAuthInterceptor_NoToken(t *testing.T) {
	interceptor := AuthInterceptor("secret")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}

	_, err := interceptor(ctx, nil, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Должен сгенерировать нового пользователя
	gotUserID, ok := UserIDFromContext(capturedCtx)
	if !ok {
		t.Fatal("expected generated user_id in context")
	}
	if gotUserID == "" {
		t.Error("expected non-empty generated user_id")
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := AuthInterceptor("secret")

	md := metadata.Pairs("authorization", "invalid.token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}

	_, err := interceptor(ctx, nil, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Невалидный токен — должен сгенерировать нового пользователя
	gotUserID, ok := UserIDFromContext(capturedCtx)
	if !ok {
		t.Fatal("expected generated user_id in context")
	}
	if gotUserID == "" {
		t.Error("expected non-empty generated user_id")
	}
}

// --- UserIDFromContext tests ---

func TestUserIDFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey{}, "user123")
	id, ok := UserIDFromContext(ctx)
	if !ok || id != "user123" {
		t.Errorf("expected user123, got %q (ok=%v)", id, ok)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	_, ok := UserIDFromContext(context.Background())
	if ok {
		t.Error("expected false for missing user_id")
	}
}

func TestUserIDFromContext_Empty(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey{}, "")
	_, ok := UserIDFromContext(ctx)
	if ok {
		t.Error("expected false for empty user_id")
	}
}
