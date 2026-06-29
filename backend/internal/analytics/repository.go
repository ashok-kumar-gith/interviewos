package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository abstracts persistence for the analytics domain so the service is
// unit-testable against a fake. The GORM implementation is gormRepository.
type Repository interface {
	// PillarInputs rolls up per-pillar coverage/confidence/revision-health and
	// mock scores for the user, applying company weights when a target company is
	// set (else uniform pillar content weights). This is the raw input to the
	// readiness calculator (SRS §6.2).
	PillarInputs(ctx context.Context, userID uuid.UUID) ([]PillarInputs, error)

	// TopicEntries returns per-topic analytics rows ranked by composite score
	// (coverage × confidence), used to bucket weak/strong topics.
	TopicEntries(ctx context.Context, userID uuid.UUID) ([]TopicEntry, error)

	// TimeSpent aggregates study-session minutes for the user over [from,to],
	// grouped by day or pillar.
	TimeSpent(ctx context.Context, userID uuid.UUID, from, to time.Time, groupBy string) (TimeSpent, error)

	// StreakDays returns the user's active study days (optionally bounded by
	// [from,to]; zero times mean unbounded).
	StreakDays(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]StreakDay, error)

	// ActiveRoadmapID returns the user's active roadmap id, or nil if none.
	ActiveRoadmapID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)

	// UpsertSnapshot writes (idempotently per user+day) the daily readiness
	// snapshot and returns the stored row decoded into a Snapshot.
	UpsertSnapshot(ctx context.Context, s Snapshot) (Snapshot, error)

	// ListSnapshots returns the user's snapshots over [from,to] (zero times =
	// unbounded) ordered by snapshot_date ascending, with total count for paging.
	ListSnapshots(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]Snapshot, int64, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) ActiveRoadmapID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var row struct {
		ID *uuid.UUID `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).Table("roadmaps").
		Select("id").
		Where("user_id = ? AND is_active AND deleted_at IS NULL", userID).
		Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return row.ID, nil
}

// PillarInputs computes the per-pillar coverage/confidence/revision-health/mock
// rollups (SRS §6.2.1/6.2.2). Coverage is item-based: completed items over
// in-scope items (skipped excluded) per pillar, derived from progress rows
// joined to content (topics/problems). Confidence is the mean rating over
// completed items. Revision health defaults to 1.0 (revision_items may be absent
// at M2). Pillar weights come from the target company's CompanyWeights when set,
// else the pillar content weight.
func (r *gormRepository) PillarInputs(ctx context.Context, userID uuid.UUID) ([]PillarInputs, error) {
	trackID, err := r.userTrackID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if trackID == uuid.Nil {
		return nil, ErrProfileNotFound
	}

	// In-scope content per pillar (denominators): topics + problems belonging to
	// the track's pillars. We count topics directly and problems via their topic.
	type covRow struct {
		Pillar    string `gorm:"column:pillar"`
		Total     int    `gorm:"column:total"`
		Completed int    `gorm:"column:completed"`
		ConfSum   int    `gorm:"column:conf_sum"`
		ConfCnt   int    `gorm:"column:conf_cnt"`
	}

	// Topics coverage: total in-scope topics per pillar and how many the user has
	// completed (status='completed'), plus confidence over completed topics.
	var topicRows []covRow
	if err := r.db.WithContext(ctx).Table("topics t").
		Select(`p.type AS pillar,
		        COUNT(t.id) AS total,
		        COUNT(utp.id) FILTER (WHERE utp.status = 'completed') AS completed,
		        COALESCE(SUM(utp.confidence) FILTER (WHERE utp.status = 'completed' AND utp.confidence IS NOT NULL), 0) AS conf_sum,
		        COUNT(utp.confidence) FILTER (WHERE utp.status = 'completed' AND utp.confidence IS NOT NULL) AS conf_cnt`).
		Joins("JOIN pillars p ON p.id = t.pillar_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN user_topic_progress utp ON utp.topic_id = t.id AND utp.user_id = ? AND utp.deleted_at IS NULL", userID).
		Where("t.track_id = ? AND t.deleted_at IS NULL", trackID).
		Group("p.type").Scan(&topicRows).Error; err != nil {
		return nil, err
	}

	// Aggregate per pillar.
	agg := map[string]*covRow{}
	for i := range topicRows {
		row := topicRows[i]
		cur := agg[row.Pillar]
		if cur == nil {
			cur = &covRow{Pillar: row.Pillar}
			agg[row.Pillar] = cur
		}
		cur.Total += row.Total
		cur.Completed += row.Completed
		cur.ConfSum += row.ConfSum
		cur.ConfCnt += row.ConfCnt
	}

	// Problem coverage folds into the DSA pillar (problems are DSA content).
	type probRow struct {
		Total     int `gorm:"column:total"`
		Completed int `gorm:"column:completed"`
		ConfSum   int `gorm:"column:conf_sum"`
		ConfCnt   int `gorm:"column:conf_cnt"`
	}
	var pr probRow
	if err := r.db.WithContext(ctx).Table("problems pb").
		Select(`COUNT(pb.id) AS total,
		        COUNT(upp.id) FILTER (WHERE upp.status = 'completed') AS completed,
		        COALESCE(SUM(upp.confidence) FILTER (WHERE upp.status = 'completed' AND upp.confidence IS NOT NULL), 0) AS conf_sum,
		        COUNT(upp.confidence) FILTER (WHERE upp.status = 'completed' AND upp.confidence IS NOT NULL) AS conf_cnt`).
		Joins("LEFT JOIN user_problem_progress upp ON upp.problem_id = pb.id AND upp.user_id = ? AND upp.deleted_at IS NULL", userID).
		Where("pb.track_id = ? AND pb.deleted_at IS NULL", trackID).
		Scan(&pr).Error; err != nil {
		// problems may be track-scoped differently; tolerate absence by treating
		// as no problem coverage rather than failing analytics.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if pr.Total > 0 {
		cur := agg[string(pillarDSA)]
		if cur == nil {
			cur = &covRow{Pillar: string(pillarDSA)}
			agg[string(pillarDSA)] = cur
		}
		cur.Total += pr.Total
		cur.Completed += pr.Completed
		cur.ConfSum += pr.ConfSum
		cur.ConfCnt += pr.ConfCnt
	}

	// Pillar weights: company weights (normalized multipliers) when targeted,
	// else pillar content weight.
	weights, err := r.pillarWeights(ctx, userID, trackID)
	if err != nil {
		return nil, err
	}

	// Mock scores per pillar (latest/average outcome, normalized to [0,1]).
	mocks, err := r.mockScores(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Revision health per pillar (defaults 1.0 when revision_items absent).
	revHealth, err := r.revisionHealth(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]PillarInputs, 0, len(agg))
	for pillar, row := range agg {
		avg := 0.0
		if row.ConfCnt > 0 {
			avg = float64(row.ConfSum) / float64(row.ConfCnt)
		}
		rh := 1.0
		if v, ok := revHealth[pillar]; ok {
			rh = v
		}
		in := PillarInputs{
			Pillar:         pillar,
			Weight:         weights[pillar],
			CompletedItems: row.Completed,
			TotalItems:     row.Total,
			AvgRating:      avg,
			RevHealth:      rh,
		}
		if ms, ok := mocks[pillar]; ok {
			in.HasMock = true
			in.MockScore = ms
		}
		out = append(out, in)
	}
	return out, nil
}

// userTrackID resolves the active track for the user from their profile. The id
// is scanned through a struct field because GORM cannot scan a bare SQL value
// into a uuid.UUID (a [16]byte array).
func (r *gormRepository) userTrackID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var row struct {
		TrackID *uuid.UUID `gorm:"column:track_id"`
	}
	err := r.db.WithContext(ctx).Table("user_profiles").
		Select("track_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").Limit(1).Scan(&row).Error
	if err != nil {
		return uuid.Nil, err
	}
	if row.TrackID == nil {
		return uuid.Nil, nil
	}
	return *row.TrackID, nil
}

// pillarWeights returns the per-pillar weight map. When the user targets a
// company, the company's pillar-level weight multipliers are used (defaulting to
// 1.0 for pillars without an explicit multiplier); otherwise the pillar content
// weight is used. Weights are returned un-normalized (the calculator normalizes).
func (r *gormRepository) pillarWeights(ctx context.Context, userID, trackID uuid.UUID) (map[string]float64, error) {
	type wRow struct {
		Type   string  `gorm:"column:type"`
		Weight float64 `gorm:"column:weight"`
	}
	var base []wRow
	if err := r.db.WithContext(ctx).Table("pillars").
		Select("type, weight").
		Where("track_id = ? AND deleted_at IS NULL", trackID).
		Scan(&base).Error; err != nil {
		return nil, err
	}
	weights := map[string]float64{}
	for _, w := range base {
		weights[w.Type] = w.Weight
		if weights[w.Type] <= 0 {
			weights[w.Type] = 1.0
		}
	}

	// Target company?
	var crow struct {
		CompanyID *uuid.UUID `gorm:"column:target_company_id"`
	}
	if err := r.db.WithContext(ctx).Table("user_profiles").
		Select("target_company_id").
		Where("user_id = ? AND deleted_at IS NULL AND target_company_id IS NOT NULL", userID).
		Order("created_at DESC").Limit(1).Scan(&crow).Error; err != nil {
		return nil, err
	}
	if crow.CompanyID == nil {
		return weights, nil
	}
	companyID := *crow.CompanyID

	// Apply company pillar-level multipliers (CompanyWeight with a pillar_id).
	type cwRow struct {
		Type       string  `gorm:"column:type"`
		Multiplier float64 `gorm:"column:weight_multiplier"`
	}
	var cw []cwRow
	if err := r.db.WithContext(ctx).Table("company_weights cw").
		Select("p.type AS type, cw.weight_multiplier AS weight_multiplier").
		Joins("JOIN pillars p ON p.id = cw.pillar_id").
		Where("cw.company_id = ? AND cw.pillar_id IS NOT NULL AND p.deleted_at IS NULL", companyID).
		Scan(&cw).Error; err != nil {
		return nil, err
	}
	for _, c := range cw {
		base := weights[c.Type]
		if base <= 0 {
			base = 1.0
		}
		weights[c.Type] = base * c.Multiplier
	}
	return weights, nil
}

// mockScores returns the mean outcome score per pillar over the user's mock
// interviews, normalized to [0,1]. Mock scores are stored on a 0..100 (or 1..5)
// scale; we normalize defensively. Returns an empty map when no mocks exist or
// the table is absent.
func (r *gormRepository) mockScores(ctx context.Context, userID uuid.UUID) (map[string]float64, error) {
	if !r.db.Migrator().HasTable("mock_interviews") {
		return map[string]float64{}, nil
	}
	type mRow struct {
		Type string  `gorm:"column:type"`
		Avg  float64 `gorm:"column:avg_score"`
	}
	var rows []mRow
	// mock_interviews carries a mock type and a numeric overall_score; we map the
	// mock type to its pillar (coding→dsa) and normalize the score below.
	if err := r.db.WithContext(ctx).Table("mock_interviews").
		Select("type, AVG(overall_score) AS avg_score").
		Where("user_id = ? AND deleted_at IS NULL AND overall_score IS NOT NULL", userID).
		Group("type").Scan(&rows).Error; err != nil {
		// Tolerate schema differences in the parallel mock module.
		return map[string]float64{}, nil
	}
	out := map[string]float64{}
	for _, m := range rows {
		out[mockTypeToPillar(m.Type)] = normalizeScore(m.Avg)
	}
	return out, nil
}

// normalizeScore maps a raw mock outcome score to [0,1]. Scores in (1..5] are
// treated as a 1–5 rating; scores in (5..100] as a percentage; already-normalized
// scores in [0,1] pass through.
func normalizeScore(v float64) float64 {
	switch {
	case v <= 0:
		return 0
	case v <= 1:
		return v
	case v <= 5:
		return clamp01((v - 1) / 4)
	default:
		return clamp01(v / 100)
	}
}

// revisionHealth returns mean per-pillar revision health in [0,1], defaulting to
// absent (caller uses 1.0) when the revision_items table does not yet exist
// (the parallel feature). When present, it computes SRS §6.2.1 health per active
// item and averages by pillar.
func (r *gormRepository) revisionHealth(ctx context.Context, userID uuid.UUID) (map[string]float64, error) {
	if !r.db.Migrator().HasTable("revision_items") {
		return map[string]float64{}, nil
	}
	// Health per item (SRS §6.2.1):
	//   graduated (is_active=false) ⇒ 1.0
	//   active: 0.5·(stage/4) + 0.5·(last_recall==correct?1:0), overdue decay
	//           × max(0.3, 1 - 0.1·days_overdue)
	type rhRow struct {
		Pillar string  `gorm:"column:pillar"`
		Health float64 `gorm:"column:health"`
		Cnt    int     `gorm:"column:cnt"`
	}
	var rows []rhRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT pillar_type AS pillar,
		       AVG(health) AS health,
		       COUNT(*) AS cnt
		FROM (
		    SELECT pillar_type,
		           CASE
		             WHEN NOT is_active THEN 1.0
		             ELSE GREATEST(
		               CASE WHEN due_at < CURRENT_DATE
		                    THEN GREATEST(0.3, 1 - 0.1 * (CURRENT_DATE - due_at))
		                    ELSE 1 END
		               * (0.5 * (LEAST(stage,4)::numeric/4) + 0.5 * (CASE WHEN last_recall = 'correct' THEN 1 ELSE 0 END)),
		               0)
		           END AS health
		    FROM revision_items
		    WHERE user_id = ? AND deleted_at IS NULL
		) s
		GROUP BY pillar_type`, userID).Scan(&rows).Error
	if err != nil {
		// Tolerate schema differences; fall back to the 1.0 default.
		return map[string]float64{}, nil
	}
	out := map[string]float64{}
	for _, rh := range rows {
		if rh.Cnt > 0 {
			out[rh.Pillar] = clamp01(rh.Health)
		}
	}
	return out, nil
}

