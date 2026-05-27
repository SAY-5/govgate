// Package scoring evaluates a submission against a checklist and produces a
// per-category and overall risk assessment. Scoring is deterministic: the same
// submission and checklist always yield the same assessment.
package scoring

// Submission is a tool put forward for review. The free-text Description is the
// human narrative; StructuredFields hold the machine-checkable facts, which may
// be filled in by a human or extracted by the Python reporter's extractor.
type Submission struct {
	Name             string           `json:"name"`
	Vendor           string           `json:"vendor"`
	Description      string           `json:"description"`
	IntendedUse      string           `json:"intended_use"`
	StructuredFields StructuredFields `json:"structured_fields"`
}

// StructuredFields are the facts the auto-checks read. Pointer/optional fields
// distinguish "declared false" from "not declared at all" where it matters.
type StructuredFields struct {
	DataRegion            string   `json:"data_region"`
	ApprovedRegions       []string `json:"approved_regions"`
	ProcessesPII          *bool    `json:"processes_pii"`
	PIIMaskedBeforeSend   *bool    `json:"pii_masked_before_send"`
	ModelFamily           string   `json:"model_family"`
	RetentionDays         *int     `json:"retention_days"`
	EncryptionInTransit   *bool    `json:"encryption_in_transit"`
	Subprocessors         []string `json:"subprocessors"`
	UnvettedSubprocessors []string `json:"unvetted_subprocessors"`
	TrainsOnCustomerData  *bool    `json:"trains_on_customer_data"`
}

// Outcome is the result of evaluating a single requirement.
type Outcome string

const (
	// OutcomePass means the requirement is satisfied.
	OutcomePass Outcome = "pass"
	// OutcomeFail means the requirement is not satisfied.
	OutcomeFail Outcome = "fail"
	// OutcomeUnknown means there was insufficient information to decide. It is
	// treated as a failure for scoring but reported distinctly.
	OutcomeUnknown Outcome = "unknown"
)

// Judgments carries human/LLM-supplied outcomes for requirements that have no
// auto_check, keyed by requirement id. Missing entries default to unknown.
type Judgments map[string]Outcome
