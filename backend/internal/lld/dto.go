package lld

// DTOs mirror the OpenAPI LLDProblem / LLDProblemDetail schemas. Nullable
// section fields use pointers so absent values serialize as JSON null per the
// contract; JSONB lists always serialize as a (possibly empty) array.

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

func metaFor(r ListResult) paginationMeta {
	return paginationMeta{
		Page:       r.Page,
		PageSize:   r.PageSize,
		Total:      r.Total,
		TotalPages: r.TotalPages,
	}
}

// problemResponse is the LLDProblem summary returned by the list endpoint.
type problemResponse struct {
	ID         string `json:"id"`
	TrackID    string `json:"track_id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	OrderIndex int    `json:"order_index"`
}

func toProblemResponse(p Problem) problemResponse {
	return problemResponse{
		ID:         p.ID.String(),
		TrackID:    p.TrackID.String(),
		Slug:       p.Slug,
		Title:      p.Title,
		Difficulty: string(p.Difficulty),
		OrderIndex: p.OrderIndex,
	}
}

// problemDetailResponse is the LLDProblemDetail returned by the get endpoint.
type problemDetailResponse struct {
	problemResponse
	RequirementsMD    *string  `json:"requirements_md"`
	EntitiesMD        *string  `json:"entities_md"`
	ClassDiagramMD    *string  `json:"class_diagram_md"`
	DesignPatterns    []string `json:"design_patterns"`
	SolidNotesMD      *string  `json:"solid_notes_md"`
	APIOrInterfaceMD  *string  `json:"api_or_interface_md"`
	TradeoffsMD       *string  `json:"tradeoffs_md"`
	FollowUpQuestions []string `json:"follow_up_questions"`
}

func toProblemDetailResponse(p *Problem) problemDetailResponse {
	patterns := []string(p.DesignPatterns)
	if patterns == nil {
		patterns = []string{}
	}
	followUps := []string(p.FollowUpQuestions)
	if followUps == nil {
		followUps = []string{}
	}
	return problemDetailResponse{
		problemResponse:   toProblemResponse(*p),
		RequirementsMD:    p.RequirementsMD,
		EntitiesMD:        p.EntitiesMD,
		ClassDiagramMD:    p.ClassDiagramMD,
		DesignPatterns:    patterns,
		SolidNotesMD:      p.SolidNotesMD,
		APIOrInterfaceMD:  p.APIOrInterfaceMD,
		TradeoffsMD:       p.TradeoffsMD,
		FollowUpQuestions: followUps,
	}
}
