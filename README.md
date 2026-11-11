# 🧠 Context-Aware Workspace Well-Being & Break Controller

[![Go](https://img.shields.io/badge/Language-Go%201.24%2B-blue)](#)
[![Privacy](https://img.shields.io/badge/Privacy-100%25%20Local-green)](#)
[![Architecture](https://img.shields.io/badge/Architecture-Edge%20Daemon%20%7C%20Heuristic%20Fatigue-purple)](#)
[![License](https://img.shields.io/badge/License-MIT-yellow)](#)

> **"Replace static, interruptive interval timers with an adaptive edge daemon that monitors user presence, software context, and physical fatigue signals locally to trigger health interventions exclusively during natural cognitive transition points."**

---

## 📐 System Architecture

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                      Local OS Environment                              │
 │                                                                        │
 │  ┌──────────────────────┐  ┌─────────────────────┐  ┌───────────────┐  │
 │  │ Hardware State       │  │ Activity Metrics    │  │ User Context  │  │
 │  │ - Mic/Cam Active     │  │ - Input Idle Time   │  │ - Fullscreen  │  │
 │  │ - Audio Capture Flag │  │ - Keystroke Cadence │  │ - App Class   │  │
 │  └──────────┬───────────┘  └──────────┬──────────┘  └───────┬───────┘  │
 │             │                         │                     │          │
 │             └─────────────────┐       │      ┌──────────────┘          │
 │                               ▼       ▼      ▼                         │
 │                        ┌────────────────────────────┐                  │
 │                        │    Signal Collector &      │                  │
 │                        │    Normalization Engine    │                  │
 │                        └──────────────┬─────────────┘                  │
 │                                       ▼                                │
 │                        ┌────────────────────────────┐                  │
 │                        │ State Machine & Heuristic  │                  │
 │                        │       Fatigue Model        │                  │
 │                        └──────────────┬─────────────┘                  │
 │                                       ▼                                │
 │                        ┌────────────────────────────┐                  │
 │                        │ Context-Aware Interrupter  │                  │
 │                        └──────────────┬─────────────┘                  │
 └───────────────────────────────────────┼────────────────────────────────┘
                                         ▼
                 ┌───────────────────────────────────────────────┐
                 │          Presentation & Action Layer          │
                 │  - Native Toast Notifications (Win32 / macOS) │
                 │  - Ambient Tray Display & Subtle Pings        │
                 │  - Audio Reminders (Optional Chimes)          │
                 └───────────────────────────────────────────────┘
```

---

## 🔬 Core Subsystems & Detection Mechanics

### 1. Hardware State & Zero-Audio Sniffing
The daemon avoids recording or listening to audio/video buffers:
- **Microphone & Camera Hooks:** Queries Windows Registry `HKCU\...\CapabilityAccessManager\ConsentStore` or macOS CoreAudio property listeners where active stream handles hold `LastUsedTimeStop == 0`.
- **Active Audio Session State:** Queries `IAudioSessionManager2` / `eCommunications` role.

### 2. Screen & Interaction Context Sensor
- **Presentation & Fullscreen Detection:** Inspects `SHQueryUserNotificationState(&state)`. Automatically suppresses notifications if:
  - `QUNS_PRESENTATION_MODE`: Presentation active (PowerPoint, Keynote).
  - `QUNS_RUNNING_D3D_FULL_SCREEN`: Fullscreen game or video player.
  - `QUNS_BUSY`: Fullscreen exclusive application.
- **Idle Tracking:** Polls `GetLastInputInfo()` (Windows) or event deltas without keyloggers.

### 3. Heuristic Fatigue Model ($F_t \in [0, 100]$)
Updated every 30 seconds:
- $w_{state}$ Exertion Weights:
  - High-intensity typing/mouse: `1.2`
  - Reading / low-input activity: `0.8`
  - Video call / meeting: `1.5`
  - Passive focus (video/media): `0.9`
- $D_{idle}$ Passive Decay Rate:
  - Drops proportionally when user steps away for $> 3$ minutes.

---

## 🔄 State Machine Transition Matrix

| Current State | Primary Event / Signal | Next State | Action Taken |
|---|---|---|---|
| `IDLE` | Mouse or keyboard input detected | `ACTIVE` | Start session timer; track focus baseline. |
| `ACTIVE` | Microphone/Camera in use OR Calendar event start | `IN_MEETING` | Mute notifications; start meeting duration timer. |
| `ACTIVE` | Input idle for > 3 minutes | `AWAY` | Pause work timer; decay fatigue score; log natural break. |
| `ACTIVE` | Idle > 3 min, audio playing, browser/media active | `PASSIVE_FOCUS` | Mitigate false idle during video lectures / long docs. |
| `IN_MEETING` | Mic/Cam shut off AND Call window closed | `TRANSITION_WINDOW` | Prime opportunity for break. Trigger post-meeting reset. |
| `ACTIVE` | Fatigue Score > 75 AND 15–45s input pause | `OPPORTUNE_BREAK` | Fire non-modal break suggestion (micro-pause or water). |
| `ACTIVE` | Fatigue Score > 90 AND duration > 90 min | `HARD_BREAK_DUE` | Escalated, prominent notification requiring acknowledgement. |

---

## 🛡️ Types of Interventions

1. **Micro-Pause (20 seconds):**
   - *Trigger:* Continuous active typing for 25 minutes without idle gaps, or fatigue $> 75$ with a natural 15–45s pause.
   - *Action:* Look away from the screen at an object 20 feet away for 20 seconds.
2. **Post-Meeting Decompression (2 to 3 minutes):**
   - *Trigger:* Detected end of a meeting that lasted $\ge 30$ minutes.
   - *Action:* "Meeting finished. Stand up, stretch your shoulders, and drink water before opening your next task."
3. **Ergonomic Full Reset (5 to 10 minutes):**
   - *Trigger:* Continuous active session hitting 90 minutes without an away period.
   - *Action:* High-priority dialog offering a 5-minute break countdown.

---

## 🔒 Local Privacy Architecture

```
[ Raw OS Signals: Keystroke timestamps, registry polling, camera flags ]
                                 │
                                 ▼
                     (In-Memory Anonymizer)
   - Discards all key values (retains only delta-time between inputs)
   - Discards all audio/video buffers (retains only boolean: IsActive)
   - Runs zero cloud analytics; no socket connection leaves localhost
                                 │
                                 ▼
                   (Encrypted Local SQLite DB)
   - Schema: date | active_minutes | meetings_count | breaks_taken
   - Database remains exclusively on local machine filesystem
```

---

## ⚡ Failure Modes & Mitigations

1. **Reading Long Docs / Watching Lectures:**
   - *Mitigation:* System checks active audio output. When detected with browser/media in foreground, transitions to `PASSIVE_FOCUS` rather than `AWAY`.
2. **Notification Blindness & Micro-Break Fatigue:**
   - *Mitigation:* Rapid dismissals ($< 2$ seconds) increment a counter. At $\ge 3$ rapid dismissals, the daemon shifts to ambient tray cues (`AMBIENT_TRAY`) instead of popups.
3. **False Call Detection:**
   - *Mitigation:* Background apps holding mic open at the noise floor ($\le 0.02$) for $> 10$ minutes are downgraded to idle.

---

## 💻 CLI Usage & Commands

```bash
# Display today's local wellness statistics
go run ./cmd/cli stats

# Run cognitive transition simulations
go run ./cmd/cli simulate --scenario meeting-end
go run ./cmd/cli simulate --scenario typing-burst
go run ./cmd/cli simulate --scenario dnd-suppression
go run ./cmd/cli simulate --scenario rapid-dismissal

# Dispatch test alerts
go run ./cmd/cli test-alert --tier micro
go run ./cmd/cli test-alert --tier post-meeting
go run ./cmd/cli test-alert --tier ergonomic
```
