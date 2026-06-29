package company

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct {
	profile  *Profile
	exists   bool
	setCalls int
	lastSet  *uuid.UUID
}

func (f *fakeRepo) GetProfile(_ context.Context, _ uuid.UUID) (*Profile, error) {
	if f.profile == nil {
		return nil, ErrProfileNotFound
	}
	return f.profile, nil
}
func (f *fakeRepo) CompanyExists(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.exists, nil
}
func (f *fakeRepo) SetTargetCompany(_ context.Context, _ uuid.UUID, companyID *uuid.UUID, _ time.Time) (*Profile, error) {
	f.setCalls++
	f.lastSet = companyID
	p := *f.profile
	p.TargetCompanyID = companyID
	return &p, nil
}

// TestSetTarget_RejectsUnknownCompany verifies an unknown company yields
// ErrCompanyNotFound and never mutates the profile (FR-CMP-002 validation).
func TestSetTarget_RejectsUnknownCompany(t *testing.T) {
	repo := &fakeRepo{profile: &Profile{ID: uuid.New(), UserID: uuid.New()}, exists: false}
	svc := NewService(ServiceConfig{Repo: repo})
	_, err := svc.SetTarget(context.Background(), repo.profile.UserID, uuid.New())
	if !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
	if repo.setCalls != 0 {
		t.Fatalf("profile must not be mutated for unknown company")
	}
}

// TestSetTarget_PersistsValidCompany verifies a valid company is persisted.
func TestSetTarget_PersistsValidCompany(t *testing.T) {
	uid := uuid.New()
	repo := &fakeRepo{profile: &Profile{ID: uuid.New(), UserID: uid}, exists: true}
	svc := NewService(ServiceConfig{Repo: repo})
	companyID := uuid.New()
	p, err := svc.SetTarget(context.Background(), uid, companyID)
	if err != nil {
		t.Fatal(err)
	}
	if repo.setCalls != 1 || repo.lastSet == nil || *repo.lastSet != companyID {
		t.Fatalf("expected company persisted, calls=%d", repo.setCalls)
	}
	if p.TargetCompanyID == nil || *p.TargetCompanyID != companyID {
		t.Fatalf("returned profile target mismatch")
	}
}
