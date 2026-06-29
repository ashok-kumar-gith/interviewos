package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Generator computes the digest-style notifications a user should see for a
// given day and upserts them idempotently. Re-running for the same user/day
// never creates duplicates: every generated notification carries a stable
// per-day dedup_key enforced by the partial unique index uq_notif_user_dedup.
//
// It depends only on the narrow read ports (PlanReader/RevisionReader/
// StreakReader/ReadinessReader) and the Repository, so it is fully unit-testable
// with fakes and stays decoupled from other modules' internals.
type Generator struct {
	repo      Repository
	plans     PlanReader
	revisions RevisionReader
	streaks   StreakReader
	readiness ReadinessReader
	weekly    WeeklyReader
	now       func() time.Time
}

// GeneratorConfig configures a Generator. All read ports are optional and
// nil-safe: a nil port simply skips the notification(s) that depend on it, so
// the generator degrades gracefully if a data source is unavailable.
type GeneratorConfig struct {
	Repo      Repository
	Plans     PlanReader
	Revisions RevisionReader
	Streaks   StreakReader
	Readiness ReadinessReader
	Weekly    WeeklyReader
	Now       func() time.Time
}

// NewGenerator constructs a Generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Generator{
		repo:      cfg.Repo,
		plans:     cfg.Plans,
		revisions: cfg.Revisions,
		streaks:   cfg.Streaks,
		readiness: cfg.Readiness,
		weekly:    cfg.Weekly,
		now:       nowFn,
	}
}

// readinessMilestoneStep is the readiness gain (in points) that triggers a
// readiness_milestone notification when the latest snapshot crosses a new
// multiple of the step relative to the previous snapshot.
const readinessMilestoneStep = 10.0

// Generate computes and upserts the appropriate notifications for userID as of
// "now" and returns every notification that is current for the day (both
// freshly created and pre-existing ones it refreshed). It is idempotent: a
// second call on the same day returns the same set without inserting duplicates.
func (g *Generator) Generate(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	if userID == uuid.Nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "user_id", Message: "is required"}}}
	}
	now := g.now().UTC()
	today := truncDay(now)
	dayStr := today.Format("2006-01-02")

	var out []Notification

	// today_plan + missed_goal both need plan facts.
	if g.plans != nil {
		ts, err := g.plans.PlanDay(ctx, userID, today)
		if err != nil {
			return nil, err
		}

		// today_plan: active roadmap has tasks today and none completed yet.
		if ts != nil && ts.TotalTasks > 0 && ts.CompletedTasks == 0 {
			hours := round1(float64(ts.EstimatedMins) / 60)
			n, err := g.upsert(ctx, userID, TypeTodayPlan, "today_plan:"+dayStr,
				fmt.Sprintf("You have %d task%s today (~%sh)", ts.TotalTasks, plural(ts.TotalTasks), trimFloat(hours)),
				"Start your plan to keep momentum.",
				map[string]any{"date": dayStr, "total_tasks": ts.TotalTasks, "estimated_minutes": ts.EstimatedMins},
			)
			if err != nil {
				return nil, err
			}
			out = append(out, *n)
		}

		// missed_goal: yesterday had planned tasks and none were completed.
		yesterday := today.AddDate(0, 0, -1)
		ys, err := g.plans.PlanDay(ctx, userID, yesterday)
		if err != nil {
			return nil, err
		}
		if ys != nil && ys.TotalTasks > 0 && ys.CompletedTasks == 0 {
			n, err := g.upsert(ctx, userID, TypeMissedGoal, "missed_goal:"+yesterday.Format("2006-01-02"),
				fmt.Sprintf("You missed %d planned task%s yesterday", ys.TotalTasks, plural(ys.TotalTasks)),
				"Reschedule or pick them up today to stay on track.",
				map[string]any{"date": yesterday.Format("2006-01-02"), "missed_tasks": ys.TotalTasks},
			)
			if err != nil {
				return nil, err
			}
			out = append(out, *n)
		}
	}

	// revision_due: active revision items due on/before today.
	if g.revisions != nil {
		due, err := g.revisions.DueCount(ctx, userID, today)
		if err != nil {
			return nil, err
		}
		if due > 0 {
			n, err := g.upsert(ctx, userID, TypeRevisionDue, "revision_due:"+dayStr,
				fmt.Sprintf("%d item%s due for revision", due, plural(due)),
				"Review them now to lock in what you've learned.",
				map[string]any{"date": dayStr, "due_count": due},
			)
			if err != nil {
				return nil, err
			}
			out = append(out, *n)
		}
	}

	// streak_reminder: an active streak exists but today isn't logged yet.
	if g.streaks != nil {
		current, loggedToday, err := g.streaks.Streak(ctx, userID, today)
		if err != nil {
			return nil, err
		}
		if current > 0 && !loggedToday {
			n, err := g.upsert(ctx, userID, TypeStreakReminder, "streak_reminder:"+dayStr,
				fmt.Sprintf("Keep your %d-day streak alive", current),
				"Complete a task today so your streak doesn't reset.",
				map[string]any{"date": dayStr, "current_streak": current},
			)
			if err != nil {
				return nil, err
			}
			out = append(out, *n)
		}
	}

	// readiness_milestone: latest readiness crossed a new 10-point threshold
	// relative to the previous snapshot.
	if g.readiness != nil {
		latest, previous, hasAny, err := g.readiness.LatestReadiness(ctx, userID)
		if err != nil {
			return nil, err
		}
		if hasAny {
			if milestone, crossed := crossedMilestone(previous, latest); crossed {
				n, err := g.upsert(ctx, userID, TypeReadinessMilestone,
					fmt.Sprintf("readiness_milestone:%d", milestone),
					fmt.Sprintf("You crossed %d%% interview readiness", milestone),
					"Great progress — keep going.",
					map[string]any{"milestone": milestone, "readiness": round1(latest)},
				)
				if err != nil {
					return nil, err
				}
				out = append(out, *n)
			}
		}
	}

	// weekly_review: a once-per-ISO-week digest summarizing the week's tasks and
	// readiness movement (FR-NOTIF-001). The dedup_key embeds the ISO year+week so
	// it fires at most once per calendar week and re-running the generator the
	// same week is idempotent.
	if g.weekly != nil {
		isoYear, isoWeek := today.ISOWeek()
		weekStart := isoWeekStart(today)
		weekEnd := weekStart.AddDate(0, 0, 6)
		tasksDone, delta, hasReadiness, err := g.weekly.WeeklySummary(ctx, userID, weekStart, weekEnd)
		if err != nil {
			return nil, err
		}
		// Only surface a review once there is something to report (any completed
		// task or any readiness signal this week), so brand-new users aren't nagged.
		if tasksDone > 0 || hasReadiness {
			payload := map[string]any{
				"iso_year":   isoYear,
				"iso_week":   isoWeek,
				"week_start": weekStart.Format("2006-01-02"),
				"week_end":   weekEnd.Format("2006-01-02"),
				"tasks_done": tasksDone,
			}
			body := fmt.Sprintf("You completed %d task%s this week.", tasksDone, plural(tasksDone))
			if hasReadiness {
				payload["readiness_delta"] = round1(delta)
				body = fmt.Sprintf("%s Readiness %s%s pts.", body, signPrefix(delta), trimFloat(round1(absFloat(delta))))
			}
			n, err := g.upsert(ctx, userID, TypeWeeklyReview,
				fmt.Sprintf("weekly_review:%04d-W%02d", isoYear, isoWeek),
				fmt.Sprintf("Your week in review (W%02d)", isoWeek),
				body, payload,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, *n)
		}
	}

	return out, nil
}

