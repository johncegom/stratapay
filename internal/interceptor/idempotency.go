package interceptor

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const lockTTL = 5 * time.Second

type idempotencyKeyed interface {
	GetIdempotencyKey() string
}

type IdempotencyShield struct {
	rdb *redis.Client
}

func Timing(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("interceptor: method=%s took=%s", info.FullMethod, time.Since(start))
	return resp, err
}

func NewIdempotencyShield(rdb *redis.Client) *IdempotencyShield {
	return &IdempotencyShield{rdb: rdb}
}

func (s *IdempotencyShield) UnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	keyed, ok := req.(idempotencyKeyed)
	if !ok {
		return handler(ctx, req)
	}

	lockKey := "idempotency:lock:" + keyed.GetIdempotencyKey()

	acquired, err := s.rdb.SetNX(ctx, lockKey, "held", lockTTL).Result()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "idempotency lock check failed: %v", err)
	}
	if !acquired {
		log.Printf("interceptor: method=%s key=%s rejected: lock already held", info.FullMethod, keyed.GetIdempotencyKey())
		return nil, status.Errorf(codes.Aborted, "a request with this idempotency key is already in progress")
	}
	defer s.rdb.Del(ctx, lockKey)

	return handler(ctx, req)
}
