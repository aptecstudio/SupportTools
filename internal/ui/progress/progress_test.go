// SPDX-License-Identifier: GPL-3.0-or-later

package progress

import (
	"context"
	"testing"
	"time"
)

func TestFraction(t *testing.T) {
	tests := []struct {
		done, total int64
		want        float64
	}{
		{0, 0, 0},
		{5, 0, 0},
		{-1, 10, 0},
		{0, 10, 0},
		{5, 10, 0.5},
		{10, 10, 1},
		{15, 10, 1},
	}
	for _, tt := range tests {
		if got := fraction(tt.done, tt.total); got != tt.want {
			t.Errorf("fraction(%d, %d) = %v, want %v", tt.done, tt.total, got, tt.want)
		}
	}
}

func TestEstimateRemaining(t *testing.T) {
	if _, ok := estimateRemaining(time.Second, 0.5); ok {
		t.Error("stima disponibile troppo presto")
	}
	if _, ok := estimateRemaining(time.Minute, 0.001); ok {
		t.Error("stima disponibile con avanzamento troppo basso")
	}
	got, ok := estimateRemaining(10*time.Second, 0.25)
	if !ok || got != 30*time.Second {
		t.Errorf("estimateRemaining(10s, 0.25) = %v, %v; want 30s, true", got, ok)
	}
	if got, ok := estimateRemaining(time.Second, 1); !ok || got != 0 {
		t.Errorf("estimateRemaining(1s, 1) = %v, %v; want 0, true", got, ok)
	}
}

func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{2 * time.Second, "pochi secondi"},
		{12 * time.Second, "15 s"},
		{58 * time.Second, "1 min"},
		{90 * time.Second, "1 min 30 s"},
		{2*time.Minute + 2*time.Second, "2 min"},
		{time.Hour + 20*time.Minute, "1 h 20 min"},
		{2*time.Hour + 10*time.Second, "2 h"},
	}
	for _, tt := range tests {
		if got := formatRemaining(tt.d); got != tt.want {
			t.Errorf("formatRemaining(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestReporterSteps(t *testing.T) {
	r := newReporter(context.Background())
	r.Step("Prima")
	r.SetStatus("file.txt")
	s := r.snapshot(0)
	if s.current != "Prima" || s.status != "file.txt" {
		t.Fatalf("current = %q, status = %q", s.current, s.status)
	}

	r.Step("Seconda")
	s = r.snapshot(0)
	if s.status != "" {
		t.Errorf("status non azzerato dal nuovo Step: %q", s.status)
	}
	want := []Step{{"Prima", StepDone}, {"Seconda", StepRunning}}
	if len(s.steps) != len(want) {
		t.Fatalf("steps = %v, want %v", s.steps, want)
	}
	for i := range want {
		if s.steps[i] != want[i] {
			t.Errorf("steps[%d] = %v, want %v", i, s.steps[i], want[i])
		}
	}

	// Senza modifiche l'elenco non viene ricopiato.
	if again := r.snapshot(s.stepsRev); again.steps != nil {
		t.Error("elenco ricopiato senza modifiche")
	}

	r.finish(StepFailed)
	s = r.snapshot(s.stepsRev)
	if s.current != "" || s.steps[1].State != StepFailed {
		t.Errorf("dopo finish: current = %q, ultimo stato = %v", s.current, s.steps[1].State)
	}
}
