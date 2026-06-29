// Package notification owns in-app notifications: user-scoped listing (with an
// optional status filter and pagination), marking a single notification read,
// and marking all read. It also exposes a narrow write-only Notifier interface
// (Create) so other modules — revision (revision_due), streak/goal reminders,
// the planner (today_plan), readiness (readiness_milestone), mocks
// (mock_scheduled), etc. — can enqueue notifications without depending on the
// full read/query surface.
//
// Only the in_app channel is delivered at GA (per the ADR open-question
// default); the email/push channel values exist in the schema so future
// delivery channels need no migration. All reads/writes are user-scoped and
// soft-delete aware.
package notification
