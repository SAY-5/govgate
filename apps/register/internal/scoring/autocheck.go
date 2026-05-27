package scoring

import "strings"

// autoCheckFn decides a requirement outcome from a submission's structured
// fields. Returning OutcomeUnknown means the fields did not carry enough
// information to decide, which scores as a failure but is surfaced separately.
type autoCheckFn func(StructuredFields) Outcome

// autoChecks is the registry of mechanical checks referenced by checklists via
// the requirement `auto_check` key.
var autoChecks = map[string]autoCheckFn{
	"data_region_present": func(f StructuredFields) Outcome {
		return boolToOutcome(strings.TrimSpace(f.DataRegion) != "")
	},
	"data_region_approved": func(f StructuredFields) Outcome {
		if strings.TrimSpace(f.DataRegion) == "" || len(f.ApprovedRegions) == 0 {
			return OutcomeUnknown
		}
		return boolToOutcome(containsFold(f.ApprovedRegions, f.DataRegion))
	},
	"pii_flag_declared": func(f StructuredFields) Outcome {
		return boolToOutcome(f.ProcessesPII != nil)
	},
	"pii_not_unmasked": func(f StructuredFields) Outcome {
		if f.ProcessesPII == nil {
			return OutcomeUnknown
		}
		if !*f.ProcessesPII {
			return OutcomePass
		}
		// PII is processed; pass only if it is masked before send.
		if f.PIIMaskedBeforeSend == nil {
			return OutcomeUnknown
		}
		return boolToOutcome(*f.PIIMaskedBeforeSend)
	},
	"model_family_known": func(f StructuredFields) Outcome {
		return boolToOutcome(strings.TrimSpace(f.ModelFamily) != "")
	},
	"retention_declared": func(f StructuredFields) Outcome {
		return boolToOutcome(f.RetentionDays != nil)
	},
	"retention_bounded": func(f StructuredFields) Outcome {
		if f.RetentionDays == nil {
			return OutcomeUnknown
		}
		return boolToOutcome(*f.RetentionDays <= 90)
	},
	"encryption_in_transit": func(f StructuredFields) Outcome {
		if f.EncryptionInTransit == nil {
			return OutcomeUnknown
		}
		return boolToOutcome(*f.EncryptionInTransit)
	},
	"subprocessors_listed": func(f StructuredFields) Outcome {
		return boolToOutcome(len(f.Subprocessors) > 0)
	},
	"no_customer_training": func(f StructuredFields) Outcome {
		if f.TrainsOnCustomerData == nil {
			return OutcomeUnknown
		}
		return boolToOutcome(!*f.TrainsOnCustomerData)
	},
	"vendor_named": func(f StructuredFields) Outcome {
		// vendor_named is checked against the submission vendor at evaluation
		// time; the registry entry is a placeholder that the evaluator overrides.
		return OutcomeUnknown
	},
}

func boolToOutcome(ok bool) Outcome {
	if ok {
		return OutcomePass
	}
	return OutcomeFail
}

func containsFold(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

// KnownAutoChecks returns the set of registered auto-check names.
func KnownAutoChecks() []string {
	out := make([]string, 0, len(autoChecks))
	for k := range autoChecks {
		out = append(out, k)
	}
	return out
}
