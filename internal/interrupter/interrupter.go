package interrupter

import (
	"fmt"
	"time"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/pkg/types"
)

// Interrupter decides whether and how an intervention should be staged.
type Interrupter struct {
	fatigueModel *fatigue.FatigueModel
}

// NewInterrupter creates a context-aware interrupter.
func NewInterrupter(fm *fatigue.FatigueModel) *Interrupter {
	return &Interrupter{
		fatigueModel: fm,
	}
}

// EvaluateIntervention checks display mode suppression and context readiness to produce a concrete Intervention.
func (i *Interrupter) EvaluateIntervention(
	interventionType types.InterventionType,
	snapshot *types.ContextSnapshot,
) (*types.Intervention, bool) {
	if interventionType == types.InterventionNone {
		return nil, false
	}

	if snapshot.DisplayMode != types.DisplayAcceptsNotifications {
		return nil, false
	}

	isAmbient := i.fatigueModel.IsAmbientModeOnly()
	now := time.Now()

	switch interventionType {
	case types.InterventionMicroPause:
		delivery := types.DeliverySubtleToast
		if isAmbient {
			delivery = types.DeliveryAmbientTray
		}

		return &types.Intervention{
			Type:          types.InterventionMicroPause,
			Delivery:      delivery,
			DurationSec:   20,
			Title:         "Cognitive Pause (20s)",
			Message:       "High continuous focus detected. Rest your eyes: look out the window or at an object 20 feet away.",
			ActionPrompt:  "Look away for 20 seconds",
			TriggerReason: fmt.Sprintf("Fatigue Score: %.1f | Natural cognitive gap (%ds pause)", i.fatigueModel.Score(), snapshot.IdleSeconds),
			CreatedAt:     now,
		}, true

	case types.InterventionPostMeetingReset:
		delivery := types.DeliverySubtleToast
		if isAmbient {
			delivery = types.DeliveryAmbientTray
		}

		return &types.Intervention{
			Type:          types.InterventionPostMeetingReset,
			Delivery:      delivery,
			DurationSec:   180,
			Title:         "Meeting Decompression (2-3 min)",
			Message:       "Call concluded. Stand up, stretch your shoulders, and drink water before opening your next task.",
			ActionPrompt:  "Stretch and rehydrate",
			TriggerReason: "Call ended after >= 30 minutes in session",
			CreatedAt:     now,
		}, true

	case types.InterventionErgonomicReset:
		delivery := types.DeliveryModalDialog
		if isAmbient {
			delivery = types.DeliverySubtleToast
		}

		return &types.Intervention{
			Type:          types.InterventionErgonomicReset,
			Delivery:      delivery,
			DurationSec:   300,
			Title:         "Ergonomic Reset Due (5 min)",
			Message:       "Continuous active session reached 90 minutes. Step away from your desk for a complete cognitive reset.",
			ActionPrompt:  "Step away from your workstation",
			TriggerReason: fmt.Sprintf("Fatigue Score: %.1f | Continuous Focus: %.1f min", i.fatigueModel.Score(), i.fatigueModel.ContinuousActiveMinutes()),
			CreatedAt:     now,
		}, true

	default:
		return nil, false
	}
}
