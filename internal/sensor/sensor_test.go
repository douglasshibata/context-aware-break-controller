package sensor

import (
	"context"
	"testing"

	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestMockSensor_Snapshot(t *testing.T) {
	mock := NewMockSensor()
	mock.IdleSecondsVal = 25
	mock.DisplayModeVal = types.DisplayAcceptsNotifications
	mock.MicInUseVal = true
	mock.PeakMicAmpVal = 0.3
	mock.ForegroundClass = "ide"

	ctx := context.Background()
	snap, err := mock.Snapshot(ctx)
	if err != nil {
		t.Fatalf("unexpected error taking snapshot: %v", err)
	}

	if snap.IdleSeconds != 25 {
		t.Errorf("expected idle 25s, got %d", snap.IdleSeconds)
	}
	if !snap.Hardware.MicInUse {
		t.Errorf("expected mic in use")
	}
	if snap.Hardware.IsSilentCallHold {
		t.Errorf("expected active call, not silent hold")
	}
	if snap.ForegroundAppClass != "ide" {
		t.Errorf("expected ide foreground app")
	}
}

func TestMockSensor_PassiveFocusConditions(t *testing.T) {
	mock := NewMockSensor()
	mock.IdleSecondsVal = 200
	mock.AudioPlayingVal = true
	mock.ForegroundClass = "browser"

	ctx := context.Background()
	snap, err := mock.Snapshot(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !(snap.IdleSeconds > 180 && snap.Hardware.AudioPlaying && (snap.ForegroundAppClass == "browser" || snap.ForegroundAppClass == "media")) {
		t.Errorf("Failed passive focus precondition check")
	}
}
