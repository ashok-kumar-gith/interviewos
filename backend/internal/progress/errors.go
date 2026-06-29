package progress

import "errors"

// Domain errors returned by the progress service. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrTaskNotFound indicates the task does not exist or is not owned by the
	// requesting user (404).
	ErrTaskNotFound = errors.New("progress: task not found")
	// ErrPlanDayNotFound indicates no plan-day exists for the requested date (404).
	ErrPlanDayNotFound = errors.New("progress: plan day not found")
	// ErrTaskAlreadyResolved indicates the task is already completed/skipped and
	// cannot be transitioned again (422).
	ErrTaskAlreadyResolved = errors.New("progress: task already resolved")
	// ErrInvalidConfidence indicates a confidence value outside 1..5 (422).
	ErrInvalidConfidence = errors.New("progress: confidence must be between 1 and 5")
	// ErrInvalidReschedule indicates a reschedule with no/invalid target date (422).
	ErrInvalidReschedule = errors.New("progress: invalid reschedule date")
	// ErrNoTargetPlanDay indicates the reschedule target date has no plan-day to
	// move the task into (422).
	ErrNoTargetPlanDay = errors.New("progress: no plan day for target date")
)
