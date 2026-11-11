package anonymizer

import (
	"time"

	"github.com/user/context-aware-break-controller/pkg/types"
)

// Anonymizer ensures strict local privacy by discarding raw keystrokes and media buffers.
type Anonymizer struct {
	lastInputTime time.Time
	hasPriorInput bool
}

// NewAnonymizer creates a new in-memory anonymizer.
func NewAnonymizer() *Anonymizer {
	return &Anonymizer{
		lastInputTime: time.Time{},
		hasPriorInput: false,
	}
}

// ProcessInputTimestamp records an input event while discarding key identities.
// Retains exclusively delta-time between inputs and typing burst flags.
func (a *Anonymizer) ProcessInputTimestamp(now time.Time) types.UserInputEvent {
	if !a.hasPriorInput {
		a.lastInputTime = now
		a.hasPriorInput = true
		return types.UserInputEvent{
			DeltaSinceLastInput: 0,
			IsTypingBurst:       false,
			Timestamp:           now,
		}
	}

	delta := now.Sub(a.lastInputTime)
	a.lastInputTime = now

	// A burst is defined as subsequent strokes within 350ms of each other
	isBurst := delta < 350*time.Millisecond && delta > 0

	return types.UserInputEvent{
		DeltaSinceLastInput: delta,
		IsTypingBurst:       isBurst,
		Timestamp:           now,
	}
}

// NormalizeHardwareState scrubs audio/video streams, retaining solely boolean handles and amplitude levels.
func (a *Anonymizer) NormalizeHardwareState(micInUse, camInUse, audioPlaying bool, peakMicAmp float64, silentDuration time.Duration) types.HardwareMetrics {
	// False call detection: if mic is open but amplitude is at noise floor (e.g. <= 0.02) for >10 minutes
	isSilentHold := micInUse && peakMicAmp <= 0.02 && silentDuration > 10*time.Minute

	effectiveMic := micInUse
	if isSilentHold {
		// Downgrade mic status to inactive due to silence timeout
		effectiveMic = false
	}

	return types.HardwareMetrics{
		MicInUse:         effectiveMic,
		CameraInUse:      camInUse,
		AudioPlaying:     audioPlaying,
		PeakMicAmplitude: peakMicAmp,
		IsSilentCallHold: isSilentHold,
	}
}
