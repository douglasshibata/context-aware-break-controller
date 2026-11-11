package sensor

import (
	"context"

	"github.com/user/context-aware-break-controller/pkg/types"
)

// Sensor provides OS hardware, screen, and input state monitoring without invasive sniffing.
type Sensor interface {
	GetIdleSeconds(ctx context.Context) (uint32, error)
	GetDisplayMode(ctx context.Context) (types.DisplayMode, error)
	GetHardwareMetrics(ctx context.Context) (types.HardwareMetrics, error)
	GetForegroundAppClass(ctx context.Context) (string, error)
	Snapshot(ctx context.Context) (*types.ContextSnapshot, error)
}
