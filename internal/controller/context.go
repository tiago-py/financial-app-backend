package controller

import "context"

type contextKey string

const (
	userIDKey    contextKey = "userID"
	requestIDKey contextKey = "requestID"
)

func UserID(ctx context.Context) string {
	value, _ := ctx.Value(userIDKey).(string)
	return value
}

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
