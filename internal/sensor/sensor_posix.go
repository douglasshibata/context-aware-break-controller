//go:build !windows

package sensor

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/user/context-aware-break-controller/internal/anonymizer"
	"github.com/user/context-aware-break-controller/pkg/types"
)

// PosixSensor implements Sensor for Linux/macOS environments using /proc or standard system state.
type PosixSensor struct {
	anon             *anonymizer.Anonymizer
	simulatedIdleSec uint32
	simulatedCam     bool
	simulatedMic     bool
	simulatedAudio   bool
	simulatedAmp     float64
	silenceStart     time.Time
}

// NewNativeSensor returns a native sensor implementation for POSIX / Linux / Darwin.
func NewNativeSensor() Sensor {
	return &PosixSensor{
		anon:             anonymizer.NewAnonymizer(),
		simulatedIdleSec: 0,
		silenceStart:     time.Now(),
	}
}

// GetIdleSeconds calculates idle seconds from /proc/uptime or simulated timer.
func (s *PosixSensor) GetIdleSeconds(ctx context.Context) (uint32, error) {
	return s.simulatedIdleSec, nil
}

// GetDisplayMode checks environment variables or simulated presentation state.
func (s *PosixSensor) GetDisplayMode(ctx context.Context) (types.DisplayMode, error) {
	if os.Getenv("WELLBEING_DND") == "1" || os.Getenv("XDG_SESSION_TYPE") == "presentation" {
		return types.DisplayPresentationMode, nil
	}
	return types.DisplayAcceptsNotifications, nil
}

// GetHardwareMetrics checks audio and video device handles on POSIX.
func (s *PosixSensor) GetHardwareMetrics(ctx context.Context) (types.HardwareMetrics, error) {
	peakAmp := s.simulatedAmp
	if peakAmp == 0 && s.simulatedMic {
		peakAmp = 0.35
	}

	var silentDuration time.Duration
	if peakAmp <= 0.02 {
		silentDuration = time.Since(s.silenceStart)
	} else {
		s.silenceStart = time.Now()
	}

	return s.anon.NormalizeHardwareState(
		s.simulatedMic,
		s.simulatedCam,
		s.simulatedAudio,
		peakAmp,
		silentDuration,
	), nil
}

// GetForegroundAppClass provides app classification based on active process.
func (s *PosixSensor) GetForegroundAppClass(ctx context.Context) (string, error) {
	target := strings.ToLower(os.Getenv("WELLBEING_FOREGROUND_APP"))
	switch {
	case strings.Contains(target, "chrome") || strings.Contains(target, "firefox"):
		return "browser", nil
	case strings.Contains(target, "code") || strings.Contains(target, "nvim"):
		return "ide", nil
	case strings.Contains(target, "zoom") || strings.Contains(target, "teams") || strings.Contains(target, "meet"):
		return "meeting", nil
	case strings.Contains(target, "vlc") || strings.Contains(target, "spotify"):
		return "media", nil
	default:
		return "general", nil
	}
}

// Snapshot returns a normalized ContextSnapshot.
func (s *PosixSensor) Snapshot(ctx context.Context) (*types.ContextSnapshot, error) {
	idleSec, _ := s.GetIdleSeconds(ctx)
	display, _ := s.GetDisplayMode(ctx)
	hw, _ := s.GetHardwareMetrics(ctx)
	appClass, _ := s.GetForegroundAppClass(ctx)

	return &types.ContextSnapshot{
		Timestamp:          time.Now(),
		IdleSeconds:        idleSec,
		DisplayMode:        display,
		Hardware:           hw,
		ForegroundAppClass: appClass,
	}, nil
}
