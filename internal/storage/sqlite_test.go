package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestSQLiteStorage_CRUD(t *testing.T) {
	tmpDB := "test_wellness.db"
	defer os.Remove(tmpDB)

	s, err := NewSQLiteStorage(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize sqlite storage: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	interv := types.Intervention{
		Type:          types.InterventionMicroPause,
		Delivery:      types.DeliverySubtleToast,
		DurationSec:   20,
		TriggerReason: "Fatigue threshold exceeded",
	}
	if err := s.RecordBreak(ctx, interv); err != nil {
		t.Fatalf("Failed to record break: %v", err)
	}

	if err := s.RecordMeeting(ctx, 45); err != nil {
		t.Fatalf("Failed to record meeting: %v", err)
	}

	if err := s.RecordActiveTime(ctx, 30, 25.5); err != nil {
		t.Fatalf("Failed to record active time: %v", err)
	}

	if err := s.RecordDismissal(ctx, true); err != nil {
		t.Fatalf("Failed to record dismissal: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	summary, err := s.GetDailySummary(ctx, today)
	if err != nil {
		t.Fatalf("Failed to get daily summary: %v", err)
	}

	if summary.BreaksTaken != 1 {
		t.Errorf("Expected 1 break taken, got %d", summary.BreaksTaken)
	}
	if summary.MeetingsCount != 1 {
		t.Errorf("Expected 1 meeting, got %d", summary.MeetingsCount)
	}
	if summary.ActiveMinutes != 30 {
		t.Errorf("Expected 30 active minutes, got %d", summary.ActiveMinutes)
	}
	if summary.DismissalsCount != 1 {
		t.Errorf("Expected 1 dismissal, got %d", summary.DismissalsCount)
	}
}
