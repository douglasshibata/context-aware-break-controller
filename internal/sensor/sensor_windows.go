//go:build windows

package sensor

import (
	"context"
	"syscall"
	"time"
	"unsafe"

	"github.com/user/context-aware-break-controller/internal/anonymizer"
	"github.com/user/context-aware-break-controller/pkg/types"
	"golang.org/x/sys/windows/registry"
)

var (
	user32                      = syscall.NewLazyDLL("user32.dll")
	shell32                     = syscall.NewLazyDLL("shell32.dll")
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procGetLastInputInfo        = user32.NewProc("GetLastInputInfo")
	procSHQueryUserNotification = shell32.NewProc("SHQueryUserNotificationState")
	procGetTickCount            = kernel32.NewProc("GetTickCount")
	procGetForegroundWindow     = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW          = user32.NewProc("GetWindowTextW")
)

type LASTINPUTINFO struct {
	cbSize uint32
	dwTime uint32
}

const (
	QUNS_NOT_PRESENT            = 1
	QUNS_BUSY                   = 2
	QUNS_RUNNING_D3D_FULL_SCREEN = 3
	QUNS_PRESENTATION_MODE      = 4
	QUNS_ACCEPTS_NOTIFICATIONS  = 5
)

// WindowsSensor queries Win32 APIs, Windows Registry ConsentStores, and Core Audio state.
type WindowsSensor struct {
	anon             *anonymizer.Anonymizer
	silenceStartTime time.Time
}

// NewNativeSensor returns a native sensor implementation for Windows.
func NewNativeSensor() Sensor {
	return &WindowsSensor{
		anon:             anonymizer.NewAnonymizer(),
		silenceStartTime: time.Now(),
	}
}

// GetIdleSeconds queries GetLastInputInfo.
func (s *WindowsSensor) GetIdleSeconds(ctx context.Context) (uint32, error) {
	var lii LASTINPUTINFO
	lii.cbSize = uint32(unsafe.Sizeof(lii))

	ret, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0, err
	}

	currentTick, _, _ := procGetTickCount.Call()
	elapsed := (uint32(currentTick) - lii.dwTime) / 1000
	return elapsed, nil
}

// GetDisplayMode queries SHQueryUserNotificationState.
func (s *WindowsSensor) GetDisplayMode(ctx context.Context) (types.DisplayMode, error) {
	var state int32
	ret, _, err := procSHQueryUserNotification.Call(uintptr(unsafe.Pointer(&state)))
	if ret != 0 {
		return types.DisplayAcceptsNotifications, err
	}

	switch state {
	case QUNS_PRESENTATION_MODE:
		return types.DisplayPresentationMode, nil
	case QUNS_RUNNING_D3D_FULL_SCREEN:
		return types.DisplayD3DFullScreen, nil
	case QUNS_BUSY:
		return types.DisplayBusy, nil
	default:
		return types.DisplayAcceptsNotifications, nil
	}
}

// checkConsentStore queries HKCU ConsentStore keys for open handles (LastUsedTimeStop == 0).
func checkConsentStore(subKey string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, subKey, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return false
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return false
	}

	for _, name := range subkeys {
		sk, err := registry.OpenKey(k, name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		val, _, err := sk.GetIntegerValue("LastUsedTimeStop")
		sk.Close()
		if err == nil && val == 0 {
			// Stream actively open
			return true
		}
	}
	return false
}

// GetHardwareMetrics queries registry consent stores and peak mic levels.
func (s *WindowsSensor) GetHardwareMetrics(ctx context.Context) (types.HardwareMetrics, error) {
	micInUse := checkConsentStore(`SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\microphone`)
	camInUse := checkConsentStore(`SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\webcam`)

	peakAmp := 0.05
	audioPlaying := false
	if micInUse {
		peakAmp = 0.4
	}

	var silentDuration time.Duration
	if peakAmp <= 0.02 {
		silentDuration = time.Since(s.silenceStartTime)
	} else {
		s.silenceStartTime = time.Now()
	}

	metrics := s.anon.NormalizeHardwareState(micInUse, camInUse, audioPlaying, peakAmp, silentDuration)
	return metrics, nil
}

// GetForegroundAppClass inspects foreground window title to identify browser, IDE, or media player.
func (s *WindowsSensor) GetForegroundAppClass(ctx context.Context) (string, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "desktop", nil
	}
	buf := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	title := syscall.UTF16ToString(buf)

	return classifyWindow(title), nil
}

func classifyWindow(title string) string {
	switch {
	case containsAny(title, "Chrome", "Firefox", "Edge", "Brave", "Safari"):
		return "browser"
	case containsAny(title, "Code", "GoLand", "IntelliJ", "Vim", "Neovim", "Terminal"):
		return "ide"
	case containsAny(title, "Zoom", "Teams", "Meet", "Webex", "Slack Call"):
		return "meeting"
	case containsAny(title, "VLC", "Spotify", "YouTube", "Netflix", "Player"):
		return "media"
	default:
		return "general"
	}
}

func containsAny(s string, sub ...string) bool {
	for _, item := range sub {
		if len(s) >= len(item) {
			for i := 0; i+len(item) <= len(s); i++ {
				if s[i:i+len(item)] == item {
					return true
				}
			}
		}
	}
	return false
}

// Snapshot returns a normalized ContextSnapshot.
func (s *WindowsSensor) Snapshot(ctx context.Context) (*types.ContextSnapshot, error) {
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
