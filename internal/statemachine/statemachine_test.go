package statemachine

import (
	"testing"
	"time"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/pkg/types"
)

func TestStateMachine_Transitions(t *testing.T) {
	fm := fatigue.NewFatigueModel()
	sm := NewStateMachine(fm)

	if sm.CurrentState() != types.StateIdle {
		t.Fatalf("Expected initial state IDLE, got %v", sm.CurrentState())
	}

	snapActive := &types.ContextSnapshot{
		IdleSeconds: 3,
	}
	state, _ := sm.Step(snapActive)
	if state != types.StateActive {
		t.Errorf("Expected transition to ACTIVE, got %v", state)
	}

	snapMeeting := &types.ContextSnapshot{
		IdleSeconds: 30,
		Hardware: types.HardwareMetrics{
			MicInUse: true,
		},
	}
	state, _ = sm.Step(snapMeeting)
	if state != types.StateInMeeting {
		t.Errorf("Expected transition to IN_MEETING, got %v", state)
	}

	sm.meetingStartTime = time.Now().Add(-35 * time.Minute)

	snapMeetingEnd := &types.ContextSnapshot{
		IdleSeconds: 5,
		Hardware: types.HardwareMetrics{
			MicInUse: false,
		},
	}
	state, interv := sm.Step(snapMeetingEnd)
	if state != types.StateTransitionWindow {
		t.Errorf("Expected transition to TRANSITION_WINDOW, got %v", state)
	}
	if interv != types.InterventionPostMeetingReset {
		t.Errorf("Expected POST_MEETING_RESET intervention, got %v", interv)
	}

	state, _ = sm.Step(&types.ContextSnapshot{IdleSeconds: 2})
	if state != types.StateActive {
		t.Errorf("Expected transition to ACTIVE, got %v", state)
	}

	snapAway := &types.ContextSnapshot{
		IdleSeconds: 210,
		Hardware: types.HardwareMetrics{
			AudioPlaying: false,
		},
	}
	state, _ = sm.Step(snapAway)
	if state != types.StateAway {
		t.Errorf("Expected transition to AWAY, got %v", state)
	}
}

func TestStateMachine_PassiveFocusMitigation(t *testing.T) {
	fm := fatigue.NewFatigueModel()
	sm := NewStateMachine(fm)

	sm.Step(&types.ContextSnapshot{IdleSeconds: 2})

	snapPassive := &types.ContextSnapshot{
		IdleSeconds:        240,
		ForegroundAppClass: "browser",
		Hardware: types.HardwareMetrics{
			AudioPlaying: true,
		},
	}
	state, _ := sm.Step(snapPassive)
	if state != types.StatePassiveFocus {
		t.Errorf("Expected transition to PASSIVE_FOCUS, got %v", state)
	}
}
