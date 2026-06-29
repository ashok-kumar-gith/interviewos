package revision

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP status
// codes and the standard error envelope (see handler.go).
var (
	// ErrItemNotFound indicates the revision item does not exist or is not owned
	// by the requesting user (404). Ownership failures are surfaced as not-found
	// so a caller cannot probe for another user's items.
	ErrItemNotFound = errors.New("revision: item not found")
	// ErrItemGraduated indicates a recall was submitted against an already
	// graduated (inactive) item (422).
	ErrItemGraduated = errors.New("revision: item already graduated")
	// ErrInvalidRecall indicates the recall value was not correct|incorrect (422).
	ErrInvalidRecall = errors.New("revision: recall must be correct or incorrect")
	// ErrUnschedulableItemType indicates the item_type is not eligible for a
	// revision item (only learning content: topic|subtopic|design_problem|
	// lld_problem|problem).
	ErrUnschedulableItemType = errors.New("revision: item type is not schedulable")
)