// isoWeekStart returns the Monday (00:00 UTC) of the ISO week containing t.
func isoWeekStart(t time.Time) time.Time {
	t = truncDay(t.UTC())
	// time.Weekday: Sunday=0..Saturday=6; ISO weeks start Monday.
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func signPrefix(v float64) string {
	if v < 0 {
		return "-"
	}
	return "+"
}

// crossedMilestone reports the highest multiple of readinessMilestoneStep that
// latest has reached but previous had not, i.e. a fresh milestone crossing.
func crossedMilestone(previous, latest float64) (int, bool) {
	prevM := int(previous/readinessMilestoneStep) * int(readinessMilestoneStep)
	latestM := int(latest/readinessMilestoneStep) * int(readinessMilestoneStep)
	if latestM > prevM && latestM > 0 {
		return latestM, true
	}
	return 0, false
}

// upsert builds a generated notification with a dedup_key and upserts it
// idempotently, returning the live row (created or pre-existing).
func (g *Generator) upsert(ctx context.Context, userID uuid.UUID, t Type, dedupKey, title, body string, payload map[string]any) (*Notification, error) {
	key := dedupKey
	n := &Notification{
		UserID:   userID,
		Type:     t,
		Channel:  ChannelInApp,
		Status:   StatusUnread,
		Title:    title,
		Body:     strOrNil(body),
		Payload:  JSONMap(payload),
		DedupKey: &key,
	}
	if n.Payload == nil {
		n.Payload = JSONMap{}
	}
	_, out, err := g.repo.UpsertByDedupKey(ctx, n)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --- formatting helpers ---

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

// trimFloat renders a 1-dp float without a trailing ".0" (e.g. 2 -> "2", 2.5 -> "2.5").
func trimFloat(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	if len(s) > 2 && s[len(s)-2:] == ".0" {
		return s[:len(s)-2]
	}
	return s
}
