package interrupter

import (
	"testing"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestInterrupter_Suppression(t *testing.T) {
	fm := fatigue.NewFatigueModel()
	interrupter := NewInterrupter(fm)

	snapFullscreen := &types.ContextSnapshot{
		DisplayMode: types.DisplayD3DFullScreen,
	}
	interv, ok := interrupter.EvaluateIntervention(types.InterventionMicroPause, snapFullscreen)
	if ok || interv != nil {
		t.Errorf("Expected intervention to be suppressed in fullscreen mode")
	}

	snapPresentation := &types.ContextSnapshot{
		DisplayMode: types.DisplayPresentationMode,
	}
	interv, ok = interrupter.EvaluateIntervention(types.InterventionMicroPause, snapPresentation)
	if ok || interv != nil {
		t.Errorf("Expected intervention to be suppressed in presentation mode")
	}

	snapNormal := &types.ContextSnapshot{
		DisplayMode: types.DisplayAcceptsNotifications,
		IdleSeconds: 20,
	}
	interv, ok = interrupter.EvaluateIntervention(types.InterventionMicroPause, snapNormal)
	if !ok || interv == nil {
		t.Fatalf("Expected intervention to trigger")
	}
	if interv.Type != types.InterventionMicroPause {
		t.Errorf("Expected MicroPause type, got %v", interv.Type)
	}
	if interv.Delivery != types.DeliverySubtleToast {
		t.Errorf("Expected DeliverySubtleToast, got %v", interv.Delivery)
	}
}

func TestInterrupter_BlindnessMitigation(t *testing.T) {
	fm := fatigue.NewFatigueModel()
	interrupter := NewInterrupter(fm)

	fm.RecordDismissal(0.5)
	fm.RecordDismissal(0.5)
	fm.RecordDismissal(0.5)

	snap := &types.ContextSnapshot{
		DisplayMode: types.DisplayAcceptsNotifications,
		IdleSeconds: 20,
	}

	interv, ok := interrupter.EvaluateIntervention(types.InterventionMicroPause, snap)
	if !ok || interv == nil {
		t.Fatalf("Expected intervention")
	}

	if interv.Delivery != types.DeliveryAmbientTray {
		t.Errorf("Expected delivery downgraded to DeliveryAmbientTray, got %v", interv.Delivery)
	}
}
