package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/user/context-aware-break-controller/pkg/types"
	_ "modernc.org/sqlite"
)

// Storage defines the interface for local encrypted/secure persistence.
type Storage interface {
	RecordBreak(ctx context.Context, intervention types.Intervention) error
	RecordMeeting(ctx context.Context, durationMin int) error
	RecordActiveTime(ctx context.Context, minutes int, fatigueScore float64) error
	RecordDismissal(ctx context.Context, isRapid bool) error
	GetDailySummary(ctx context.Context, date string) (*types.DailyWellnessSummary, error)
	Close() error
}

// SQLiteStorage persists anonymous daily wellness metrics locally without cloud leakage.
type SQLiteStorage struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteStorage opens or creates the local SQLite database.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	if dbPath == "" {
		dbPath = "wellness_local.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &SQLiteStorage{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStorage) initSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	CREATE TABLE IF NOT EXISTS daily_wellness_log (
		date TEXT PRIMARY KEY,
		active_minutes INTEGER DEFAULT 0,
		meetings_count INTEGER DEFAULT 0,
		breaks_taken INTEGER DEFAULT 0,
		fatigue_score_avg REAL DEFAULT 0.0,
		dismissals_count INTEGER DEFAULT 0,
		last_updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS break_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT,
		break_type TEXT,
		delivery TEXT,
		duration_sec INTEGER,
		trigger_reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(query)
	return err
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// RecordBreak inserts a break event and increments the daily breaks_taken count.
func (s *SQLiteStorage) RecordBreak(ctx context.Context, intervention types.Intervention) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO daily_wellness_log (date, breaks_taken, last_updated_at)
		VALUES (?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(date) DO UPDATE SET
			breaks_taken = breaks_taken + 1,
			last_updated_at = CURRENT_TIMESTAMP;
	`, date)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO break_events (date, break_type, delivery, duration_sec, trigger_reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?);
	`, date, string(intervention.Type), string(intervention.Delivery), intervention.DurationSec, intervention.TriggerReason, time.Now())
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RecordMeeting increments the daily meeting counter.
func (s *SQLiteStorage) RecordMeeting(ctx context.Context, durationMin int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_wellness_log (date, meetings_count, last_updated_at)
		VALUES (?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(date) DO UPDATE SET
			meetings_count = meetings_count + 1,
			last_updated_at = CURRENT_TIMESTAMP;
	`, date)
	return err
}

// RecordActiveTime aggregates continuous active minutes and updates the running fatigue average.
func (s *SQLiteStorage) RecordActiveTime(ctx context.Context, minutes int, fatigueScore float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_wellness_log (date, active_minutes, fatigue_score_avg, last_updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(date) DO UPDATE SET
			active_minutes = active_minutes + ?,
			fatigue_score_avg = (fatigue_score_avg + ?) / 2.0,
			last_updated_at = CURRENT_TIMESTAMP;
	`, date, minutes, fatigueScore, minutes, fatigueScore)
	return err
}

// RecordDismissal tracks rapid dismissals for audit and blindness tracking.
func (s *SQLiteStorage) RecordDismissal(ctx context.Context, isRapid bool) error {
	if !isRapid {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	date := todayDate()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_wellness_log (date, dismissals_count, last_updated_at)
		VALUES (?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(date) DO UPDATE SET
			dismissals_count = dismissals_count + 1,
			last_updated_at = CURRENT_TIMESTAMP;
	`, date)
	return err
}

// GetDailySummary retrieves the aggregated metrics for a specified date.
func (s *SQLiteStorage) GetDailySummary(ctx context.Context, date string) (*types.DailyWellnessSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if date == "" {
		date = todayDate()
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT date, active_minutes, meetings_count, breaks_taken, fatigue_score_avg, dismissals_count, last_updated_at
		FROM daily_wellness_log
		WHERE date = ?;
	`, date)

	var summary types.DailyWellnessSummary
	var lastUpdated string
	err := row.Scan(
		&summary.Date,
		&summary.ActiveMinutes,
		&summary.MeetingsCount,
		&summary.BreaksTaken,
		&summary.FatigueScoreAvg,
		&summary.DismissalsCount,
		&lastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &types.DailyWellnessSummary{
				Date:          date,
				LastUpdatedAt: time.Now(),
			}, nil
		}
		return nil, err
	}

	summary.LastUpdatedAt, _ = time.Parse(time.RFC3339, lastUpdated)
	return &summary, nil
}

// Close closes the SQLite connection.
func (s *SQLiteStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
