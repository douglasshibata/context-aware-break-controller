package types

import "time"

// State represents the user's operational and fatigue state in the state machine.
type State string

const (
	StateIdle             State = "IDLE"
	StateActive           State = "ACTIVE"
	StateInMeeting        State = "IN_MEETING"
	StateAway             State = "AWAY"
	StateTransitionWindow State = "TRANSITION_WINDOW"
	StateOpportuneBreak   State = "OPPORTUNE_BREAK"
	StateHardBreakDue     State = "HARD_BREAK_DUE"
	StatePassiveFocus     State = "PASSIVE_FOCUS"
)

// InterventionType designates the tier of well-being intervention triggered.
type InterventionType string

const (
	InterventionMicroPause       InterventionType = "MICRO_PAUSE"
	InterventionPostMeetingReset InterventionType = "POST_MEETING_RESET"
	InterventionErgonomicReset   InterventionType = "ERGONOMIC_RESET"
	InterventionNone             InterventionType = "NONE"
)

// InterventionDelivery designates whether the intervention is modal, non-modal, or ambient.
type InterventionDelivery string

const (
	DeliveryAmbientTray InterventionDelivery = "AMBIENT_TRAY"
	DeliverySubtleToast InterventionDelivery = "SUBTLE_TOAST"
	DeliveryModalDialog InterventionDelivery = "MODAL_DIALOG"
)

// Intervention represents an actionable break recommendation.
type Intervention struct {
	Type          InterventionType     `json:"type"`
	Delivery      InterventionDelivery `json:"delivery"`
	DurationSec   int                  `json:"duration_sec"`
	Title         string               `json:"title"`
	Message       string               `json:"message"`
	ActionPrompt  string               `json:"action_prompt"`
	TriggerReason string               `json:"trigger_reason"`
	CreatedAt     time.Time            `json:"created_at"`
}

// UserInputEvent carries anonymous input delta information without key codes.
type UserInputEvent struct {
	DeltaSinceLastInput time.Duration `json:"delta_since_last_input"`
	IsTypingBurst       bool          `json:"is_typing_burst"`
	Timestamp           time.Time     `json:"timestamp"`
}

// HardwareMetrics captures hardware handles without sniffing audio/video payloads.
type HardwareMetrics struct {
	MicInUse         bool    `json:"mic_in_use"`
	CameraInUse      bool    `json:"camera_in_use"`
	AudioPlaying     bool    `json:"audio_playing"`
	PeakMicAmplitude float64 `json:"peak_mic_amplitude"` // Range: 0.0 (silence) to 1.0 (loud)
	IsSilentCallHold bool    `json:"is_silent_call_hold"` // True if mic held open but silent > 10m
}

// DisplayMode reflects OS-level user notification / fullscreen states.
type DisplayMode string

const (
	DisplayAcceptsNotifications DisplayMode = "ACCEPTS_NOTIFICATIONS"
	DisplayPresentationMode    DisplayMode = "PRESENTATION_MODE"
	DisplayD3DFullScreen        DisplayMode = "RUNNING_D3D_FULL_SCREEN"
	DisplayBusy                 DisplayMode = "BUSY"
)

// ContextSnapshot represents a normalized reading of all local edge signals at a point in time.
type ContextSnapshot struct {
	Timestamp           time.Time       `json:"timestamp"`
	IdleSeconds         uint32          `json:"idle_seconds"`
	DisplayMode         DisplayMode     `json:"display_mode"`
	Hardware            HardwareMetrics `json:"hardware"`
	ForegroundAppClass  string          `json:"foreground_app_class"` // e.g., "browser", "ide", "media", "meeting"
	KeystrokeCadenceWPM float64         `json:"keystroke_cadence_wpm"`
	CalendarEventActive bool            `json:"calendar_event_active"`
}

// DailyWellnessSummary stores aggregated local stats for the encrypted SQLite database.
type DailyWellnessSummary struct {
	Date            string    `json:"date"`              // YYYY-MM-DD
	ActiveMinutes   int       `json:"active_minutes"`    // Cumulative productive focus
	MeetingsCount   int       `json:"meetings_count"`    // Completed meetings
	BreaksTaken     int       `json:"breaks_taken"`      // Acknowledged breaks
	FatigueScoreAvg float64   `json:"fatigue_score_avg"` // Daily average F_t
	DismissalsCount int       `json:"dismissals_count"`  // Count of rapid dismissals (<2s)
	LastUpdatedAt   time.Time `json:"last_updated_at"`
}
