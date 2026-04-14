package auth

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	gen "github.com/vector-10/schedula/backend/gen"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func internalErr(msg string, err error) error {
	slog.Error(msg, "error", err)
	return status.Error(codes.Internal, msg)
}

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
	if req.FirstName == "" || req.LastName == "" {
		return nil, status.Error(codes.InvalidArgument, "first_name and last_name are required")
	}
	if req.WeekStart == "" {
		req.WeekStart = "monday"
	}
	if req.WeekStart != "monday" && req.WeekStart != "sunday" {
		return nil, status.Error(codes.InvalidArgument, "week_start must be 'monday' or 'sunday'")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, internalErr("failed to hash password", err)
	}

	var userID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, timezone, week_start, first_name, last_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.Email, string(hash), req.Timezone, req.WeekStart, req.FirstName, req.LastName, time.Now().UTC()).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, internalErr("failed to create user", err)
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return nil, internalErr("failed to generate token", err)
	}

	slog.Info("user registered", "user_id", userID)
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
		return nil, internalErr("failed to query user", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return nil, internalErr("failed to generate token", err)
	}

	return &gen.LoginResponse{Token: token, UserId: userID}, nil
}

func (s *Service) GetProfile(ctx context.Context, _ *gen.GetProfileRequest) (*gen.GetProfileResponse, error) {
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	var profile gen.GetProfileResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, first_name, last_name, timezone, week_start
		FROM users WHERE id = $1
	`, userID).Scan(
		&profile.Id,
		&profile.Email,
		&profile.FirstName,
		&profile.LastName,
		&profile.Timezone,
		&profile.WeekStart,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, internalErr("failed to get profile", err)
	}

	return &profile, nil
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
