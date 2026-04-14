package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	gen "github.com/vector-10/schedula/backend/gen"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	gen.UnimplementedAuthServiceServer
	db        *sql.DB
	jwtSecret string
}

func NewService(db *sql.DB, jwtSecret string) *Service {
	return &Service{db: db, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, req *gen.RegisterRequest) (*gen.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" || req.Timezone == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, and timezone are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	var userID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, timezone, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.Email, string(hash), req.Timezone, time.Now().UTC()).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &gen.RegisterResponse{Token: token, UserId: userID}, nil
}

func (s *Service) Login(ctx context.Context, req *gen.LoginRequest) (*gen.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	var userID, passwordHash string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash FROM users WHERE email = $1
	`, req.Email).Scan(&userID, &passwordHash)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to query user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &gen.LoginResponse{Token: token, UserId: userID}, nil
}

func (s *Service) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
