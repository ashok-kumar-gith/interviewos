package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeRepo is an in-memory Repository for service unit tests. It enforces
// ownership scoping the same way the gorm implementation does.
type fakeRepo struct {
	mocks    map[uuid.UUID]*Interview
	findings map[uuid.UUID]*Finding
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		mocks:    map[uuid.UUID]*Interview{},
		findings: map[uuid.UUID]*Finding{},
	}
}

func (f *fakeRepo) Create(_ context.Context, m *Interview) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	cp := *m
	f.mocks[m.ID] = &cp
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, userID, id uuid.UUID) (*Interview, error) {
	m, ok := f.mocks[id]
	if !ok || m.UserID != userID {
		return nil, ErrMockNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeRepo) GetByIDWithFindings(_ context.Context, userID, id uuid.UUID) (*Interview, error) {
	m, err := f.GetByID(context.Background(), userID, id)
	if err != nil {
		return nil, err
	}
	for _, fi := range f.findings {
		if fi.MockInterviewID == id {
			m.Findings = append(m.Findings, *fi)
		}
	}
	return m, nil
}

func (f *fakeRepo) List(_ context.Context, userID uuid.UUID, filter ListFilter) ([]Interview, int64, error) {
	var out []Interview
	for _, m := range f.mocks {
		if m.UserID != userID {
			continue
		}
		if filter.Type != nil && m.Type != *filter.Type {
			continue
		}
		out = append(out, *m)
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) Update(_ context.Context, m *Interview) error {
	existing, ok := f.mocks[m.ID]
	if !ok || existing.UserID != m.UserID {
		return ErrMockNotFound
	}
	cp := *m
	f.mocks[m.ID] = &cp
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, userID, id uuid.UUID) error {
	m, ok := f.mocks[id]
	if !ok || m.UserID != userID {
		return ErrMockNotFound
	}
	delete(f.mocks, id)
	return nil
}

func (f *fakeRepo) AddFinding(_ context.Context, fi *Finding) error {
	if fi.ID == uuid.Nil {
		fi.ID = uuid.New()
	}
	cp := *fi
	f.findings[fi.ID] = &cp
	return nil
}

func (f *fakeRepo) UpdateFinding(_ context.Context, fi *Finding) error {
	cp := *fi
	f.findings[fi.ID] = &cp
	return nil
}

func (f *fakeRepo) ListFindings(_ context.Context, userID uuid.UUID) ([]Finding, error) {
	var out []Finding
	for _, fi := range f.findings {
		if fi.UserID == userID {
			out = append(out, *fi)
		}
	}
	return out, nil
}

func ptrInt(n int) *int       { return &n }
func ptrF(f float64) *float64 { return &f }

func TestService_Create_Validation(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	ctx := context.Background()
	uid := uuid.New()

	// Missing type.
	if _, err := svc.Create(ctx, uid, CreateInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing type, got %v", err)
	}

	// Invalid type.
	if _, err := svc.Create(ctx, uid, CreateInput{Type: "nope"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid type, got %v", err)
	}

	// Out-of-range score.
	if _, err := svc.Create(ctx, uid, CreateInput{Type: TypeCoding, OverallScore: ptrF(150)}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for overall_score>100, got %v", err)
	}

	// Bad duration.
	if _, err := svc.Create(ctx, uid, CreateInput{Type: TypeCoding, DurationMinutes: ptrInt(-5)}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for negative duration, got %v", err)
	}
}

func TestService_Create_DefaultsOutcome(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	m, err := svc.Create(context.Background(), uuid.New(), CreateInput{Type: TypeSystemDesign})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Outcome != OutcomeNotRated {
		t.Fatalf("expected default outcome not_rated, got %q", m.Outcome)
	}
	if m.ID == uuid.Nil {
		t.Fatal("expected an id to be assigned")
	}
}

func TestService_Get_Ownership(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()

	m, err := svc.Create(ctx, owner, CreateInput{Type: TypeCoding})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Get(ctx, owner, m.ID); err != nil {
		t.Fatalf("owner Get failed: %v", err)
	}
	if _, err := svc.Get(ctx, other, m.ID); !errors.Is(err, ErrMockNotFound) {
		t.Fatalf("expected not-found for other user, got %v", err)
	}
}

func TestService_Update_OwnershipAndValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()

	m, _ := svc.Create(ctx, owner, CreateInput{Type: TypeCoding})

	// Non-owner update => not found.
	if _, err := svc.Update(ctx, other, m.ID, UpdateInput{Type: TypeCoding}); !errors.Is(err, ErrMockNotFound) {
		t.Fatalf("expected not-found updating as non-owner, got %v", err)
	}

	// Invalid update payload => validation.
	if _, err := svc.Update(ctx, owner, m.ID, UpdateInput{Type: "bogus"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	// Valid update.
	updated, err := svc.Update(ctx, owner, m.ID, UpdateInput{Type: TypeLLD, Outcome: OutcomeHire, OverallScore: ptrF(80)})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Type != TypeLLD || updated.Outcome != OutcomeHire {
		t.Fatalf("update did not apply fields: %+v", updated)
	}
}

func TestService_Delete_Ownership(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()
	m, _ := svc.Create(ctx, owner, CreateInput{Type: TypeCoding})

	if err := svc.Delete(ctx, other, m.ID); !errors.Is(err, ErrMockNotFound) {
		t.Fatalf("expected not-found deleting as non-owner, got %v", err)
	}
	if err := svc.Delete(ctx, owner, m.ID); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if err := svc.Delete(ctx, owner, m.ID); !errors.Is(err, ErrMockNotFound) {
		t.Fatalf("expected not-found on second delete, got %v", err)
	}
}

func TestService_AddFinding_OwnershipAndValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()
	m, _ := svc.Create(ctx, owner, CreateInput{Type: TypeCoding})

	// Adding to a mock the user does not own => not found (and not leaked).
	_, err := svc.AddFinding(ctx, other, m.ID, FindingInput{Severity: SeverityMajor, Category: "x", Detail: "y"})
	if !errors.Is(err, ErrMockNotFound) {
		t.Fatalf("expected not-found adding finding as non-owner, got %v", err)
	}

	// Missing required fields => validation.
	_, err = svc.AddFinding(ctx, owner, m.ID, FindingInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	// Invalid severity => validation.
	_, err = svc.AddFinding(ctx, owner, m.ID, FindingInput{Severity: "huge", Category: "c", Detail: "d"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for bad severity, got %v", err)
	}

	// Valid finding.
	f, err := svc.AddFinding(ctx, owner, m.ID, FindingInput{Severity: SeverityMajor, Category: "Correctness", Detail: "off-by-one"})
	if err != nil {
		t.Fatalf("add finding: %v", err)
	}
	if f.MockInterviewID != m.ID || f.UserID != owner {
		t.Fatalf("finding not scoped correctly: %+v", f)
	}
}

func TestService_Weaknesses_Aggregation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()
	m, _ := svc.Create(ctx, owner, CreateInput{Type: TypeCoding})

	dsa := PillarDSA
	// communication: minor(2) + major(4) = 6, count 2
	// correctness: blocker(8) = 8, count 1
	mustFinding(t, svc, ctx, owner, m.ID, FindingInput{Severity: SeverityMinor, Category: "communication", Detail: "rambling"})
	mustFinding(t, svc, ctx, owner, m.ID, FindingInput{Severity: SeverityMajor, Category: "Communication", Detail: "unclear", PillarType: &dsa})
	mustFinding(t, svc, ctx, owner, m.ID, FindingInput{Severity: SeverityBlocker, Category: "correctness", Detail: "wrong answer", PillarType: &dsa})

	// Another user's finding must not leak into the owner's summary.
	m2, _ := svc.Create(ctx, other, CreateInput{Type: TypeCoding})
	mustFinding(t, svc, ctx, other, m2.ID, FindingInput{Severity: SeverityBlocker, Category: "communication", Detail: "noise"})

	sum, err := svc.Weaknesses(ctx, owner)
	if err != nil {
		t.Fatalf("weaknesses: %v", err)
	}
	if sum.TotalFindings != 3 {
		t.Fatalf("expected 3 findings for owner, got %d", sum.TotalFindings)
	}
	if sum.GeneratedBy != "deterministic" {
		t.Fatalf("expected deterministic detector, got %q", sum.GeneratedBy)
	}
	if len(sum.Items) != 2 {
		t.Fatalf("expected 2 weakness areas, got %d: %+v", len(sum.Items), sum.Items)
	}
	// correctness (score 8) ranks above communication (score 6).
	if sum.Items[0].Area != "correctness" {
		t.Fatalf("expected correctness ranked first, got %q (items: %+v)", sum.Items[0].Area, sum.Items)
	}
	if sum.Items[0].Score != 8 || sum.Items[0].MaxSeverity != SeverityBlocker {
		t.Fatalf("correctness agg wrong: %+v", sum.Items[0])
	}
	comm := sum.Items[1]
	if comm.Area != "communication" || comm.Count != 2 || comm.Score != 6 {
		t.Fatalf("communication agg wrong: %+v", comm)
	}
	if comm.MaxSeverity != SeverityMajor {
		t.Fatalf("communication max severity wrong: %+v", comm)
	}
	// Case-insensitive grouping: "Communication" merged with "communication".
	if comm.SeverityCounts[SeverityMinor] != 1 || comm.SeverityCounts[SeverityMajor] != 1 {
		t.Fatalf("communication severity counts wrong: %+v", comm.SeverityCounts)
	}
	if comm.Pillar == nil || *comm.Pillar != PillarDSA {
		t.Fatalf("expected dominant pillar dsa for communication, got %v", comm.Pillar)
	}
}

func TestService_Weaknesses_Empty(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	sum, err := svc.Weaknesses(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("weaknesses: %v", err)
	}
	if sum.TotalFindings != 0 || len(sum.Items) != 0 {
		t.Fatalf("expected empty summary, got %+v", sum)
	}
}

func TestDeterministicDetector_StableOrder(t *testing.T) {
	det := NewDeterministicWeaknessDetector()
	// Two areas with identical score+count must break ties by area name asc.
	findings := []Finding{
		{Category: "zebra", Severity: SeverityMajor},
		{Category: "alpha", Severity: SeverityMajor},
	}
	sum, _ := det.Detect(context.Background(), findings)
	if len(sum.Items) != 2 || sum.Items[0].Area != "alpha" {
		t.Fatalf("expected alpha first on tie, got %+v", sum.Items)
	}
}

func mustFinding(t *testing.T, svc *Service, ctx context.Context, uid, mockID uuid.UUID, in FindingInput) {
	t.Helper()
	if _, err := svc.AddFinding(ctx, uid, mockID, in); err != nil {
		t.Fatalf("add finding: %v", err)
	}
}
