package roadmap

import "errors"

// Domain errors returned by the roadmap service. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrNoActiveRoadmap indicates the user has no active roadmap (404).
	ErrNoActiveRoadmap = errors.New("roadmap: no active roadmap")
	// ErrNotFound indicates a requested roadmap/week/plan-day does not exist (404).
	ErrNotFound = errors.New("roadmap: not found")
	// ErrProfileRequired indicates the user must complete intake before
	// generating a roadmap (422).
	ErrProfileRequired = errors.New("roadmap: intake profile required")
	// ErrActiveRoadmapExists indicates an active roadmap already exists and
	// regenerate was not requested (409).
	ErrActiveRoadmapExists = errors.New("roadmap: active roadmap already exists")
)
