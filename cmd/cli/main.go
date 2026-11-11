package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/user/context-aware-break-controller/internal/fatigue"
	"github.com/user/context-aware-break-controller/internal/interrupter"
	"github.com/user/context-aware-break-controller/internal/notification"
	"github.com/user/context-aware-break-controller/internal/sensor"
	"github.com/user/context-aware-break-controller/internal/statemachine"
	"github.com/user/context-aware-break-controller/internal/storage"
	"github.com/user/context-aware-break-controller/pkg/types"
)

func main() {
	statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
	simCmd := flag.NewFlagSet("simulate", flag.ExitOnError)
	testAlertCmd := flag.NewFlagSet("test-alert", flag.ExitOnError)

	dbPath := flag.String("db", "wellness_local.db", "Path to SQLite database")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "stats":
		_ = statsCmd.Parse(os.Args[2:])
		store, err := storage.NewSQLiteStorage(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		today := time.Now().Format("2006-01-02")
		summary, err := store.GetDailySummary(ctx, today)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying summary: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("📊 Local Well-Being Summary for Today (" + today + ")")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("⏱️  Active Focus Minutes : %d min\n", summary.ActiveMinutes)
		fmt.Printf("📞 Meetings Completed   : %d\n", summary.MeetingsCount)
		fmt.Printf("🍵 Breaks Taken         : %d\n", summary.BreaksTaken)
		fmt.Printf("⚡ Avg Fatigue Score    : %.2f / 100\n", summary.FatigueScoreAvg)
		fmt.Printf("🚫 Rapid Dismissals (<2s): %d\n", summary.DismissalsCount)
		fmt.Printf("🕒 Last Updated At      : %s\n", summary.LastUpdatedAt.Format("15:04:05"))

	case "simulate":
		scenario := simCmd.String("scenario", "meeting-end", "Scenario to simulate: meeting-end | typing-burst | dnd-suppression | rapid-dismissal")
		_ = simCmd.Parse(os.Args[2:])

		runSimulation(*scenario)

	case "test-alert":
		tier := testAlertCmd.String("tier", "micro", "Intervention tier: micro | post-meeting | ergonomic")
		_ = testAlertCmd.Parse(os.Args[2:])

		notifier := notification.NewSystemNotifier()
		var interv types.Intervention

		switch *tier {
		case "post-meeting":
			interv = types.Intervention{
				Type:          types.InterventionPostMeetingReset,
				Delivery:      types.DeliverySubtleToast,
				DurationSec:   180,
				Title:         "Meeting Decompression (2-3 min)",
				Message:       "Call concluded. Stand up, stretch your shoulders, and drink water.",
				ActionPrompt:  "Stretch & rehydrate",
				TriggerReason: "CLI test-alert trigger",
				CreatedAt:     time.Now(),
			}
		case "ergonomic":
			interv = types.Intervention{
				Type:          types.InterventionErgonomicReset,
				Delivery:      types.DeliveryModalDialog,
				DurationSec:   300,
				Title:         "Ergonomic Reset Due (5 min)",
				Message:       "Continuous active session reached 90 minutes. Step away from your desk.",
				ActionPrompt:  "Step away from workstation",
				TriggerReason: "CLI test-alert trigger",
				CreatedAt:     time.Now(),
			}
		default:
			interv = types.Intervention{
				Type:          types.InterventionMicroPause,
				Delivery:      types.DeliverySubtleToast,
				DurationSec:   20,
				Title:         "Cognitive Pause (20s)",
				Message:       "Look out the window or at an object 20 feet away for 20 seconds.",
				ActionPrompt:  "Look away for 20s",
				TriggerReason: "CLI test-alert trigger",
				CreatedAt:     time.Now(),
			}
		}

		_ = notifier.Dispatch(ctx, &interv)
		fmt.Printf("Dispatched %s test alert successfully.\n", *tier)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: break-ctl <command> [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  stats        Display today's local privacy wellness statistics")
	fmt.Println("  simulate     Run end-to-end cognitive transition simulations")
	fmt.Println("               (--scenario meeting-end | typing-burst | dnd-suppression | rapid-dismissal)")
	fmt.Println("  test-alert   Dispatch test notification (--tier micro | post-meeting | ergonomic)")
}

func runSimulation(scenario string) {
	fmt.Printf("🚀 Running simulation scenario: '%s'\n", scenario)
	fm := fatigue.NewFatigueModel()
	sm := statemachine.NewStateMachine(fm)
	intr := interrupter.NewInterrupter(fm)
	mock := sensor.NewMockSensor()
	ctx := context.Background()

	switch scenario {
	case "meeting-end":
		// User starts active
		mock.IdleSecondsVal = 2
		snapStart, _ := mock.Snapshot(ctx)
		sm.Step(snapStart)

		// User in call with camera and mic
		mock.MicInUseVal = true
		mock.CamInUseVal = true
		mock.PeakMicAmpVal = 0.4
		snapCall, _ := mock.Snapshot(ctx)
		sm.Step(snapCall)
		fmt.Println("1. In meeting: State =", sm.CurrentState())

		// Simulate call lasted 35 minutes
		sm.SetMeetingDurationForTest(35 * time.Minute)

		// Meeting ends after 30+ minutes
		mock.MicInUseVal = false
		mock.CamInUseVal = false
		mock.IdleSecondsVal = 20
		snapEnd, _ := mock.Snapshot(ctx)
		state, rec := sm.Step(snapEnd)
		fmt.Printf("2. Meeting ended: State = %s, Recommended Intervention = %s\n", state, rec)
		interv, ok := intr.EvaluateIntervention(rec, snapEnd)
		if ok {
			fmt.Printf("3. Interrupter staged: %s [%s] -> %s\n", interv.Title, interv.Delivery, interv.Message)
		}

	case "typing-burst":
		mock.IdleSecondsVal = 20 // Natural transition pause
		mock.KeystrokeCadence = 75
		// High fatigue
		for i := 0; i < 80; i++ {
			snap, _ := mock.Snapshot(ctx)
			fm.Update(snap, types.StateActive, 60.0)
		}
		snap, _ := mock.Snapshot(ctx)
		sm.Step(&types.ContextSnapshot{IdleSeconds: 2})
		state, rec := sm.Step(snap)
		fmt.Printf("1. High fatigue active state: Score = %.1f, State = %s, Rec = %s\n", fm.Score(), state, rec)
		interv, ok := intr.EvaluateIntervention(rec, snap)
		if ok {
			fmt.Printf("2. Interrupter staged: %s [%s] -> %s\n", interv.Title, interv.Delivery, interv.ActionPrompt)
		}

	case "dnd-suppression":
		mock.DisplayModeVal = types.DisplayPresentationMode
		mock.IdleSecondsVal = 20
		snap, _ := mock.Snapshot(ctx)
		interv, ok := intr.EvaluateIntervention(types.InterventionMicroPause, snap)
		if !ok && interv == nil {
			fmt.Println("✅ Success: Notification properly suppressed in Presentation/Fullscreen Mode.")
		}

	case "rapid-dismissal":
		fmt.Println("Simulating 3 rapid dismissals (< 2.0s)...")
		fm.RecordDismissal(0.8)
		fm.RecordDismissal(1.1)
		fm.RecordDismissal(0.5)
		snap, _ := mock.Snapshot(ctx)
		interv, _ := intr.EvaluateIntervention(types.InterventionMicroPause, snap)
		fmt.Printf("Resulting delivery mode: %s (Notification Blindness mitigation active: %v)\n", interv.Delivery, fm.IsAmbientModeOnly())
	}
}
