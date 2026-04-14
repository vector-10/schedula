package appointments

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	gen "github.com/vector-10/schedula/backend/gen"
	"github.com/vector-10/schedula/backend/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	gen.UnimplementedAppointmentServiceServer
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type scanner interface {
	Scan(dest ...any) error
}

type appointmentRow struct {
	id                string
	userID            string
	title             string
	description       sql.NullString
	startTime         time.Time
	endTime           time.Time
	status            string
	recurrenceGroupID sql.NullString
	createdAt         time.Time
	updatedAt         time.Time
}

func (r *appointmentRow) scan(s scanner) error {
	return s.Scan(
		&r.id, &r.userID, &r.title, &r.description,
		&r.startTime, &r.endTime, &r.status,
		&r.recurrenceGroupID, &r.createdAt, &r.updatedAt,
	)
}

func (r *appointmentRow) toProto() *gen.Appointment {
	a := &gen.Appointment{
		Id:        r.id,
		UserId:    r.userID,
		Title:     r.title,
		Status:    r.status,
		StartTime: timestamppb.New(r.startTime),
		EndTime:   timestamppb.New(r.endTime),
		CreatedAt: timestamppb.New(r.createdAt),
		UpdatedAt: timestamppb.New(r.updatedAt),
	}
	if r.description.Valid {
		a.Description = r.description.String
	}
	if r.recurrenceGroupID.Valid {
		a.RecurrenceGroupId = r.recurrenceGroupID.String
	}
	return a
}

type occurrence struct {
	startTime time.Time
	endTime   time.Time
}

