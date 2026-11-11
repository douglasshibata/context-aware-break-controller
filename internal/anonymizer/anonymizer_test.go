package anonymizer

import (
	"testing"
	"time"
)

func TestAnonymizer_ProcessInputTimestamp(t *testing.T) {
	anon := NewAnonymizer()

	now := time.Now()
	ev1 := anon.ProcessInputTimestamp(now)
	if ev1.IsTypingBurst {
		t.Errorf("First event should not be typing burst")
	}

	ev2 := anon.ProcessInputTimestamp(now.Add(200 * time.Millisecond))
	if !ev2.IsTypingBurst {
		t.Errorf("Subsequent rapid event (200ms) should be identified as typing burst")
	}

	ev3 := anon.ProcessInputTimestamp(now.Add(1 * time.Second))
	if ev3.IsTypingBurst {
		t.Errorf("Event after 800ms gap should not be typing burst")
	}
}

func TestAnonymizer_FalseCallDetection(t *testing.T) {
	anon := NewAnonymizer()

	// Case 1: Active call with normal speaking amplitude
	hw1 := anon.NormalizeHardwareState(true, true, true, 0.45, 0)
	if !hw1.MicInUse || hw1.IsSilentCallHold {
		t.Errorf("Expected active call to be marked in use and not silent hold")
	}

	// Case 2: Mic held open but peak amplitude at noise floor for > 10m (601 seconds)
	hw2 := anon.NormalizeHardwareState(true, false, false, 0.01, 601*time.Second)
	if hw2.MicInUse {
		t.Errorf("Mic should be downgraded to false due to >10m silence")
	}
	if !hw2.IsSilentCallHold {
		t.Errorf("IsSilentCallHold should be true")
	}

	// Case 3: Mic held open but silence is only 2 minutes (< 10m)
	hw3 := anon.NormalizeHardwareState(true, false, false, 0.01, 120*time.Second)
	if !hw3.MicInUse {
		t.Errorf("Mic should still be true since silence duration is under 10m threshold")
	}
}
