// Package behavioral owns the STAR behavioral-story builder: CRUD over a user's
// behavioral_stories plus a deterministic AI "improve" feature that flags weak
// framing, missing metrics, and weak action verbs and produces a strength
// score. The improver is hidden behind the Improver interface so a real
// Claude-API implementation can replace the deterministic stub later.
package behavioral
