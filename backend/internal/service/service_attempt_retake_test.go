package service

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

// fakeAttemptRepo is an in-memory AttemptRepo for retake-limit tests.
type fakeAttemptRepo struct {
	inProgress *model.ExamAttempt
	submitted  []model.ExamAttempt
	created    int
}

func (f *fakeAttemptRepo) CreateAttempt(_ context.Context, a *model.ExamAttempt) error {
	f.created++
	a.ID = uint(100 + f.created)
	return nil
}

func (f *fakeAttemptRepo) FindAttemptByID(_ context.Context, id uint) (*model.ExamAttempt, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeAttemptRepo) UpdateAttempt(_ context.Context, a *model.ExamAttempt) error { return nil }

func (f *fakeAttemptRepo) FindInProgressAttempt(_ context.Context, examID, studentID uint) (*model.ExamAttempt, error) {
	if f.inProgress != nil {
		return f.inProgress, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeAttemptRepo) CountSubmittedAttempts(_ context.Context, examID, studentID uint) (int64, error) {
	return int64(len(f.submitted)), nil
}

func (f *fakeAttemptRepo) FindLatestSubmittedAttempt(_ context.Context, examID, studentID uint) (*model.ExamAttempt, error) {
	if len(f.submitted) == 0 {
		return nil, repository.ErrNotFound
	}
	return &f.submitted[len(f.submitted)-1], nil
}

func (f *fakeAttemptRepo) ListAttemptsByStudent(_ context.Context, studentID, examID uint, page, pageSize int) ([]model.ExamAttempt, int64, error) {
	return nil, 0, nil
}

func (f *fakeAttemptRepo) ListAttemptsByExam(_ context.Context, examID uint) ([]model.ExamAttempt, error) {
	return nil, nil
}

// fakeExamRepo returns a single published exam with one question.
type fakeExamRepo struct {
	exam *model.Exam
}

func (f *fakeExamRepo) CreateExam(_ context.Context, exam *model.Exam) error { return nil }
func (f *fakeExamRepo) FindExamByID(_ context.Context, id uint) (*model.Exam, error) {
	return f.exam, nil
}
func (f *fakeExamRepo) UpdateExam(_ context.Context, exam *model.Exam) error { return nil }
func (f *fakeExamRepo) DeleteExam(_ context.Context, id uint) error          { return nil }
func (f *fakeExamRepo) ListExams(_ context.Context, filter repository.ExamFilter, page, pageSize int) ([]model.Exam, int64, error) {
	return nil, 0, nil
}
func (f *fakeExamRepo) ReplaceExamQuestions(_ context.Context, examID uint, items []model.ExamQuestion) error {
	return nil
}
func (f *fakeExamRepo) ListExamQuestions(_ context.Context, examID uint) ([]model.ExamQuestion, error) {
	return []model.ExamQuestion{{ID: 1, ExamID: examID, QuestionID: 1, Score: 5}}, nil
}
func (f *fakeExamRepo) CountExamQuestions(_ context.Context, examID uint) (int64, error) {
	return 1, nil
}

type fakeQuestionRepo struct{}

func (fakeQuestionRepo) CreateQuestion(_ context.Context, q *model.Question) error { return nil }
func (fakeQuestionRepo) CreateQuestionsBatch(_ context.Context, questions []model.Question) error {
	return nil
}
func (fakeQuestionRepo) FindQuestionByID(_ context.Context, id uint) (*model.Question, error) {
	return nil, repository.ErrNotFound
}
func (fakeQuestionRepo) UpdateQuestion(_ context.Context, q *model.Question) error { return nil }
func (fakeQuestionRepo) DeleteQuestion(_ context.Context, id uint) error           { return nil }
func (fakeQuestionRepo) ListQuestions(_ context.Context, filter repository.QuestionFilter, page, pageSize int) ([]model.Question, int64, error) {
	return nil, 0, nil
}
func (fakeQuestionRepo) ListQuestionsByTypeDifficulty(_ context.Context, qtype, difficulty string) ([]model.Question, error) {
	return nil, nil
}
func (fakeQuestionRepo) FindQuestionsByIDs(_ context.Context, ids []uint) (map[uint]model.Question, error) {
	return map[uint]model.Question{
		1: {ID: 1, Type: constants.QuestionSingle, Content: "q", Options: `[{"key":"A","text":"a"}]`, Answer: `"A"`},
	}, nil
}

type fakeAnswerRepo struct{}

func (fakeAnswerRepo) SaveAnswer(_ context.Context, answer *model.Answer) error { return nil }
func (fakeAnswerRepo) ListAnswersByAttempt(_ context.Context, attemptID uint) ([]model.Answer, error) {
	return nil, nil
}

type fakeWrongRepo struct{}

func (fakeWrongRepo) UpsertWrongQuestion(_ context.Context, w *model.WrongQuestion) error {
	return nil
}
func (fakeWrongRepo) ListWrongQuestions(_ context.Context, studentID uint, knowledgePoint string, page, pageSize int) ([]model.WrongQuestion, int64, error) {
	return nil, 0, nil
}
func (fakeWrongRepo) DeleteWrongQuestion(_ context.Context, id, studentID uint) error {
	return nil
}
func (fakeWrongRepo) MarkWrongQuestionResolved(_ context.Context, id, studentID uint) error {
	return nil
}

func newAttemptService(exam *model.Exam, attempts *fakeAttemptRepo) *AttemptService {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewAttemptService(&fakeExamRepo{exam: exam}, fakeQuestionRepo{}, attempts, fakeAnswerRepo{}, fakeWrongRepo{}, logger)
}

func submittedAttempt(minutesAgo float64) model.ExamAttempt {
	t := time.Now().Add(-time.Duration(minutesAgo * float64(time.Minute)))
	return model.ExamAttempt{ID: 1, ExamID: 1, StudentID: 7, Status: constants.AttemptSubmitted, SubmittedAt: &t}
}

func TestStartAttemptRetakeLimits(t *testing.T) {
	baseExam := func() *model.Exam {
		return &model.Exam{ID: 1, Title: "期末", Status: constants.ExamPublished, DurationMinutes: 60, MaxAttempts: 1, RetakeWaitMinutes: 0}
	}

	t.Run("first attempt allowed", func(t *testing.T) {
		attempts := &fakeAttemptRepo{}
		svc := newAttemptService(baseExam(), attempts)
		if _, err := svc.Start(context.Background(), 7, 1); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if attempts.created != 1 {
			t.Fatalf("expected 1 created attempt, got %d", attempts.created)
		}
	})

	t.Run("max attempts reached", func(t *testing.T) {
		attempts := &fakeAttemptRepo{submitted: []model.ExamAttempt{submittedAttempt(120)}}
		svc := newAttemptService(baseExam(), attempts)
		_, err := svc.Start(context.Background(), 7, 1)
		if err == nil || !strings.Contains(err.Error(), "已达到作答次数上限") {
			t.Fatalf("Start() error = %v, want limit reached", err)
		}
		if attempts.created != 0 {
			t.Fatalf("no new attempt should be created, got %d", attempts.created)
		}
	})

	t.Run("in progress attempt resumes without consuming quota", func(t *testing.T) {
		exam := baseExam()
		attempts := &fakeAttemptRepo{
			inProgress: &model.ExamAttempt{ID: 9, ExamID: 1, StudentID: 7, Status: constants.AttemptInProgress, StartedAt: time.Now(), Deadline: time.Now().Add(time.Hour)},
			submitted:  []model.ExamAttempt{submittedAttempt(120)},
		}
		svc := newAttemptService(exam, attempts)
		resp, err := svc.Start(context.Background(), 7, 1)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if resp.AttemptID != 9 {
			t.Fatalf("expected resumed attempt 9, got %d", resp.AttemptID)
		}
		if attempts.created != 0 {
			t.Fatalf("resume must not create a new attempt, got %d", attempts.created)
		}
	})

	t.Run("within wait window rejected with next start time", func(t *testing.T) {
		exam := baseExam()
		exam.MaxAttempts = 3
		exam.RetakeWaitMinutes = 30
		attempts := &fakeAttemptRepo{submitted: []model.ExamAttempt{submittedAttempt(10)}}
		svc := newAttemptService(exam, attempts)
		_, err := svc.Start(context.Background(), 7, 1)
		if err == nil || !strings.Contains(err.Error(), "再次开始") {
			t.Fatalf("Start() error = %v, want wait window rejection", err)
		}
		if attempts.created != 0 {
			t.Fatalf("no new attempt should be created, got %d", attempts.created)
		}
	})

	t.Run("wait window elapsed allowed", func(t *testing.T) {
		exam := baseExam()
		exam.MaxAttempts = 3
		exam.RetakeWaitMinutes = 30
		attempts := &fakeAttemptRepo{submitted: []model.ExamAttempt{submittedAttempt(45)}}
		svc := newAttemptService(exam, attempts)
		if _, err := svc.Start(context.Background(), 7, 1); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if attempts.created != 1 {
			t.Fatalf("expected 1 created attempt, got %d", attempts.created)
		}
	})
}
