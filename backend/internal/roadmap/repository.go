package roadmap

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository abstracts persistence for the roadmap domain so the service is
// unit-testable against a fake. The GORM implementation is gormRepository.
type Repository interface {
	// GetActive returns the user's active roadmap WITHOUT its week graph
	// (lightweight), or ErrNoActiveRoadmap.
	GetActive(ctx context.Context, userID uuid.UUID) (*Roadmap, error)
	// GetActiveWithWeeks returns the active roadmap with its weeks (no days), or
	// ErrNoActiveRoadmap.
	GetActiveWithWeeks(ctx context.Context, userID uuid.UUID) (*Roadmap, error)
	// GetWeek returns a single week (with days + tasks) for a roadmap owned by
	// the user, or ErrNotFound.
	GetWeek(ctx context.Context, userID, roadmapID uuid.UUID, weekNumber int) (*RoadmapWeek, error)
	// GetPlanDay returns the user's plan-day for a date (with tasks), or ErrNotFound.
	GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDay, error)
	// CreateGraph persists a full roadmap graph (roadmap → weeks → days → tasks)
	// in one transaction. If replaceActive is true, any existing active roadmap
	// for the user is archived (is_active=false) first so the partial unique
	// index is honored.
	CreateGraph(ctx context.Context, rm *Roadmap, replaceActive bool) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetActive(ctx context.Context, userID uuid.UUID) (*Roadmap, error) {
	var rm Roadmap
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = true", userID).
		First(&rm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoActiveRoadmap
	}
	if err != nil {
		return nil, err
	}
	return &rm, nil
}

func (r *gormRepository) GetActiveWithWeeks(ctx context.Context, userID uuid.UUID) (*Roadmap, error) {
	rm, err := r.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	var weeks []RoadmapWeek
	if err := r.db.WithContext(ctx).
		Where("roadmap_id = ?", rm.ID).
		Order("week_number ASC").Find(&weeks).Error; err != nil {
		return nil, err
	}
	rm.Weeks = weeks
	return rm, nil
}

func (r *gormRepository) GetWeek(ctx context.Context, userID, roadmapID uuid.UUID, weekNumber int) (*RoadmapWeek, error) {
	// Ownership check: the roadmap must belong to the user.
	var owned int64
	if err := r.db.WithContext(ctx).Model(&Roadmap{}).
		Where("id = ? AND user_id = ?", roadmapID, userID).
		Count(&owned).Error; err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, ErrNotFound
	}

	var week RoadmapWeek
	err := r.db.WithContext(ctx).
		Where("roadmap_id = ? AND week_number = ?", roadmapID, weekNumber).
		First(&week).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var days []PlanDay
	if err := r.db.WithContext(ctx).
		Where("roadmap_week_id = ?", week.ID).
		Order("date ASC").Find(&days).Error; err != nil {
		return nil, err
	}
	if err := r.loadTasksForDays(ctx, days); err != nil {
		return nil, err
	}
	week.Days = days
	return &week, nil
}

func (r *gormRepository) GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDay, error) {
	var day PlanDay
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND date = ?", userID, date.Format("2006-01-02")).
		First(&day).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadTasksForDays(ctx, []PlanDay{day}); err != nil {
		return nil, err
	}
	// loadTasksForDays mutates copies in the slice; reload tasks onto day directly.
	var tasks []PlanTask
	if err := r.db.WithContext(ctx).
		Where("plan_day_id = ?", day.ID).
		Order("sort_order ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	day.Tasks = tasks
	return &day, nil
}

// loadTasksForDays attaches tasks to each day in the slice with a single query.
func (r *gormRepository) loadTasksForDays(ctx context.Context, days []PlanDay) error {
	if len(days) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(days))
	for i := range days {
		ids[i] = days[i].ID
	}
	var tasks []PlanTask
	if err := r.db.WithContext(ctx).
		Where("plan_day_id IN ?", ids).
		Order("plan_day_id ASC, sort_order ASC").Find(&tasks).Error; err != nil {
		return err
	}
	byDay := make(map[uuid.UUID][]PlanTask, len(days))
	for _, t := range tasks {
		byDay[t.PlanDayID] = append(byDay[t.PlanDayID], t)
	}
	for i := range days {
		days[i].Tasks = byDay[days[i].ID]
	}
	return nil
}

func (r *gormRepository) CreateGraph(ctx context.Context, rm *Roadmap, replaceActive bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if replaceActive {
			// Archive any existing active roadmap to free the partial unique index.
			if err := tx.Model(&Roadmap{}).
				Where("user_id = ? AND is_active = true", rm.UserID).
				Updates(map[string]any{"is_active": false, "status": "archived"}).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(rm).Error; err != nil {
			return err
		}
		for wi := range rm.Weeks {
			w := &rm.Weeks[wi]
			w.RoadmapID = rm.ID
			if err := tx.Create(w).Error; err != nil {
				return err
			}
			for di := range w.Days {
				d := &w.Days[di]
				d.RoadmapWeekID = w.ID
				d.UserID = rm.UserID
				if err := tx.Create(d).Error; err != nil {
					return err
				}
				if len(d.Tasks) == 0 {
					continue
				}
				for ti := range d.Tasks {
					d.Tasks[ti].PlanDayID = d.ID
					d.Tasks[ti].UserID = rm.UserID
				}
				if err := tx.Create(&d.Tasks).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
