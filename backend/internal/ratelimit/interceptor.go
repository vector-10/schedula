package ratelimit

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var limitedMethods = map[string]bool{
	"/schedula.v1.AuthService/Register": true,
	"/schedula.v1.AuthService/Login":    true,
}

const (
	maxRequests = 10
	window      = time.Minute
)

type entry struct {
	count     int
	windowEnd time.Time
}

type Limiter struct {
	mu      sync.Mutex
	clients map[string]*entry
}

func NewLimiter() *Limiter {
	return &Limiter{clients: make(map[string]*entry)}
}

func (l *Limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.clients[ip]
	if !ok || now.After(e.windowEnd) {
		l.clients[ip] = &entry{count: 1, windowEnd: now.Add(window)}
		return true
	}
	if e.count >= maxRequests {
		return false
	}
	e.count++
	return true
}

func (l *Limiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !limitedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		ip := ipFromContext(ctx)
		if !l.allow(ip) {
			return nil, status.Error(codes.ResourceExhausted, "too many requests, please try again later")
		}

		return handler(ctx, req)
	}
}

func ipFromContext(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
