package content

// DTOs mirror the OpenAPI content/DSA/company schemas. Nullable fields use
// pointers so absent values serialize as JSON null per the contract.

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

func metaFor[T any](r ListResult[T]) paginationMeta {
	return paginationMeta{
		Page:       r.Page,
		PageSize:   r.PageSize,
		Total:      r.Total,
		TotalPages: r.TotalPages,
	}
}

type trackResponse struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Seniority   *string `json:"seniority"`
	IsActive    bool    `json:"is_active"`
	SortOrder   int     `json:"sort_order"`
}

func toTrackResponse(t Track) trackResponse {
	return trackResponse{
		ID:          t.ID.String(),
		Slug:        t.Slug,
		Name:        t.Name,
		Description: t.Description,
		Seniority:   t.Seniority,
		IsActive:    t.IsActive,
		SortOrder:   t.SortOrder,
	}
}

type pillarResponse struct {
	ID          string  `json:"id"`
	TrackID     string  `json:"track_id"`
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Weight      float64 `json:"weight"`
	SortOrder   int     `json:"sort_order"`
}

func toPillarResponse(p Pillar) pillarResponse {
	return pillarResponse{
		ID:          p.ID.String(),
		TrackID:     p.TrackID.String(),
		Type:        string(p.Type),
		Name:        p.Name,
		Description: p.Description,
		Weight:      p.Weight,
		SortOrder:   p.SortOrder,
	}
}

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

