package appointments

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vector-10/schedula/backend/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	gen "github.com/vector-10/schedula/backend/gen"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewService(db), mock
}

func ctxWithUser(userID string) context.Context {
	return auth.WithUserID(context.Background(), userID)
}

func expectTimezone(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT timezone FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"timezone"}).AddRow("UTC"))
}

func expectNoIdempotencyKey(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT appointment_ids FROM idempotency_keys").
		WillReturnRows(sqlmock.NewRows([]string{"appointment_ids"}))
}

func futureSlot() (start, end time.Time) {
	start = time.Now().UTC().Add(24 * time.Hour).Truncate(time.Minute)
	end = start.Add(time.Hour)
	return
}

func TestCreateAppointment_Unauthenticated(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateAppointment(context.Background(), &gen.CreateAppointmentRequest{
		Title: "Test",
	})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestCreateAppointment_EmptyTitle(t *testing.T) {
	svc, mock := newTestService(t)
	expectTimezone(mock)

	start, end := futureSlot()
	_, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:          "",
		StartTime:      timestamppb.New(start),
		EndTime:        timestamppb.New(end),
		IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateAppointment_EndBeforeStart(t *testing.T) {
	svc, mock := newTestService(t)
	expectTimezone(mock)

	start, end := futureSlot()
	_, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:          "Test",
		StartTime:      timestamppb.New(end),
		EndTime:        timestamppb.New(start),
		IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateAppointment_ConflictDetected(t *testing.T) {
	svc, mock := newTestService(t)
	start, end := futureSlot()

	expectTimezone(mock)
	expectNoIdempotencyKey(mock)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM appointments").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("SELECT start_time FROM appointments").
		WillReturnRows(sqlmock.NewRows([]string{"start_time"}).AddRow(start))
	mock.ExpectRollback()

	_, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:          "Test",
		StartTime:      timestamppb.New(start),
		EndTime:        timestamppb.New(end),
		IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAppointment_Success(t *testing.T) {
	svc, mock := newTestService(t)
	start, end := futureSlot()

	expectTimezone(mock)
	expectNoIdempotencyKey(mock)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM appointments").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("SELECT start_time FROM appointments").
		WillReturnRows(sqlmock.NewRows([]string{"start_time"}))
	mock.ExpectExec("INSERT INTO appointments").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO idempotency_keys").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	resp, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:          "Team standup",
		StartTime:      timestamppb.New(start),
		EndTime:        timestamppb.New(end),
		IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	require.Len(t, resp.Appointments, 1)
	assert.Equal(t, "Team standup", resp.Appointments[0].Title)
	assert.Equal(t, "scheduled", resp.Appointments[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAppointment_IdempotencyHit(t *testing.T) {
	svc, mock := newTestService(t)
	start, end := futureSlot()
	now := time.Now().UTC()
	apptID := "appt-existing"

	expectTimezone(mock)
	mock.ExpectQuery("SELECT appointment_ids FROM idempotency_keys").
		WillReturnRows(sqlmock.NewRows([]string{"appointment_ids"}).
			AddRow("{" + apptID + "}"))
	mock.ExpectQuery("SELECT id, user_id, title").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "title", "description",
			"start_time", "end_time", "status",
			"recurrence_group_id", "created_at", "updated_at",
		}).AddRow(
			apptID, "user-123", "Team standup", sql.NullString{},
			start, end, "scheduled",
			sql.NullString{}, now, now,
		))

	resp, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:          "Team standup",
		StartTime:      timestamppb.New(start),
		EndTime:        timestamppb.New(end),
		IdempotencyKey: "existing-key",
	})

	require.NoError(t, err)
	require.Len(t, resp.Appointments, 1)
	assert.Equal(t, apptID, resp.Appointments[0].Id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAppointment_RecurrenceTwoOccurrences(t *testing.T) {
	svc, mock := newTestService(t)
	start, end := futureSlot()
	recurrenceEnd := start.AddDate(0, 0, 8)

	expectTimezone(mock)
	expectNoIdempotencyKey(mock)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM appointments").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	for i := 0; i < 2; i++ {
		mock.ExpectQuery("SELECT start_time FROM appointments").
			WillReturnRows(sqlmock.NewRows([]string{"start_time"}))
	}
	for i := 0; i < 2; i++ {
		mock.ExpectExec("INSERT INTO appointments").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec("INSERT INTO idempotency_keys").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	resp, err := svc.CreateAppointment(ctxWithUser("user-123"), &gen.CreateAppointmentRequest{
		Title:             "Weekly sync",
		StartTime:         timestamppb.New(start),
		EndTime:           timestamppb.New(end),
		IdempotencyKey:    "key-recur",
		RecurrenceRule:    "WEEKLY",
		RecurrenceEndDate: timestamppb.New(recurrenceEnd),
	})

	require.NoError(t, err)
	assert.Len(t, resp.Appointments, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAppointments_LazyStatusUpdate(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)

	mock.ExpectQuery("SELECT timezone, week_start FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "week_start"}).
			AddRow("UTC", "monday"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE appointments SET status = 'completed'").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, user_id, title").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "title", "description",
			"start_time", "end_time", "status",
			"recurrence_group_id", "created_at", "updated_at",
		}).AddRow(
			"appt-1", "user-123", "Past meeting", sql.NullString{},
			past, past.Add(time.Hour), "completed",
			sql.NullString{}, now, now,
		))
	mock.ExpectCommit()

	resp, err := svc.GetAppointments(ctxWithUser("user-123"), &gen.GetAppointmentsRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Appointments, 1)
	assert.Equal(t, "completed", resp.Appointments[0].Status)
	assert.Equal(t, "UTC", resp.UserTimezone)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAppointments_Empty(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery("SELECT timezone, week_start FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"timezone", "week_start"}).
			AddRow("UTC", "monday"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE appointments SET status = 'completed'").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT id, user_id, title").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "title", "description",
			"start_time", "end_time", "status",
			"recurrence_group_id", "created_at", "updated_at",
		}))
	mock.ExpectCommit()

	resp, err := svc.GetAppointments(ctxWithUser("user-123"), &gen.GetAppointmentsRequest{})

	require.NoError(t, err)
	assert.Empty(t, resp.Appointments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCancelAppointment_Success(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now().UTC()
	start, end := futureSlot()

	mock.ExpectExec("UPDATE appointments SET status = 'cancelled'").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id, user_id, title").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "title", "description",
			"start_time", "end_time", "status",
			"recurrence_group_id", "created_at", "updated_at",
		}).AddRow(
			"appt-1", "user-123", "Team standup", sql.NullString{},
			start, end, "cancelled",
			sql.NullString{}, now, now,
		))

	resp, err := svc.CancelAppointment(ctxWithUser("user-123"), &gen.CancelAppointmentRequest{
		AppointmentId: "appt-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "cancelled", resp.Appointment.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCancelAppointment_NotFound(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectExec("UPDATE appointments SET status = 'cancelled'").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := svc.CancelAppointment(ctxWithUser("user-123"), &gen.CancelAppointmentRequest{
		AppointmentId: "appt-nonexistent",
	})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCancelAppointment_MissingID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CancelAppointment(ctxWithUser("user-123"), &gen.CancelAppointmentRequest{
		AppointmentId: "",
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
