//go:build integration

package appointments

import (
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vector-10/schedula/backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	gen "github.com/vector-10/schedula/backend/gen"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

func createTestUser(t *testing.T, db *sql.DB) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.MinCost)
	require.NoError(t, err)

	var userID string
	err = db.QueryRow(`
		INSERT INTO users (email, password_hash, timezone, week_start, first_name, last_name, created_at)
		VALUES ($1, $2, 'UTC', 'monday', 'Test', 'User', NOW())
		RETURNING id
	`, "integration-test-"+time.Now().Format("20060102150405.000000000")+"@example.com", string(hash)).Scan(&userID)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Exec(`DELETE FROM idempotency_keys WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM appointments WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

func TestConcurrentBooking_OnlyOneSucceeds(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	svc := NewService(db)
	ctx := auth.WithUserID(t.Context(), userID)

	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Minute)
	end := start.Add(time.Hour)

	type result struct {
		err error
	}

	results := make([]result, 2)
	var wg sync.WaitGroup
	ready := make(chan struct{})

	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-ready
			_, err := svc.CreateAppointment(ctx, &gen.CreateAppointmentRequest{
				Title:          "Concurrent booking",
				StartTime:      timestamppb.New(start),
				EndTime:        timestamppb.New(end),
				IdempotencyKey: "concurrent-key-" + string(rune('A'+idx)),
			})
			results[idx] = result{err: err}
		}(i)
	}

	close(ready)
	wg.Wait()

	okCount := 0
	conflictCount := 0
	for _, r := range results {
		if r.err == nil {
			okCount++
		} else if status.Code(r.err) == codes.AlreadyExists {
			conflictCount++
		}
	}

	assert.Equal(t, 1, okCount, "exactly one booking should succeed")
	assert.Equal(t, 1, conflictCount, "exactly one booking should be rejected as conflict")
}
