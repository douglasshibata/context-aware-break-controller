package notification

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/user/context-aware-break-controller/pkg/types"
)

// Notifier abstracts desktop and ambient break alerts.
type Notifier interface {
	Dispatch(ctx context.Context, intervention *types.Intervention) error
	GetLastDispatched() *types.Intervention
}

// SystemNotifier delivers alerts natively via desktop notifications or ambient console logs.
type SystemNotifier struct {
	mu             sync.Mutex
	lastDispatched *types.Intervention
}

// NewSystemNotifier creates a new desktop notifier.
func NewSystemNotifier() *SystemNotifier {
	return &SystemNotifier{}
}

// Dispatch routes the notification based on delivery mode and OS capabilities.
func (n *SystemNotifier) Dispatch(ctx context.Context, interv *types.Intervention) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastDispatched = interv

	switch interv.Delivery {
	case types.DeliveryAmbientTray:
		fmt.Printf("🌿 [AMBIENT TRAY PING] %s: %s (Reason: %s)\n", interv.Title, interv.Message, interv.TriggerReason)
		return nil

	case types.DeliverySubtleToast, types.DeliveryModalDialog:
		return n.sendNativeToast(interv)

	default:
		fmt.Printf("🔔 [WELLBEING] %s: %s\n", interv.Title, interv.Message)
		return nil
	}
}

// GetLastDispatched returns the most recent dispatched intervention.
func (n *SystemNotifier) GetLastDispatched() *types.Intervention {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.lastDispatched
}

func (n *SystemNotifier) sendNativeToast(interv *types.Intervention) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, interv.Message, interv.Title)
		return exec.Command("osascript", "-e", script).Run()

	case "linux":
		if path, err := exec.LookPath("notify-send"); err == nil {
			return exec.Command(path, "-u", "normal", interv.Title, interv.Message).Run()
		}
		fmt.Printf("🔔 [DESKTOP TOAST] %s: %s\n", interv.Title, interv.Message)
		return nil

	case "windows":
		psScript := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; Write-Output "%s: %s"`, interv.Title, interv.Message)
		_ = exec.Command("powershell", "-NoProfile", "-Command", psScript).Run()
		fmt.Printf("🔔 [WIN TOAST] %s: %s\n", interv.Title, interv.Message)
		return nil

	default:
		fmt.Printf("🔔 [NOTIFICATION] %s: %s\n", interv.Title, interv.Message)
		return nil
	}
}
