package statemachine

import (
	"sync"
	"time"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/pkg/types"
)

// StateMachine evaluates local edge signals and controls wellness state transitions every 10 seconds.
type StateMachine struct {
	mu                  sync.RWMutex
	currentState        types.State
	previousState       types.State
	sessionStartTime    time.Time
	meetingStartTime    time.Time
	meetingDuration     time.Duration
	fatigueModel        *fatigue.FatigueModel
	pendingIntervention types.InterventionType
	lastStateChange     time.Time
}

// NewStateMachine initializes a state machine at StateIdle.
func NewStateMachine(fm *fatigue.FatigueModel) *StateMachine {
	now := time.Now()
	return &StateMachine{
		currentState:     types.StateIdle,
		previousState:    types.StateIdle,
		sessionStartTime: now,
		fatigueModel:     fm,
		lastStateChange:  now,
	}
}

// CurrentState returns the current state.
func (sm *StateMachine) CurrentState() types.State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// MeetingDuration returns the duration of the current or last completed meeting.
func (sm *StateMachine) MeetingDuration() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.meetingDuration
}

// SetMeetingDurationForTest overrides meeting duration for testing and CLI simulations.
func (sm *StateMachine) SetMeetingDurationForTest(d time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.meetingStartTime = time.Now().Add(-d)
}

// Step evaluates a ContextSnapshot every 10 seconds and executes transitions according to the specification.
func (sm *StateMachine) Step(snapshot *types.ContextSnapshot) (types.State, types.InterventionType) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	fatigueScore := sm.fatigueModel.Score()
	activeMinutes := sm.fatigueModel.ContinuousActiveMinutes()
	idleSec := snapshot.IdleSeconds

	var nextState = sm.currentState
	var intervention = types.InterventionNone

	isPassiveFocus := idleSec > 180 && snapshot.Hardware.AudioPlaying &&
		(snapshot.ForegroundAppClass == "browser" || snapshot.ForegroundAppClass == "media")

	switch sm.currentState {
	case types.StateIdle:
		if idleSec < 15 {
			nextState = types.StateActive
			sm.sessionStartTime = now
		}

	case types.StateActive:
		if (snapshot.Hardware.MicInUse || snapshot.Hardware.CameraInUse || snapshot.CalendarEventActive) && !snapshot.Hardware.IsSilentCallHold {
			nextState = types.StateInMeeting
			sm.meetingStartTime = now
			break
		}

		if idleSec > 180 {
			if isPassiveFocus {
				nextState = types.StatePassiveFocus
			} else {
				nextState = types.StateAway
			}
			break
		}

		if fatigueScore >= 90.0 && activeMinutes >= 90.0 {
			nextState = types.StateHardBreakDue
			intervention = types.InterventionErgonomicReset
			break
		}

		if fatigueScore >= 75.0 && idleSec >= 15 && idleSec <= 45 {
			nextState = types.StateOpportuneBreak
			intervention = types.InterventionMicroPause
			break
		}

		if activeMinutes >= 25.0 && idleSec >= 15 && idleSec <= 45 {
			intervention = types.InterventionMicroPause
		}

	case types.StatePassiveFocus:
		if idleSec < 15 {
			nextState = types.StateActive
			break
		}
		if !snapshot.Hardware.AudioPlaying && idleSec > 180 {
			nextState = types.StateAway
			break
		}

	case types.StateInMeeting:
		if !snapshot.Hardware.MicInUse && !snapshot.Hardware.CameraInUse && !snapshot.CalendarEventActive {
			sm.meetingDuration = time.Since(sm.meetingStartTime)
			nextState = types.StateTransitionWindow
			if sm.meetingDuration >= 30*time.Minute {
				intervention = types.InterventionPostMeetingReset
			}
		}

	case types.StateTransitionWindow:
		if idleSec < 15 {
			nextState = types.StateActive
		} else if idleSec > 180 {
			nextState = types.StateAway
		}

	case types.StateOpportuneBreak:
		if idleSec < 10 {
			nextState = types.StateActive
		} else if idleSec > 180 {
			nextState = types.StateAway
		}

	case types.StateHardBreakDue:
		if idleSec > 180 {
			nextState = types.StateAway
		}

	case types.StateAway:
		if idleSec < 15 {
			nextState = types.StateActive
			sm.sessionStartTime = now
		}
	}

	if nextState != sm.currentState {
		sm.previousState = sm.currentState
		sm.currentState = nextState
		sm.lastStateChange = now
	}

	return sm.currentState, intervention
}
