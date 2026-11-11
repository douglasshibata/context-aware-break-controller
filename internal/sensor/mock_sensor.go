package sensor

import (
	"context"
	"time"

	"github.com/user/context-aware-break-controller/internal/anonymizer"
	"github.com/user/context-aware-break-controller/pkg/types"
)

// MockSensor allows precise deterministic tests of all edge cases and sensor inputs.
type MockSensor struct {
	anon             *anonymizer.Anonymizer
	IdleSecondsVal   uint32
	DisplayModeVal   types.DisplayMode
	MicInUseVal      bool
	CamInUseVal      bool
	AudioPlayingVal  bool
	PeakMicAmpVal    float64
	SilentDuration   time.Duration
	ForegroundClass  string
	KeystrokeCadence float64
	CalendarEventVal bool
}

// NewMockSensor returns a new configurable mock sensor.
func NewMockSensor() *MockSensor {
	return &MockSensor{
		anon:            anonymizer.NewAnonymizer(),
		DisplayModeVal:  types.DisplayAcceptsNotifications,
		ForegroundClass: "ide",
	}
}

func (m *MockSensor) GetIdleSeconds(ctx context.Context) (uint32, error) {
	return m.IdleSecondsVal, nil
}

func (m *MockSensor) GetDisplayMode(ctx context.Context) (types.DisplayMode, error) {
	return m.DisplayModeVal, nil
}

func (m *MockSensor) GetHardwareMetrics(ctx context.Context) (types.HardwareMetrics, error) {
	return m.anon.NormalizeHardwareState(
		m.MicInUseVal,
		m.CamInUseVal,
		m.AudioPlayingVal,
		m.PeakMicAmpVal,
		m.SilentDuration,
	), nil
}

func (m *MockSensor) GetForegroundAppClass(ctx context.Context) (string, error) {
	return m.ForegroundClass, nil
}

func (m *MockSensor) Snapshot(ctx context.Context) (*types.ContextSnapshot, error) {
	hw, _ := m.GetHardwareMetrics(ctx)
	return &types.ContextSnapshot{
		Timestamp:           time.Now(),
		IdleSeconds:         m.IdleSecondsVal,
		DisplayMode:         m.DisplayModeVal,
		Hardware:            hw,
		ForegroundAppClass:  m.ForegroundClass,
		KeystrokeCadenceWPM: m.KeystrokeCadence,
		CalendarEventActive: m.CalendarEventVal,
	}, nil
}
