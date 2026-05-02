package grpcserver

import (
	"context"
	"errors"
	"net/url"

	pb "github.com/anon-d/urlshortener/internal/grpc/pb"
	"github.com/anon-d/urlshortener/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ShortenerServer реализует gRPC-сервис ShortenerService.
// Является фасадом к общей бизнес-логике в service.Service.
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	service *service.Service
	urlAddr string
	logger  *zap.SugaredLogger
}

// NewShortenerServer создаёт новый экземпляр gRPC-сервера.
func NewShortenerServer(svc *service.Service, urlAddr string, logger *zap.SugaredLogger) *ShortenerServer {
	return &ShortenerServer{
		service: svc,
		urlAddr: urlAddr,
		logger:  logger,
	}
}

// ShortenURL сокращает переданный URL (аналог POST /api/shorten).
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	userID, _ := UserIDFromContext(ctx)

	id, err := s.service.ShortenURL(ctx, req.GetUrl(), userID)
	if err != nil {
		var conflictErr *service.ConflictError
		if errors.As(err, &conflictErr) {
			shortURL, joinErr := url.JoinPath(s.urlAddr, id)
			if joinErr != nil {
				s.logger.Errorw("failed to join URL path", "error", joinErr)
				return nil, status.Error(codes.Internal, "internal error")
			}
			// Передаём shortURL через details, т.к. gRPC игнорирует ответ при ошибке
			st, stErr := status.New(codes.AlreadyExists, "url already exists").
				WithDetails(&pb.URLShortenResponse{Result: shortURL})
			if stErr != nil {
				s.logger.Errorw("failed to attach status details", "error", stErr)
				return nil, status.Error(codes.Internal, "internal error")
			}
			return nil, st.Err()
		}
		s.logger.Errorw("failed to shorten URL", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	shortURL, err := url.JoinPath(s.urlAddr, id)
	if err != nil {
		s.logger.Errorw("failed to join URL path", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.URLShortenResponse{Result: shortURL}, nil
}

// ExpandURL возвращает оригинальный URL по короткому идентификатору (аналог GET /<id>).
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	data, err := s.service.GetURLByShortURL(ctx, req.GetId())
	if err != nil {
		s.logger.Errorw("failed to get URL", "error", err, "short_url", req.GetId())
		return nil, status.Error(codes.NotFound, "url not found")
	}

	if data.IsDeleted {
		return nil, status.Error(codes.NotFound, "url has been deleted")
	}

	return &pb.URLExpandResponse{Result: data.OriginalURL}, nil
}

// ListUserURLs возвращает все URL, созданные пользователем (аналог GET /api/user/urls).
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	urls, err := s.service.GetURLsByUser(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to get user URLs", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if len(urls) == 0 {
		return &pb.UserURLsResponse{}, nil
	}

	result := make([]*pb.URLData, 0, len(urls))
	for _, item := range urls {
		shortURL, err := url.JoinPath(s.urlAddr, item.ShortURL)
		if err != nil {
			s.logger.Errorw("failed to join URL path", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		result = append(result, &pb.URLData{
			ShortUrl:    shortURL,
			OriginalUrl: item.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{Url: result}, nil
}
