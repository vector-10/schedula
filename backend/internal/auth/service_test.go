package auth

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	gen "github.com/vector-10/schedula/backend/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewService(db, "test-secret"), mock
}

func TestRegister_Success(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("INSERT INTO users").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-123"))

	resp, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "monday",
		FirstName: "John",
		LastName:  "Doe",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", resp.UserId)
	assert.NotEmpty(t, resp.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("INSERT INTO users").
		WillReturnError(&pq.Error{Code: "23505"})

	_, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "monday",
		FirstName: "John",
		LastName:  "Doe",
	})

	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_MissingFields(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:    "test@example.com",
		Password: "",
		Timezone: "UTC",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_MissingName(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "monday",
		FirstName: "",
		LastName:  "",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLogin_Success(t *testing.T) {
	svc, mock := newTestService(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT id, password_hash FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).
			AddRow("user-123", string(hash)))

	resp, err := svc.Login(context.Background(), &gen.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", resp.UserId)
	assert.NotEmpty(t, resp.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT id, password_hash FROM users").
		WillReturnError(sql.ErrNoRows)

	_, err := svc.Login(context.Background(), &gen.LoginRequest{
		Email:    "notfound@example.com",
		Password: "password123",
	})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, mock := newTestService(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT id, password_hash FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).
			AddRow("user-123", string(hash)))

	_, err = svc.Login(context.Background(), &gen.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_MissingCredentials(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Login(context.Background(), &gen.LoginRequest{
		Email:    "",
		Password: "",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
