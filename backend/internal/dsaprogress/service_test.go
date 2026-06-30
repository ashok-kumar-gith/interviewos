package dsaprogress

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo is an in-memory Repository for service unit tests.
type fakeRepo struct {
	exists bool
	stored *Progress
}

func (f *fakeRepo) ProblemExists(_ context.Context, _ uuid.UUID) (bool, error) { return f.exists, nil }
func (f *fakeRepo) Get(_ context.Context, _, _ uuid.UUID) (*Progress, error)   { return f.stored, nil }
func (f *fakeRepo) List(_ context.Context, _ uuid.UUID) ([]Progress, error) {
	if f.stored == nil {
		return nil, nil
	}
	return []Progress{*f.stored}, nil
}
func (f *fakeRepo) Delete(_ context.Context, _, _ uuid.UUID) error { f.stored = nil; return nil }
func (f *fakeRepo) Upsert(_ context.Context, userID, problemID uuid.UUID, in Input, now time.Time) (*Progress, error) {
	p := &Progress{
		UserID: userID, ProblemID: problemID, Solved: in.Solved,
		Confidence: in.Confidence, TimeSpentMinutes: in.TimeSpentMinutes,
		SolutionCode: in.SolutionCode, SolutionLanguage: in.SolutionLanguage, SolutionNotes: in.SolutionNotes,
	}
	if in.Solved {
		p.Status = "completed"
		p.SolvedAt = &now
	} else {
		p.Status = "in_progress"
	}
	f.stored = p
	return p, nil
}

func ptr[T any](v T) *T { return &v }

func TestSave_ProblemMustExist(t *testing.T) {
	svc := NewService(&fakeRepo{exists: false})
	if _, err := svc.Save(context.Background(), uuid.New(), uuid.New(), Input{Solved: true}); err != ErrProblemNotFound {
		t.Fatalf("want ErrProblemNotFound, got %v", err)
	}
}

func TestSave_ValidatesConfidence(t *testing.T) {
	svc := NewService(&fakeRepo{exists: true})
	bad := int16(9)
	if _, err := svc.Save(context.Background(), uuid.New(), uuid.New(), Input{Confidence: &bad}); err != ErrValidation {
		t.Fatalf("want ErrValidation for confidence 9, got %v", err)
	}
}

func TestSave_SolvedStoresSolution(t *testing.T) {
	repo := &fakeRepo{exists: true}
	svc := NewService(repo)
	got, err := svc.Save(context.Background(), uuid.New(), uuid.New(), Input{
		Solved: true, SolutionCode: ptr("print(1)"), SolutionLanguage: ptr("python"),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.Solved || got.Status != "completed" || got.SolvedAt == nil {
		t.Fatalf("expected solved/completed with solved_at, got %+v", got)
	}
	if got.SolutionCode == nil || *got.SolutionCode != "print(1)" {
		t.Fatalf("solution code not stored: %+v", got.SolutionCode)
	}
}

func TestSave_BlankSolutionTrimmedToNil(t *testing.T) {
	svc := NewService(&fakeRepo{exists: true})
	got, err := svc.Save(context.Background(), uuid.New(), uuid.New(), Input{
		Solved: false, SolutionCode: ptr("   "),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.SolutionCode != nil {
		t.Fatalf("blank code should trim to nil, got %v", *got.SolutionCode)
	}
}

func TestDelete(t *testing.T) {
	repo := &fakeRepo{exists: true, stored: &Progress{Solved: true}}
	svc := NewService(repo)
	if err := svc.Delete(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if repo.stored != nil {
		t.Fatal("expected stored progress cleared")
	}
}
