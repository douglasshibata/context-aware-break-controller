package fatigue

import (
	"math"
	"sync"
	"time"

	"github.com/user/context-aware-break-controller/pkg/types"
)

// Exertion weights w_state
const (
	WeightHighIntensity = 1.2 // High-intensity typing/mouse usage
	WeightReading       = 0.8 // Reading / low-input activity
	WeightMeeting       = 1.5 // Video call / meeting fatigue
	WeightPassiveFocus  = 0.9 // Video lecture / media listening
)

// FatigueModel manages the scalar fatigue score F_t in [0, 100] and tracks dismissal habits.
type FatigueModel struct {
	mu                  sync.RWMutex
	currentScore        float64
	lastUpdated         time.Time
	continuousActiveMin float64
	rapidDismissals     int       // Count of dismissals occurring < 2s after presentation
	lastPromptTime      time.Time
	ambientModeOnly     bool      // Mitigates notification blindness by shifting to ambient mode
}

// NewFatigueModel initializes a fatigue model with baseline zero.
func NewFatigueModel() *FatigueModel {
	return &FatigueModel{
		currentScore: 0.0,
		lastUpdated:  time.Now(),
	}
}

// Score returns the current scalar fatigue score.
func (f *FatigueModel) Score() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return math.Round(f.currentScore*100) / 100
}

// ContinuousActiveMinutes returns the tracked unbroken active time in minutes.
func (f *FatigueModel) ContinuousActiveMinutes() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.continuousActiveMin
}

// IsAmbientModeOnly returns true if rapid dismissals forced ambient-only notifications.
func (f *FatigueModel) IsAmbientModeOnly() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ambientModeOnly
}

// RecordPromptDispatched marks when a prompt was shown.
func (f *FatigueModel) RecordPromptDispatched() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastPromptTime = time.Now()
}

// RecordDismissal tracks user prompt dismissal speed. If dismissed < 2 seconds, increments rapid count.
func (f *FatigueModel) RecordDismissal(elapsedSec float64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if elapsedSec < 2.0 {
		f.rapidDismissals++
		if f.rapidDismissals >= 3 {
			f.ambientModeOnly = true
		}
	} else {
		if f.rapidDismissals > 0 {
			f.rapidDismissals--
		}
		if f.rapidDismissals < 2 {
			f.ambientModeOnly = false
		}
	}
}

// ResetBreakTaken reduces fatigue score after a completed break.
func (f *FatigueModel) ResetBreakTaken(durationSec int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	decayAmount := float64(durationSec) / 6.0
	f.currentScore = math.Max(0.0, f.currentScore-decayAmount)
	f.continuousActiveMin = 0.0
	f.rapidDismissals = 0
	f.ambientModeOnly = false
}

// Update calculates the 30-second interval update for F_t.
func (f *FatigueModel) Update(snapshot *types.ContextSnapshot, state types.State, deltaSec float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	if deltaSec <= 0 {
		deltaSec = 30.0
	}

	if snapshot.IdleSeconds > 180 {
		idleMinutes := float64(snapshot.IdleSeconds) / 60.0
		decayRate := 1.5 * (deltaSec / 60.0) * (idleMinutes / 3.0)
		f.currentScore = math.Max(0.0, f.currentScore-decayRate)

		f.continuousActiveMin = math.Max(0.0, f.continuousActiveMin-(deltaSec/60.0))
		f.lastUpdated = time.Now()
		return f.currentScore
	}

	var wState float64
	switch state {
	case types.StateInMeeting:
		wState = WeightMeeting
	case types.StatePassiveFocus:
		wState = WeightPassiveFocus
	default:
		if snapshot.KeystrokeCadenceWPM > 40 || snapshot.IdleSeconds < 5 {
			wState = WeightHighIntensity
		} else {
			wState = WeightReading
		}
	}

	incrementalFatigue := (deltaSec / 60.0) * 1.0 * wState
	f.currentScore = math.Min(100.0, f.currentScore+incrementalFatigue)

	f.continuousActiveMin += (deltaSec / 60.0)
	f.lastUpdated = time.Now()

	return f.currentScore
}
