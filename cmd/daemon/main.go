package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	fmt.Println("================================================================")
	fmt.Println(" 🧠 Context-Aware Workspace Well-Being & Break Controller Daemon")
	fmt.Println(" Local-First | Privacy-Preserving | Cognitive Transition Detection")
	fmt.Println("================================================================")

	dbPath := os.Getenv("WELLBEING_DB_PATH")
	if dbPath == "" {
		dbPath = "wellness_local.db"
	}

	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	fatigueModel := fatigue.NewFatigueModel()
	sm := statemachine.NewStateMachine(fatigueModel)
	intr := interrupter.NewInterrupter(fatigueModel)
	notifier := notification.NewSystemNotifier()
	snsr := sensor.NewNativeSensor()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// State machine evaluation tick: 10 seconds
	smTicker := time.NewTicker(10 * time.Second)
	defer smTicker.Stop()

	// Fatigue score recalculation tick: 30 seconds
	fatigueTicker := time.NewTicker(30 * time.Second)
	defer fatigueTicker.Stop()

	// Persistent metrics logging tick: 60 seconds
	logTicker := time.NewTicker(60 * time.Second)
	defer logTicker.Stop()

	fmt.Println("[DAEMON] Initialized successfully. Monitoring local environment...")

	for {
		select {
		case <-sigChan:
			fmt.Println("\n[DAEMON] Graceful shutdown requested. Saving final session state...")
			_ = store.RecordActiveTime(ctx, int(fatigueModel.ContinuousActiveMinutes()), fatigueModel.Score())
			fmt.Println("[DAEMON] Shutdown complete. Goodbye.")
			return

		case <-fatigueTicker.C:
			snap, err := snsr.Snapshot(ctx)
			if err == nil {
				score := fatigueModel.Update(snap, sm.CurrentState(), 30.0)
				fmt.Printf("[METRICS] Fatigue Score: %.2f/100 | State: %s | Idle: %ds\n", score, sm.CurrentState(), snap.IdleSeconds)
			}

		case <-smTicker.C:
			snap, err := snsr.Snapshot(ctx)
			if err != nil {
				continue
			}

			// Transition state machine
			prevState := sm.CurrentState()
			currState, recommendedIntervention := sm.Step(snap)

			if prevState != currState {
				fmt.Printf("[STATE-CHANGE] Transitioned from %s -> %s\n", prevState, currState)

				// Log completed meetings
				if prevState == types.StateInMeeting && currState == types.StateTransitionWindow {
					meetingMin := int(sm.MeetingDuration().Minutes())
					_ = store.RecordMeeting(ctx, meetingMin)
				}
			}

			// Evaluate intervention
			if recommendedIntervention != types.InterventionNone {
				intervention, ok := intr.EvaluateIntervention(recommendedIntervention, snap)
				if ok && intervention != nil {
					_ = notifier.Dispatch(ctx, intervention)
					_ = store.RecordBreak(ctx, *intervention)
					fatigueModel.RecordPromptDispatched()
				} else {
					fmt.Printf("[INTERRUPTER] Intervention %s suppressed by display mode (%s)\n", recommendedIntervention, snap.DisplayMode)
				}
			}

		case <-logTicker.C:
			// Persist ongoing active time snapshot
			activeMin := int(fatigueModel.ContinuousActiveMinutes())
			if activeMin > 0 {
				_ = store.RecordActiveTime(ctx, 1, fatigueModel.Score())
			}
		}
	}
}
