package service

import (
	"testing"
	"time"

	"github.com/gbexam/online-exam/internal/model"
)

func TestNormalizeExamPolicy(t *testing.T) {
	exam := &model.Exam{MaxAttempts: 0, WaitMinutes: -1}
	NormalizeExamPolicy(exam)
	if exam.MaxAttempts != 1 {
		t.Fatalf("MaxAttempts = %d, want 1", exam.MaxAttempts)
	}
	if exam.WaitMinutes != 0 {
		t.Fatalf("WaitMinutes = %d, want 0", exam.WaitMinutes)
	}
}

func TestEvaluateAttemptPolicy(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.Local)
	submitted := now.Add(-10 * time.Minute)
	tests := []struct {
		name        string
		maxAttempts int
		waitMinutes int
		policy      AttemptPolicy
		wantState   string
		wantStart   bool
		wantWait    bool
	}{
		{
			name:        "fresh student can start once",
			maxAttempts: 1,
			wantState:   AttemptStateAvailable,
			wantStart:   true,
		},
		{
			name:        "in progress never consumes a slot and cannot start new",
			maxAttempts: 1,
			policy:      AttemptPolicy{HasInProgress: true},
			wantState:   AttemptStateInProgress,
		},
		{
			name:        "single attempt exhausted",
			maxAttempts: 1,
			policy:      AttemptPolicy{UsedAttempts: 1, LatestSubmittedAt: &submitted},
			wantState:   AttemptStateReached,
		},
		{
			name:        "second attempt allowed with zero wait",
			maxAttempts: 2,
			waitMinutes: 0,
			policy:      AttemptPolicy{UsedAttempts: 1, LatestSubmittedAt: &submitted},
			wantState:   AttemptStateAvailable,
			wantStart:   true,
		},
		{
			name:        "second attempt blocked during waiting period",
			maxAttempts: 3,
			waitMinutes: 30,
			policy:      AttemptPolicy{UsedAttempts: 1, LatestSubmittedAt: &submitted},
			wantState:   AttemptStateWaiting,
			wantWait:    true,
		},
		{
			name:        "second attempt allowed after waiting period",
			maxAttempts: 3,
			waitMinutes: 5,
			policy:      AttemptPolicy{UsedAttempts: 1, LatestSubmittedAt: &submitted},
			wantState:   AttemptStateAvailable,
			wantStart:   true,
		},
		{
			name:        "unfinished paper does not consume attempts",
			maxAttempts: 1,
			policy:      AttemptPolicy{HasInProgress: true, UsedAttempts: 0},
			wantState:   AttemptStateInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exam := &model.Exam{MaxAttempts: tt.maxAttempts, WaitMinutes: tt.waitMinutes}
			state, canStart, nextStartAt := EvaluateAttemptPolicy(exam, tt.policy, now)
			if state != tt.wantState {
				t.Fatalf("state = %q, want %q", state, tt.wantState)
			}
			if canStart != tt.wantStart {
				t.Fatalf("canStart = %v, want %v", canStart, tt.wantStart)
			}
			if tt.wantWait {
				if nextStartAt == nil {
					t.Fatal("nextStartAt = nil, want cooldown end time")
				}
				want := submitted.Add(time.Duration(tt.waitMinutes) * time.Minute)
				if !nextStartAt.Equal(want) {
					t.Fatalf("nextStartAt = %v, want %v", nextStartAt, want)
				}
			}
		})
	}
}
