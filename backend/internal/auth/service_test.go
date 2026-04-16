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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
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

func TestLogin_DatabaseError(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT id, password_hash FROM users").
		WillReturnError(sql.ErrConnDone)

	_, err := svc.Login(context.Background(), &gen.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_InvalidWeekStart(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "wednesday",
		FirstName: "John",
		LastName:  "Doe",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_DatabaseError(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("INSERT INTO users").
		WillReturnError(sql.ErrConnDone)

	_, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "monday",
		FirstName: "John",
		LastName:  "Doe",
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfile_Unauthenticated(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetProfile(context.Background(), &gen.GetProfileRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGetProfile_Success(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT id, email, first_name, last_name, timezone, week_start").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "timezone", "week_start"}).
			AddRow("user-123", "test@example.com", "John", "Doe", "UTC", "monday"))

	ctx := WithUserID(context.Background(), "user-123")
	resp, err := svc.GetProfile(ctx, &gen.GetProfileRequest{})

	require.NoError(t, err)
	assert.Equal(t, "user-123", resp.Id)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfile_NotFound(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT id, email, first_name, last_name, timezone, week_start").
		WillReturnError(sql.ErrNoRows)

	ctx := WithUserID(context.Background(), "user-123")
	_, err := svc.GetProfile(ctx, &gen.GetProfileRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithUserID_RoundTrip(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-abc")
	assert.Equal(t, "user-abc", GetUserIDFromContext(ctx))
}

func TestGetUserIDFromContext_Missing(t *testing.T) {
	assert.Equal(t, "", GetUserIDFromContext(context.Background()))
}

func TestGetProfile_DatabaseError(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT id, email, first_name, last_name, timezone, week_start").
		WillReturnError(sql.ErrConnDone)

	ctx := WithUserID(context.Background(), "user-123")
	_, err := svc.GetProfile(ctx, &gen.GetProfileRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_DefaultWeekStart(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("INSERT INTO users").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-123"))

	resp, err := svc.Register(context.Background(), &gen.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		Timezone:  "UTC",
		WeekStart: "",
		FirstName: "John",
		LastName:  "Doe",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnaryInterceptor_PublicMethod(t *testing.T) {
	m := NewMiddleware("secret")
	interceptor := m.UnaryInterceptor()

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/schedula.v1.AuthService/Register"}
	_, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestUnaryInterceptor_MissingMetadata(t *testing.T) {
	m := NewMiddleware("secret")
	interceptor := m.UnaryInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/schedula.v1.AppointmentService/GetAppointments"}
	_, err := interceptor(context.Background(), nil, info, nil)

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryInterceptor_MissingAuthHeader(t *testing.T) {
	m := NewMiddleware("secret")
	interceptor := m.UnaryInterceptor()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	info := &grpc.UnaryServerInfo{FullMethod: "/schedula.v1.AppointmentService/GetAppointments"}
	_, err := interceptor(ctx, nil, info, nil)

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryInterceptor_InvalidToken(t *testing.T) {
	m := NewMiddleware("secret")
	interceptor := m.UnaryInterceptor()

	md := metadata.Pairs("authorization", "Bearer not-a-real-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/schedula.v1.AppointmentService/GetAppointments"}
	_, err := interceptor(ctx, nil, info, nil)

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryInterceptor_ValidToken(t *testing.T) {
	m := NewMiddleware("secret")
	svc := NewService(nil, "secret")

	token, err := svc.generateToken("user-123")
	require.NoError(t, err)

	interceptor := m.UnaryInterceptor()
	var gotUserID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		gotUserID = GetUserIDFromContext(ctx)
		return nil, nil
	}

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/schedula.v1.AppointmentService/GetAppointments"}
	_, err = interceptor(ctx, nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "user-123", gotUserID)
}