func (s *Service) CreateAppointment(ctx context.Context, req *gen.CreateAppointmentRequest) (*gen.CreateAppointmentResponse, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.StartTime == nil || req.EndTime == nil {
		return nil, status.Error(codes.InvalidArgument, "start_time and end_time are required")
	}
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	startTime := req.StartTime.AsTime()
	endTime := req.EndTime.AsTime()

	if !endTime.After(startTime) {
		return nil, status.Error(codes.InvalidArgument, "end_time must be after start_time")
	}

	if resp, err := s.checkIdempotency(ctx, req.IdempotencyKey, userID); err == nil {
		return resp, nil
	}

	occurrences := []occurrence{{startTime, endTime}}
	duration := endTime.Sub(startTime)

	if req.RecurrenceRule != "" {
		if req.RecurrenceRule != "WEEKLY" {
			return nil, status.Error(codes.InvalidArgument, "only WEEKLY recurrence is supported")
		}
		if req.RecurrenceEndDate == nil {
			return nil, status.Error(codes.InvalidArgument, "recurrence_end_date is required for recurring appointments")
		}

		endDate := req.RecurrenceEndDate.AsTime()
		current := startTime

		for {
			next := current.AddDate(0, 0, 7)
			if next.After(endDate) {
				break
			}
			occurrences = append(occurrences, occurrence{next, next.Add(duration)})
			current = next
			if len(occurrences) > 4 {
				return nil, status.Error(codes.InvalidArgument,
					"recurrence rule would generate more than 4 occurrences; shorten recurrence_end_date")
			}
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback()

	lockRows, err := tx.QueryContext(ctx, `
		SELECT id FROM appointments
		WHERE user_id = $1 AND status = 'scheduled'
		FOR UPDATE
	`, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to acquire lock")
	}
	lockRows.Close()

	for _, occ := range occurrences {
		var conflictStart time.Time
		err := tx.QueryRowContext(ctx, `
			SELECT start_time FROM appointments
			WHERE user_id    = $1
			  AND status     = 'scheduled'
			  AND start_time < $3
			  AND end_time   > $2
			LIMIT 1
		`, userID, occ.startTime, occ.endTime).Scan(&conflictStart)

		if err == nil {
			return nil, status.Errorf(codes.AlreadyExists,
				"appointment on %s conflicts with an existing appointment at %s",
				occ.startTime.UTC().Format("2006-01-02 15:04 UTC"),
				conflictStart.UTC().Format("2006-01-02 15:04 UTC"),
			)
		}
		if err != sql.ErrNoRows {
			return nil, status.Error(codes.Internal, "conflict check failed")
		}
	}

	var recurrenceGroupID *string
	if len(occurrences) > 1 {
		id := uuid.New().String()
		recurrenceGroupID = &id
	}

	now := time.Now().UTC()
	appointmentIDs := make([]string, 0, len(occurrences))
	appointments := make([]*gen.Appointment, 0, len(occurrences))

	for _, occ := range occurrences {
		apptID := uuid.New().String()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO appointments
				(id, user_id, title, description, start_time, end_time, status, recurrence_group_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'scheduled', $7, $8, $9)
		`,
			apptID, userID, req.Title,
			sql.NullString{String: req.Description, Valid: req.Description != ""},
			occ.startTime.UTC(), occ.endTime.UTC(),
			recurrenceGroupID, now, now,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to insert appointment")
		}

		a := &gen.Appointment{
			Id:          apptID,
			UserId:      userID,
			Title:       req.Title,
			Description: req.Description,
			StartTime:   timestamppb.New(occ.startTime),
			EndTime:     timestamppb.New(occ.endTime),
			Status:      "scheduled",
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now),
		}
		if recurrenceGroupID != nil {
			a.RecurrenceGroupId = *recurrenceGroupID
		}
		appointments = append(appointments, a)
		appointmentIDs = append(appointmentIDs, apptID)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO idempotency_keys (key, user_id, appointment_ids, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (key, user_id) DO NOTHING
	`, req.IdempotencyKey, userID, pq.Array(appointmentIDs))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to store idempotency key")
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	return &gen.CreateAppointmentResponse{Appointments: appointments}, nil
}

func (s *Service) GetAppointments(ctx context.Context, _ *gen.GetAppointmentsRequest) (*gen.GetAppointmentsResponse, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	var timezone, weekStart string
	if err := s.db.QueryRowContext(ctx, `SELECT timezone, week_start FROM users WHERE id = $1`, userID).Scan(&timezone, &weekStart); err != nil {
		return nil, status.Error(codes.Internal, "failed to get user preferences")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE appointments
		SET    status = 'completed', updated_at = NOW()
		WHERE  user_id = $1 AND status = 'scheduled' AND end_time < NOW()
	`, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update completed appointments")
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, title, description, start_time, end_time,
		       status, recurrence_group_id, created_at, updated_at
		FROM   appointments
		WHERE  user_id = $1
		ORDER  BY start_time ASC
	`, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to query appointments")
	}
	defer rows.Close()

	appointments := make([]*gen.Appointment, 0)
	for rows.Next() {
		var row appointmentRow
		if err := row.scan(rows); err != nil {
			return nil, status.Error(codes.Internal, "failed to scan appointment")
		}
		appointments = append(appointments, row.toProto())
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "row iteration error")
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	return &gen.GetAppointmentsResponse{
		Appointments: appointments,
		UserTimezone: timezone,
		WeekStart:    weekStart,
	}, nil
}

func (s *Service) CancelAppointment(ctx context.Context, req *gen.CancelAppointmentRequest) (*gen.CancelAppointmentResponse, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	if req.AppointmentId == "" {
		return nil, status.Error(codes.InvalidArgument, "appointment_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE appointments
		SET    status = 'cancelled', updated_at = NOW()
		WHERE  id = $1 AND user_id = $2 AND status = 'scheduled'
	`, req.AppointmentId, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel appointment")
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, status.Error(codes.NotFound, "appointment not found, already cancelled, or completed")
	}

	var row appointmentRow
	if err := row.scan(s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, description, start_time, end_time,
		       status, recurrence_group_id, created_at, updated_at
		FROM   appointments WHERE id = $1
	`, req.AppointmentId)); err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch cancelled appointment")
	}

	return &gen.CancelAppointmentResponse{Appointment: row.toProto()}, nil
}

func (s *Service) checkIdempotency(ctx context.Context, key, userID string) (*gen.CreateAppointmentResponse, error) {
	var appointmentIDs []string
	err := s.db.QueryRowContext(ctx, `
		SELECT appointment_ids FROM idempotency_keys WHERE key = $1 AND user_id = $2
	`, key, userID).Scan(pq.Array(&appointmentIDs))
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, description, start_time, end_time,
		       status, recurrence_group_id, created_at, updated_at
		FROM   appointments
		WHERE  id = ANY($1::uuid[])
		ORDER  BY start_time ASC
	`, pq.Array(appointmentIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appointments := make([]*gen.Appointment, 0)
	for rows.Next() {
		var row appointmentRow
		if err := row.scan(rows); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		appointments = append(appointments, row.toProto())
	}

	return &gen.CreateAppointmentResponse{Appointments: appointments}, nil
}
