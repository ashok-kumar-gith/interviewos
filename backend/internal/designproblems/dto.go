package designproblems

// DTOs mirror the OpenAPI DesignProblem / DesignProblemDetail schemas. Nullable
// fields use pointers so absent values serialize as JSON null per the contract.

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

// designProblemResponse is the list-view DesignProblem schema.
type designProblemResponse struct {
	ID         string `json:"id"`
	TrackID    string `json:"track_id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	OrderIndex int    `json:"order_index"`
}

func toDesignProblemResponse(d DesignProblem) designProblemResponse {
	return designProblemResponse{
		ID:         d.ID.String(),
		TrackID:    d.TrackID.String(),
		Slug:       d.Slug,
		Title:      d.Title,
		Difficulty: string(d.Difficulty),
		OrderIndex: d.OrderIndex,
	}
}

// designProblemDetailResponse is the DesignProblemDetail schema with all
// structured sections.
type designProblemDetailResponse struct {
	designProblemResponse
	RequirementsMD       *string  `json:"requirements_md"`
	CapacityEstimationMD *string  `json:"capacity_estimation_md"`
	APIDesignMD          *string  `json:"api_design_md"`
	DataModelMD          *string  `json:"data_model_md"`
	HighLevelDesignMD    *string  `json:"high_level_design_md"`
	CachingMD            *string  `json:"caching_md"`
	QueueingMD           *string  `json:"queueing_md"`
	ScalingMD            *string  `json:"scaling_md"`
	TradeoffsMD          *string  `json:"tradeoffs_md"`
	FailureHandlingMD    *string  `json:"failure_handling_md"`
	AlternativesMD       *string  `json:"alternatives_md"`
	InterviewTipsMD      *string  `json:"interview_tips_md"`
	FollowUpQuestions    []string `json:"follow_up_questions"`
}

func toDesignProblemDetailResponse(d *DesignProblem) designProblemDetailResponse {
	fq := []string(d.FollowUpQuestions)
	if fq == nil {
		fq = []string{}
	}
	return designProblemDetailResponse{
		designProblemResponse: toDesignProblemResponse(*d),
		RequirementsMD:        d.RequirementsMD,
		CapacityEstimationMD:  d.CapacityEstimationMD,
		APIDesignMD:           d.APIDesignMD,
		DataModelMD:           d.DataModelMD,
		HighLevelDesignMD:     d.HighLevelDesignMD,
		CachingMD:             d.CachingMD,
		QueueingMD:            d.QueueingMD,
		ScalingMD:             d.ScalingMD,
		TradeoffsMD:           d.TradeoffsMD,
		FailureHandlingMD:     d.FailureHandlingMD,
		AlternativesMD:        d.AlternativesMD,
		InterviewTipsMD:       d.InterviewTipsMD,
		FollowUpQuestions:     fq,
	}
}
