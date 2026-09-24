package service

import (
	"fmt"
	"time"

	"github.com/gbexam/online-exam/internal/model"
)

// Attempt states reported to students on the exam list.
const (
	AttemptStateAvailable  = "available"
	AttemptStateInProgress = "in_progress"
	AttemptStateWaiting    = "waiting"
	AttemptStateReached    = "reached_limit"
)

// AttemptPolicy captures the per-student counters used to decide whether a new
// attempt may be started. An unfinished attempt never consumes a slot: it can
// always be resumed and is excluded from UsedAttempts.
type AttemptPolicy struct {
	HasInProgress     bool
	UsedAttempts      int
	LatestSubmittedAt *time.Time
}

// NormalizeExamPolicy fills defaults for exam limits so legacy rows created
// before the attempt-limit feature behave as "1 attempt, no waiting".
func NormalizeExamPolicy(exam *model.Exam) {
	if exam.MaxAttempts <= 0 {
		exam.MaxAttempts = 1
	}
	if exam.WaitMinutes < 0 {
		exam.WaitMinutes = 0
	}
}

// EvaluateAttemptPolicy decides whether the student may start a new attempt.
// state is always populated; canStart is true only when a brand-new attempt is
// allowed; nextStartAt is set while a cooldown is in effect.
func EvaluateAttemptPolicy(exam *model.Exam, p AttemptPolicy, now time.Time) (state string, canStart bool, nextStartAt *time.Time) {
	NormalizeExamPolicy(exam)
	switch {
	case p.HasInProgress:
		return AttemptStateInProgress, false, nil
	case p.UsedAttempts >= exam.MaxAttempts:
		return AttemptStateReached, false, nil
	case exam.WaitMinutes > 0 && p.UsedAttempts > 0 && p.LatestSubmittedAt != nil &&
		now.Before(p.LatestSubmittedAt.Add(time.Duration(exam.WaitMinutes)*time.Minute)):
		t := p.LatestSubmittedAt.Add(time.Duration(exam.WaitMinutes) * time.Minute)
		return AttemptStateWaiting, false, &t
	default:
		return AttemptStateAvailable, true, nil
	}
}

// checkStartPolicy returns a validation error carrying a Chinese message when a
// new attempt is not allowed. It must only be called after ruling out an
// existing in-progress attempt.
func checkStartPolicy(exam *model.Exam, p AttemptPolicy, now time.Time) error {
	state, canStart, nextStartAt := EvaluateAttemptPolicy(exam, p, now)
	if canStart || state == AttemptStateInProgress {
		return nil
	}
	if state == AttemptStateReached {
		return fmt.Errorf("%w: 已达到作答次数上限（%d 次）", ErrValidation, exam.MaxAttempts)
	}
	return fmt.Errorf("%w: 补考等待中，可再次开始时间：%s", ErrValidation, nextStartAt.Format("2006-01-02 15:04:05"))
}
