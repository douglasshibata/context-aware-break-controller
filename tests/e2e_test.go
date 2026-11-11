package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/internal/interrupter"
	"github.com/user/context-aware-break-controller/internal/notification"
	"github.com/user/context-aware-break-controller/internal/sensor"
	"github.com/user/context-aware-break-controller/internal/statemachine"
	"github.com/user/context-aware-break-controller/internal/storage"
	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestEndToEnd_FullSessionCycle(t *testing.T) {
	testDB := "test_e2e_wellness.db"
	defer os.Remove(testDB)

	store, err := storage.NewSQLiteStorage(testDB)
	if err != nil {
		t.Fatalf("Failed to create test db: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	fm := fatigue.NewFatigueModel()
	sm := statemachine.NewStateMachine(fm)
	intr := interrupter.NewInterrupter(fm)
	notifier := notification.NewSystemNotifier()
	mock := sensor.NewMockSensor()

	// Stage 1: User starts at desk (IDLE -> ACTIVE)
	mock.IdleSecondsVal = 3
	snapActive, _ := mock.Snapshot(ctx)
	state, _ := sm.Step(snapActive)
	if state != types.StateActive {
		t.Fatalf("Expected ACTIVE state, got %v", state)
	}

	// Stage 2: User attends a 40-minute Zoom call (ACTIVE -> IN_MEETING)
	mock.MicInUseVal = true
	mock.CamInUseVal = true
	mock.PeakMicAmpVal = 0.5
	snapMeeting, _ := mock.Snapshot(ctx)
	state, _ = sm.Step(snapMeeting)
	if state != types.StateInMeeting {
		t.Fatalf("Expected IN_MEETING state, got %v", state)
	}

	// Fast-forward meeting duration to 40 minutes
	sm.SetMeetingDurationForTest(40 * time.Minute)

	// Stage 3: Meeting ends (Mic/Cam off) -> TRANSITION_WINDOW with POST_MEETING_RESET
	mock.MicInUseVal = false
	mock.CamInUseVal = false
	mock.IdleSecondsVal = 20 // Natural post-meeting pause
	snapMeetingEnd, _ := mock.Snapshot(ctx)
	state, rec := sm.Step(snapMeetingEnd)

	if state != types.StateTransitionWindow {
		t.Fatalf("Expected TRANSITION_WINDOW, got %v", state)
	}
	if rec != types.InterventionPostMeetingReset {
		t.Fatalf("Expected POST_MEETING_RESET, got %v", rec)
	}

	// Evaluate with interrupter
	interv, ok := intr.EvaluateIntervention(rec, snapMeetingEnd)
	if !ok || interv == nil {
		t.Fatalf("Expected intervention staged")
	}

	// Dispatch notification & record in storage
	if err := notifier.Dispatch(ctx, interv); err != nil {
		t.Fatalf("Failed to dispatch: %v", err)
	}
	if err := store.RecordBreak(ctx, *interv); err != nil {
		t.Fatalf("Failed to record break: %v", err)
	}
	if err := store.RecordMeeting(ctx, 40); err != nil {
		t.Fatalf("Failed to record meeting: %v", err)
	}

	// Stage 4: User steps away for 10 minutes (AWAY state + fatigue decay)
	mock.IdleSecondsVal = 600
	snapAway, _ := mock.Snapshot(ctx)
	state, _ = sm.Step(snapAway)
	if state != types.StateAway {
		t.Fatalf("Expected AWAY state, got %v", state)
	}

	// Stage 5: Verify SQLite daily summary
	today := time.Now().Format("2006-01-02")
	summary, err := store.GetDailySummary(ctx, today)
	if err != nil {
		t.Fatalf("Failed to query summary: %v", err)
	}

	if summary.BreaksTaken != 1 {
		t.Errorf("Expected 1 break recorded, got %d", summary.BreaksTaken)
	}
	if summary.MeetingsCount != 1 {
		t.Errorf("Expected 1 meeting recorded, got %d", summary.MeetingsCount)
	}
}

func TestEdgeCase_FalseCallDetectionAndRecovery(t *testing.T) {
	mock := sensor.NewMockSensor()
	ctx := context.Background()

	// App holds mic open, but peak amplitude is 0.01 (noise floor) for 15 minutes (> 10m threshold)
	mock.MicInUseVal = true
	mock.PeakMicAmpVal = 0.01
	mock.SilentDuration = 15 * time.Minute

	snap, _ := mock.Snapshot(ctx)
	if snap.Hardware.MicInUse {
		t.Errorf("Expected false-call mic suppression to turn MicInUse false")
	}
	if !snap.Hardware.IsSilentCallHold {
		t.Errorf("Expected IsSilentCallHold to be true")
	}
}

func TestEdgeCase_NotificationBlindnessAmbientShift(t *testing.T) {
	fm := fatigue.NewFatigueModel()
	intr := interrupter.NewInterrupter(fm)
	mock := sensor.NewMockSensor()
	mock.IdleSecondsVal = 20
	ctx := context.Background()

	// Initial alert: Subtle Toast
	snap, _ := mock.Snapshot(ctx)
	interv1, ok := intr.EvaluateIntervention(types.InterventionMicroPause, snap)
	if !ok || interv1.Delivery != types.DeliverySubtleToast {
		t.Fatalf("Expected DeliverySubtleToast initially")
	}

	// User dismisses promptly within 1.0s three times
	fm.RecordDismissal(0.8)
	fm.RecordDismissal(0.9)
	fm.RecordDismissal(1.2)

	// Next alert should automatically shift to Ambient Tray
	interv2, ok := intr.EvaluateIntervention(types.InterventionMicroPause, snap)
	if !ok || interv2.Delivery != types.DeliveryAmbientTray {
		t.Fatalf("Expected DeliveryAmbientTray after 3 rapid dismissals, got %v", interv2.Delivery)
	}
}