func toTopicResponse(t Topic) topicResponse {
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

func toSubtopicResponse(s Subtopic) subtopicResponse {
	return subtopicResponse{
		ID:             s.ID.String(),
		TopicID:        s.TopicID.String(),
		Slug:           s.Slug,
		Name:           s.Name,
		ContentMD:      s.ContentMD,
		EstimatedHours: s.EstimatedHours,
		SortOrder:      s.SortOrder,
	}
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

func toResourceResponse(r Resource) resourceResponse {
	var diff *string
	if r.Difficulty != nil {
		s := string(*r.Difficulty)
		diff = &s
	}
	return resourceResponse{
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
	}
}

type topicDetailResponse struct {
	topicResponse
	ConceptMD         *string            `json:"concept_md"`
	CommonMistakes    *string            `json:"common_mistakes"`
	ExpectedQuestions []string           `json:"expected_questions"`
	Prerequisites     []string           `json:"prerequisites"`
	Subtopics         []subtopicResponse `json:"subtopics"`
	Resources         []resourceResponse `json:"resources"`
}

func toTopicDetailResponse(b *TopicBundle) topicDetailResponse {
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
		subs = append(subs, toSubtopicResponse(s))
	}
	res := make([]resourceResponse, 0, len(b.Resources))
	for _, r := range b.Resources {
		res = append(res, toResourceResponse(r))
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

type patternResponse struct {
	ID          string  `json:"id"`
	TrackID     string  `json:"track_id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	WhenToUse   *string `json:"when_to_use"`
}

func toPatternResponse(p Pattern) patternResponse {
	return patternResponse{
		ID:          p.ID.String(),
		TrackID:     p.TrackID.String(),
		Slug:        p.Slug,
		Name:        p.Name,
		Description: p.Description,
		WhenToUse:   p.WhenToUse,
	}
}

type problemResponse struct {
	ID               string  `json:"id"`
	TrackID          string  `json:"track_id"`
	TopicID          *string `json:"topic_id"`
	Slug             string  `json:"slug"`
	Title            string  `json:"title"`
	Difficulty       string  `json:"difficulty"`
	Platform         string  `json:"platform"`
	ExternalID       *string `json:"external_id"`
	URL              *string `json:"url"`
	EstimatedMinutes int     `json:"estimated_minutes"`
	FrequencyScore   float64 `json:"frequency_score"`
	IsPremium        bool    `json:"is_premium"`
}

func toProblemResponse(p Problem) problemResponse {
	var topicID *string
	if p.TopicID != nil {
		s := p.TopicID.String()
		topicID = &s
	}
	return problemResponse{
		ID:               p.ID.String(),
		TrackID:          p.TrackID.String(),
		TopicID:          topicID,
		Slug:             p.Slug,
		Title:            p.Title,
		Difficulty:       string(p.Difficulty),
		Platform:         string(p.Platform),
		ExternalID:       p.ExternalID,
		URL:              p.URL,
		EstimatedMinutes: p.EstimatedMinutes,
		FrequencyScore:   p.FrequencyScore,
		IsPremium:        p.IsPremium,
	}
}

type problemSourceResponse struct {
	Source     string  `json:"source"`
	SourceRank *int    `json:"source_rank"`
	SourceURL  *string `json:"source_url"`
}

type companyFrequencyResponse struct {
	CompanyID      string  `json:"company_id"`
	CompanyName    string  `json:"company_name"`
	Frequency      float64 `json:"frequency"`
	LastSeenPeriod *string `json:"last_seen_period"`
}

type problemDetailResponse struct {
	problemResponse
	PromptSummary    *string                    `json:"prompt_summary"`
	ApproachMD       *string                    `json:"approach_md"`
	CommonMistakes   *string                    `json:"common_mistakes"`
	Patterns         []patternResponse          `json:"patterns"`
	Sources          []problemSourceResponse    `json:"sources"`
	CompanyFrequency []companyFrequencyResponse `json:"company_frequency"`
}

func toProblemDetailResponse(b *ProblemBundle) problemDetailResponse {
	pats := make([]patternResponse, 0, len(b.Patterns))
	for _, p := range b.Patterns {
		pats = append(pats, toPatternResponse(p))
	}
	srcs := make([]problemSourceResponse, 0, len(b.Sources))
	for _, s := range b.Sources {
		srcs = append(srcs, problemSourceResponse{
			Source:     string(s.Source),
			SourceRank: s.SourceRank,
			SourceURL:  s.SourceURL,
		})
	}
	freqs := make([]companyFrequencyResponse, 0, len(b.CompanyFrequency))
	for _, f := range b.CompanyFrequency {
		freqs = append(freqs, companyFrequencyResponse{
			CompanyID:      f.CompanyID.String(),
			CompanyName:    f.CompanyName,
			Frequency:      f.Frequency,
			LastSeenPeriod: f.LastSeenPeriod,
		})
	}
	return problemDetailResponse{
		problemResponse:  toProblemResponse(b.Problem),
		PromptSummary:    b.Problem.PromptSummary,
		ApproachMD:       b.Problem.ApproachMD,
		CommonMistakes:   b.Problem.CommonMistakes,
		Patterns:         pats,
		Sources:          srcs,
		CompanyFrequency: freqs,
	}
}

type companyResponse struct {
	ID              string  `json:"id"`
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	LogoURL         *string `json:"logo_url"`
	Description     *string `json:"description"`
	IsFullyWeighted bool    `json:"is_fully_weighted"`
}

func toCompanyResponse(c Company) companyResponse {
	return companyResponse{
		ID:              c.ID.String(),
		Slug:            c.Slug,
		Name:            c.Name,
		LogoURL:         c.LogoURL,
		Description:     c.Description,
		IsFullyWeighted: c.IsFullyWeighted,
	}
}

type companyWeightResponse struct {
	PillarID         *string `json:"pillar_id"`
	TopicID          *string `json:"topic_id"`
	WeightMultiplier float64 `json:"weight_multiplier"`
	Note             *string `json:"note"`
}

type companyDetailResponse struct {
	companyResponse
	InterviewStyleMD *string                 `json:"interview_style_md"`
	Weights          []companyWeightResponse `json:"weights"`
}

func toCompanyDetailResponse(b *CompanyBundle) companyDetailResponse {
	weights := make([]companyWeightResponse, 0, len(b.Weights))
	for _, w := range b.Weights {
		var pillarID, topicID *string
		if w.PillarID != nil {
			s := w.PillarID.String()
			pillarID = &s
		}
		if w.TopicID != nil {
			s := w.TopicID.String()
			topicID = &s
		}
		weights = append(weights, companyWeightResponse{
			PillarID:         pillarID,
			TopicID:          topicID,
			WeightMultiplier: w.WeightMultiplier,
			Note:             w.Note,
		})
	}
	return companyDetailResponse{
		companyResponse:  toCompanyResponse(b.Company),
		InterviewStyleMD: b.Company.InterviewStyleMD,
		Weights:          weights,
	}
}
