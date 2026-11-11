package fatigue

import (
	"testing"

	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestFatigueModel_GrowthAndDecay(t *testing.T) {
	fm := NewFatigueModel()

	if fm.Score() != 0.0 {
		t.Fatalf("Expected initial score 0, got %v", fm.Score())
	}

	snapActive := &types.ContextSnapshot{
		IdleSeconds:         2,
		KeystrokeCadenceWPM: 65,
	}

	fm.Update(snapActive, types.StateActive, 30.0)
	fm.Update(snapActive, types.StateActive, 30.0)

	scoreAfter1Min := fm.Score()
	if scoreAfter1Min < 1.15 || scoreAfter1Min > 1.25 {
		t.Errorf("Expected fatigue around 1.2, got %v", scoreAfter1Min)
	}

	snapMeeting := &types.ContextSnapshot{
		IdleSeconds: 60,
	}
	fm.Update(snapMeeting, types.StateInMeeting, 60.0)
	scoreAfterMeeting := fm.Score()
	if scoreAfterMeeting < 2.6 || scoreAfterMeeting > 2.8 {
		t.Errorf("Expected fatigue around 2.7, got %v", scoreAfterMeeting)
	}

	snapAway := &types.ContextSnapshot{
		IdleSeconds: 300,
	}
	fm.Update(snapAway, types.StateAway, 120.0)
	scoreAfterAway := fm.Score()
	if scoreAfterAway >= scoreAfterMeeting {
		t.Errorf("Fatigue should have decayed when user is away, was %v, now %v", scoreAfterMeeting, scoreAfterAway)
	}
}

func TestFatigueModel_NotificationBlindnessMitigation(t *testing.T) {
	fm := NewFatigueModel()

	if fm.IsAmbientModeOnly() {
		t.Errorf("Should not start in ambient mode")
	}

	fm.RecordDismissal(1.1)
	fm.RecordDismissal(0.8)
	fm.RecordDismissal(1.5)

	if !fm.IsAmbientModeOnly() {
		t.Errorf("Expected ambientModeOnly to be true after 3 rapid dismissals")
	}

	fm.ResetBreakTaken(300)
	if fm.IsAmbientModeOnly() {
		t.Errorf("Expected ambientModeOnly to reset to false after break taken")
	}
}
