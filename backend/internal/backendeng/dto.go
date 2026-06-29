package backendeng

import "github.com/interviewos/backend/internal/content"

// DTOs mirror the OpenAPI Topic / TopicDetail schemas (shared with the content
// catalog) so the backend-engineering endpoints return an identical shape.

// paginationMeta is the meta block of a paginated response.
type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// paginatedResponse is the canonical {data, meta} list envelope.
type paginatedResponse struct {
	Data any            `json:"data"`
	Meta paginationMeta `json:"meta"`
}

// topicResponse is the summary projection of a topic (list endpoint).
type topicResponse struct {
	ID             string  `json:"id"`
	PillarID       string  `json:"pillar_id"`
	TrackID        string  `json:"track_id"`
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	Summary        *string `json:"summary"`
	Difficulty     string  `json:"difficulty"`
	Priority       string  `json:"priority"`
	EstimatedHours float64 `json:"estimated_hours"`
	SortOrder      int     `json:"sort_order"`
}

func toTopicResponse(t content.Topic) topicResponse {
	return topicResponse{
		ID:             t.ID.String(),
		PillarID:       t.PillarID.String(),
		TrackID:        t.TrackID.String(),
		Slug:           t.Slug,
		Name:           t.Name,
		Summary:        t.Summary,
		Difficulty:     string(t.Difficulty),
		Priority:       string(t.Priority),
		EstimatedHours: t.EstimatedHours,
		SortOrder:      t.SortOrder,
	}
}

type subtopicResponse struct {
	ID             string  `json:"id"`
	TopicID        string  `json:"topic_id"`
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	ContentMD      *string `json:"content_md"`
	EstimatedHours float64 `json:"estimated_hours"`
	SortOrder      int     `json:"sort_order"`
}

type resourceResponse struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Title            string  `json:"title"`
	Author           *string `json:"author"`
	URL              *string `json:"url"`
	Provider         *string `json:"provider"`
	Description      *string `json:"description"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	Difficulty       *string `json:"difficulty"`
	Priority         string  `json:"priority"`
	IsFree           bool    `json:"is_free"`
}

// topicDetailResponse is the full topic projection (detail endpoint).
type topicDetailResponse struct {
	topicResponse
	ConceptMD         *string            `json:"concept_md"`
	CommonMistakes    *string            `json:"common_mistakes"`
	ExpectedQuestions []string           `json:"expected_questions"`
	Prerequisites     []string           `json:"prerequisites"`
	Subtopics         []subtopicResponse `json:"subtopics"`
	Resources         []resourceResponse `json:"resources"`
}

func toTopicDetailResponse(b *content.TopicBundle) topicDetailResponse {
	eq := []string(b.Topic.ExpectedQuestions)
	if eq == nil {
		eq = []string{}
	}
	pre := []string(b.Topic.Prerequisites)
	if pre == nil {
		pre = []string{}
	}
	subs := make([]subtopicResponse, 0, len(b.Subtopics))
	for _, s := range b.Subtopics {
		subs = append(subs, subtopicResponse{
			ID:             s.ID.String(),
			TopicID:        s.TopicID.String(),
			Slug:           s.Slug,
			Name:           s.Name,
			ContentMD:      s.ContentMD,
			EstimatedHours: s.EstimatedHours,
			SortOrder:      s.SortOrder,
		})
	}
	res := make([]resourceResponse, 0, len(b.Resources))
	for _, r := range b.Resources {
		var diff *string
		if r.Difficulty != nil {
			d := string(*r.Difficulty)
			diff = &d
		}
		res = append(res, resourceResponse{
			ID:               r.ID.String(),
			Type:             string(r.Type),
			Title:            r.Title,
			Author:           r.Author,
			URL:              r.URL,
			Provider:         r.Provider,
			Description:      r.Description,
			EstimatedMinutes: r.EstimatedMinutes,
			Difficulty:       diff,
			Priority:         string(r.Priority),
			IsFree:           r.IsFree,
		})
	}
	return topicDetailResponse{
		topicResponse:     toTopicResponse(b.Topic),
		ConceptMD:         b.Topic.ConceptMD,
		CommonMistakes:    b.Topic.CommonMistakes,
		ExpectedQuestions: eq,
		Prerequisites:     pre,
		Subtopics:         subs,
		Resources:         res,
	}
}
