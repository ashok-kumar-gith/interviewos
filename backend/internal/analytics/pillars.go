package analytics

// pillarType enumerates the pillar_type values used across analytics (mirrors
// the DB pillar_type enum). Kept module-local so analytics carries no extra
// dependency on other modules' constants.
type pillarType string

const (
	pillarDSA                pillarType = "dsa"
	pillarSystemDesign       pillarType = "system_design"
	pillarLLD                pillarType = "lld"
	pillarBackendEngineering pillarType = "backend_engineering"
	pillarBehavioral         pillarType = "behavioral"
	pillarResume             pillarType = "resume"
)

// mockTypeToPillar maps a mock_type enum value to its pillar_type. The mock
// "coding" type corresponds to the DSA pillar; the remainder share their name.
func mockTypeToPillar(mockType string) string {
	switch mockType {
	case "coding":
		return string(pillarDSA)
	case "system_design":
		return string(pillarSystemDesign)
	case "lld":
		return string(pillarLLD)
	case "behavioral":
		return string(pillarBehavioral)
	case "backend_engineering":
		return string(pillarBackendEngineering)
	default:
		return mockType
	}
}