func (r *gormRepository) TopicEntries(ctx context.Context, userID uuid.UUID) ([]TopicEntry, error) {
	trackID, err := r.userTrackID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if trackID == uuid.Nil {
		return nil, ErrProfileNotFound
	}
	type tRow struct {
		TopicID    uuid.UUID `gorm:"column:topic_id"`
		TopicName  string    `gorm:"column:topic_name"`
		Pillar     string    `gorm:"column:pillar"`
		Status     *string   `gorm:"column:status"`
		Confidence *int      `gorm:"column:confidence"`
	}
	var rows []tRow
	if err := r.db.WithContext(ctx).Table("topics t").
		Select(`t.id AS topic_id, t.name AS topic_name, p.type AS pillar,
		        utp.status AS status, utp.confidence AS confidence`).
		Joins("JOIN pillars p ON p.id = t.pillar_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN user_topic_progress utp ON utp.topic_id = t.id AND utp.user_id = ? AND utp.deleted_at IS NULL", userID).
		Where("t.track_id = ? AND t.deleted_at IS NULL", trackID).
		Order("t.sort_order ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TopicEntry, 0, len(rows))
	for _, row := range rows {
		completion := 0.0
		if row.Status != nil && *row.Status == "completed" {
			completion = 1.0
		}
		conf := 0.0
		if row.Confidence != nil {
			conf = clamp01((float64(*row.Confidence) - 1) / 4)
		}
		out = append(out, TopicEntry{
			TopicID:       row.TopicID,
			TopicName:     row.TopicName,
			PillarType:    row.Pillar,
			Confidence:    row.Confidence,
			CompletionPct: completion * 100,
			Score:         completion * (0.5 + 0.5*conf),
		})
	}
	return out, nil
}

func (r *gormRepository) TimeSpent(ctx context.Context, userID uuid.UUID, from, to time.Time, groupBy string) (TimeSpent, error) {
	q := r.db.WithContext(ctx).Table("study_sessions").
		Where("user_id = ? AND deleted_at IS NULL", userID)
	if !from.IsZero() {
		q = q.Where("started_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("started_at < ?", to.AddDate(0, 0, 1))
	}

	type bRow struct {
		Key     string `gorm:"column:key"`
		Minutes int    `gorm:"column:minutes"`
	}
	var rows []bRow
	if groupBy == "pillar" {
		if err := q.Select("COALESCE(pillar_type::text, 'unknown') AS key, COALESCE(SUM(duration_minutes),0) AS minutes").
			Group("pillar_type").Order("key ASC").Scan(&rows).Error; err != nil {
			return TimeSpent{}, err
		}
	} else {
		groupBy = "day"
		if err := q.Select("to_char(started_at, 'YYYY-MM-DD') AS key, COALESCE(SUM(duration_minutes),0) AS minutes").
			Group("to_char(started_at, 'YYYY-MM-DD')").Order("key ASC").Scan(&rows).Error; err != nil {
			return TimeSpent{}, err
		}
	}
	out := TimeSpent{GroupBy: groupBy, Buckets: make([]TimeBucket, 0, len(rows))}
	for _, b := range rows {
		out.Buckets = append(out.Buckets, TimeBucket{Key: b.Key, Minutes: b.Minutes})
		out.TotalMinutes += b.Minutes
	}
	return out, nil
}

func (r *gormRepository) StreakDays(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]StreakDay, error) {
	q := r.db.WithContext(ctx).Table("streak_days").
		Where("user_id = ? AND deleted_at IS NULL", userID)
	if !from.IsZero() {
		q = q.Where("date >= ?", from.Format("2006-01-02"))
	}
	if !to.IsZero() {
		q = q.Where("date <= ?", to.Format("2006-01-02"))
	}
	type sRow struct {
		Date           time.Time `gorm:"column:date"`
		TasksCompleted int       `gorm:"column:tasks_completed"`
		MinutesStudied int       `gorm:"column:minutes_studied"`
		GoalMet        bool      `gorm:"column:goal_met"`
	}
	var rows []sRow
	if err := q.Select("date, tasks_completed, minutes_studied, goal_met").
		Order("date ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]StreakDay, 0, len(rows))
	for _, s := range rows {
		out = append(out, StreakDay{
			Date:           s.Date,
			TasksCompleted: s.TasksCompleted,
			MinutesStudied: s.MinutesStudied,
			GoalMet:        s.GoalMet,
		})
	}
	return out, nil
}

func (r *gormRepository) UpsertSnapshot(ctx context.Context, s Snapshot) (Snapshot, error) {
	pillarJSON, err := json.Marshal(s.PillarReadiness)
	if err != nil {
		return Snapshot{}, err
	}
	weakJSON, err := json.Marshal(uuidStrings(s.WeakTopics))
	if err != nil {
		return Snapshot{}, err
	}
	strongJSON, err := json.Marshal(uuidStrings(s.StrongTopics))
	if err != nil {
		return Snapshot{}, err
	}

	row := ReadinessSnapshot{
		UserID:             s.UserID,
		RoadmapID:          s.RoadmapID,
		SnapshotDate:       s.SnapshotDate,
		OverallReadiness:   round2(s.OverallReadiness),
		PillarReadiness:    pillarJSON,
		CompletionPct:      round2(s.CompletionPct),
		AvgConfidence:      s.AvgConfidence,
		RevisionHealth:     s.RevisionHealth,
		EstimatedReadyDate: s.EstimatedReadyDate,
		WeakTopics:         weakJSON,
		StrongTopics:       strongJSON,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "user_id"}, {Name: "snapshot_date"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
		DoUpdates: clause.Assignments(map[string]any{
			"roadmap_id":           row.RoadmapID,
			"overall_readiness":    row.OverallReadiness,
			"pillar_readiness":     row.PillarReadiness,
			"completion_pct":       row.CompletionPct,
			"avg_confidence":       row.AvgConfidence,
			"revision_health":      row.RevisionHealth,
			"estimated_ready_date": row.EstimatedReadyDate,
			"weak_topics":          row.WeakTopics,
			"strong_topics":        row.StrongTopics,
			"updated_at":           time.Now().UTC(),
		}),
	}).Create(&row).Error; err != nil {
		return Snapshot{}, err
	}

	// Re-read to return the stored (and possibly conflict-updated) row.
	var stored ReadinessSnapshot
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND snapshot_date = ? AND deleted_at IS NULL", s.UserID, s.SnapshotDate.Format("2006-01-02")).
		First(&stored).Error; err != nil {
		return Snapshot{}, err
	}
	return decodeSnapshot(stored)
}

func (r *gormRepository) ListSnapshots(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]Snapshot, int64, error) {
	q := r.db.WithContext(ctx).Model(&ReadinessSnapshot{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)
	if !from.IsZero() {
		q = q.Where("snapshot_date >= ?", from.Format("2006-01-02"))
	}
	if !to.IsZero() {
		q = q.Where("snapshot_date <= ?", to.Format("2006-01-02"))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []ReadinessSnapshot
	page := q.Order("snapshot_date ASC")
	if limit > 0 {
		page = page.Limit(limit).Offset(offset)
	}
	if err := page.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]Snapshot, 0, len(rows))
	for i := range rows {
		s, err := decodeSnapshot(rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, nil
}

// decodeSnapshot maps a stored row into the service Snapshot view, decoding the
// JSONB fields.
func decodeSnapshot(row ReadinessSnapshot) (Snapshot, error) {
	s := Snapshot{
		ID:                 row.ID,
		UserID:             row.UserID,
		RoadmapID:          row.RoadmapID,
		SnapshotDate:       row.SnapshotDate,
		OverallReadiness:   row.OverallReadiness,
		CompletionPct:      row.CompletionPct,
		AvgConfidence:      row.AvgConfidence,
		RevisionHealth:     row.RevisionHealth,
		EstimatedReadyDate: row.EstimatedReadyDate,
		PillarReadiness:    map[string]float64{},
	}
	if len(row.PillarReadiness) > 0 {
		if err := json.Unmarshal(row.PillarReadiness, &s.PillarReadiness); err != nil {
			return Snapshot{}, err
		}
	}
	weak, err := parseUUIDs(row.WeakTopics)
	if err != nil {
		return Snapshot{}, err
	}
	strong, err := parseUUIDs(row.StrongTopics)
	if err != nil {
		return Snapshot{}, err
	}
	s.WeakTopics = weak
	s.StrongTopics = strong
	return s, nil
}

func parseUUIDs(b []byte) ([]uuid.UUID, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var strs []string
	if err := json.Unmarshal(b, &strs); err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
