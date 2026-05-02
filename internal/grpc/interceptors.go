// Package grpcserver содержит gRPC-сервер и перехватчики для сервиса сокращения URL.
package grpcserver

import (
	"context"

	"github.com/anon-d/urlshortener/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// userIDKey — тип ключа для хранения user_id в context.Context.
type userIDKey struct{}

// AuthInterceptor возвращает gRPC UnaryServerInterceptor для авторизации.
// Читает подписанный user_id из metadata "authorization".
// Если токен отсутствует или невалиден, генерирует новый user_id
// и отправляет подписанное значение обратно через response headers.
func AuthInterceptor(secretKey string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		var userID string

		// Извлекаем metadata из запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get("authorization")
			if len(values) > 0 {
				if validUserID, valid := auth.ValidateSignedValue(values[0], secretKey); valid {
					userID = validUserID
				}
			}
		}

		// Если авторизация не предоставлена или невалидна — генерируем нового пользователя
		if userID == "" {
			userID = auth.GenerateUserID()
			signedValue := auth.SignValue(userID, secretKey)
			// Отправляем подписанный токен обратно в response headers
			header := metadata.Pairs("authorization", signedValue)
			_ = grpc.SetHeader(ctx, header)
		}

		// Записываем user_id в контекст
		ctx = context.WithValue(ctx, userIDKey{}, userID)

		return handler(ctx, req)
	}
}

// UserIDFromContext извлекает user_id из context.Context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey{}).(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}
