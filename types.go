package verifex

// ── Screening Types ─────────────────────────────────────────────────────────

// V3Explainability is the feature-engineering explainability output for a match.
type V3Explainability struct {
	HumanReadable         string                     `json:"human_readable"`
	CalibratedConfidence  float64                    `json:"calibrated_confidence"`
	FeatureContributions  []FeatureContribution      `json:"feature_contributions"`
	ThresholdReasoning    ThresholdReasoning         `json:"threshold_reasoning"`
	Modifiers             []Modifier                 `json:"modifiers"`
	Calibration           CalibrationInfo            `json:"calibration"`
	EvidenceSummary       EvidenceSummary            `json:"evidence_summary"`
}

// FeatureContribution is a per-field evidence breakdown.
type FeatureContribution struct {
	Field               string  `json:"field"`
	Level               string  `json:"level"`
	ContributionPercent float64 `json:"contribution_percent"`
	Description         string  `json:"description"`
}

// ThresholdReasoning is the decision band and rationale.
type ThresholdReasoning struct {
	Decision    string `json:"decision"`
	Explanation string `json:"explanation"`
}

// Modifier is a bonus or penalty applied to the score.
type Modifier struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CalibrationInfo contains raw vs calibrated confidence.
type CalibrationInfo struct {
	Temperature          float64 `json:"temperature"`
	RawConfidence        float64 `json:"raw_confidence"`
	CalibratedConfidence float64 `json:"calibrated_confidence"`
}

// EvidenceSummary is a human-readable per-field evidence description.
type EvidenceSummary struct {
	Name        string `json:"name"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
	EntityType  string `json:"entity_type"`
	Identifier  string `json:"identifier"`
}

// Match represents a single sanctions list match.
type Match struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Aliases          []string           `json:"aliases"`
	Source           string             `json:"source"`
	EntityType       string             `json:"entity_type"`
	Nationality      *string            `json:"nationality"`
	DateOfBirth      *string            `json:"date_of_birth"`
	Reason           *string            `json:"reason"`
	Confidence       int                `json:"confidence"`
	RiskLevel        string             `json:"risk_level"`
	MatchType        string             `json:"match_type"`
	V3Explainability *V3Explainability  `json:"v3_explainability,omitempty"`
}

// ScreenResult is the response from screening a single entity.
type ScreenResult struct {
	Query        ScreenRequest `json:"query"`
	Matches      []Match       `json:"matches"`
	TotalMatches int           `json:"total_matches"`
	RiskLevel    string        `json:"risk_level"`
	ScreenedAt   string        `json:"screened_at"`
	RequestID    string        `json:"request_id"`
	ListsChecked []string      `json:"lists_checked"`
	APIVersion   string        `json:"api_version"`

	// CoverageStatus is "complete" or "partial", or "" when the API did not
	// state coverage. Empty is NOT "complete" — it means the question was
	// never answered.
	CoverageStatus string `json:"coverage_status,omitempty"`

	// UnavailableSources names the sources that could not be screened.
	UnavailableSources []string `json:"unavailable_sources,omitempty"`

	// CoverageNotice names which categories were not screened.
	CoverageNotice string `json:"coverage_notice,omitempty"`

	// ScreeningUnavailable is true when the screen produced no usable verdict.
	ScreeningUnavailable bool `json:"screening_unavailable,omitempty"`

	// RestrictedMatches counts matches found but withheld because the plan does
	// not include their source.
	RestrictedMatches int `json:"restricted_matches,omitempty"`
	// ScreeningMode is the contract this screen ran under. "exact_only"
	// matches literal identity and nothing else.
	ScreeningMode string `json:"screening_mode,omitempty"`
	// ClearanceEligible is the engine's own answer to "may this be treated as
	// a clearance?". A pointer so a MISSING field is distinguishable from an
	// explicit false — defaulting a missing flag to eligible is how this
	// became a false clear the first time.
	ClearanceEligible *bool `json:"clearance_eligible,omitempty"`
	// ExactOnlyWithheldCandidates counts candidates the mode withheld.
	ExactOnlyWithheldCandidates int `json:"exact_only_withheld_candidates,omitempty"`

	// RestrictedSources names sources whose matches were withheld.
	RestrictedSources []string `json:"restricted_sources,omitempty"`

	// SourcesExcludedByPlan names categories never queried for plan reasons.
	SourcesExcludedByPlan []string `json:"sources_excluded_by_plan,omitempty"`

	// ClearScope is "checked_sources_only" when a no-hit is scoped to the
	// checked sources rather than universal.
	ClearScope string `json:"clear_scope,omitempty"`

	// PlanScopeNotice explains plan-scoped coverage in prose.
	PlanScopeNotice string `json:"plan_scope_notice,omitempty"`

	// UpgradeHint is the upgrade path when the plan limited this screen.
	UpgradeHint string `json:"upgrade_hint,omitempty"`
}

// ClearBlockers lists every reason this result may not be treated as a clear.
// Empty means clear.
//
// Returned as a slice rather than a bool so a caller can say WHY a screen did
// not clear — "coverage_incomplete" and "restricted_matches" call for very
// different follow-up, and collapsing them into false loses the only
// information that makes the result actionable.
func (r *ScreenResult) ClearBlockers() []string {
	reasons := []string{}
	if r.RiskLevel != "clear" {
		reasons = append(reasons, "matches_found")
	}
	if r.ScreeningUnavailable {
		reasons = append(reasons, "screening_unavailable")
	}
	if !r.HasCompleteCoverage() {
		reasons = append(reasons, "coverage_incomplete")
	}
	// Matches EXIST; the plan hid them. The most misleading case of all,
	// because TotalMatches can read 0 while a sanctions hit sits behind an
	// entitlement.
	if r.RestrictedMatches > 0 {
		reasons = append(reasons, "restricted_matches")
	}
	// A no-hit scoped to the checked sources is not a universal clear.
	if r.ClearScope == "checked_sources_only" {
		reasons = append(reasons, "plan_scoped_clear")
	}
	// Exact-Only found nothing that matched LITERALLY. The mode does not check
	// spelling variants, transliterations, aliases or typos, so a clear here
	// would be a clearance the mode cannot earn.
	if (r.ClearanceEligible != nil && !*r.ClearanceEligible) || r.ScreeningMode == "exact_only" {
		reasons = append(reasons, "exact_only_not_a_clearance")
	}
	return reasons
}

// HasCompleteCoverage reports whether the API explicitly stated that every
// in-plan source was screened. Absent coverage is not complete: it means the
// screen did not report whether every source was reachable, and there is no
// safe way to assume it was.
func (r *ScreenResult) HasCompleteCoverage() bool {
	return r.CoverageStatus == "complete"
}

// IsClear reports whether nothing matched, everything was screened, and
// nothing was withheld.
//
// A "clear" risk level on its own only means "nothing was found in the sources
// we managed to search". If a sanctions list was unreachable, that is not the
// same as "this party is not sanctioned", and treating it as such is a false
// clear — the most dangerous defect a screening product can ship.
//
// Deliberately fail-closed: unknown coverage returns false. For the previous
// behaviour use IsRiskClear, whose name says what it actually checks.
func (r *ScreenResult) IsClear() bool {
	return len(r.ClearBlockers()) == 0
}

// IsRiskClear reports whether the engine found no matches, IGNORING source
// coverage. Only meaningful alongside CoverageStatus; on its own it cannot
// support a compliance decision.
func (r *ScreenResult) IsRiskClear() bool {
	return r.RiskLevel == "clear"
}

// IsMatch returns true if at least one match was found.
func (r *ScreenResult) IsMatch() bool {
	return r.TotalMatches > 0
}

// HighestConfidence returns the highest confidence score among matches, or 0.
func (r *ScreenResult) HighestConfidence() int {
	max := 0
	for _, m := range r.Matches {
		if m.Confidence > max {
			max = m.Confidence
		}
	}
	return max
}

// BatchScreenResult is the response from batch screening.
type BatchScreenResult struct {
	Results         []ScreenResult `json:"results"`
	TotalDurationMs int            `json:"total_duration_ms"`
}

// ── Usage Types ─────────────────────────────────────────────────────────────

// UsageStats contains API usage statistics.
type UsageStats struct {
	Plan             string          `json:"plan"`
	MonthlyQuota     int             `json:"monthly_quota"`
	CurrentMonthUsage int            `json:"current_month_usage"`
	Remaining        int             `json:"remaining"`
	DailyBreakdown   []DailyUsage    `json:"daily_breakdown"`
	Period           UsagePeriod     `json:"period"`
}

// DailyUsage is usage for a single day.
type DailyUsage struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// UsagePeriod is the current billing period.
type UsagePeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ── API Key Types ───────────────────────────────────────────────────────────

// APIKeyInfo is metadata about an API key.
type APIKeyInfo struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	IsActive   bool    `json:"is_active"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
	UsageCount int     `json:"usage_count"`
}

// APIKeyCreated is returned when a new key is created.
type APIKeyCreated struct {
	Key    string `json:"key"`
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
}

// ── Health Types ────────────────────────────────────────────────────────────

// HealthResponse is the API health status.
type HealthResponse struct {
	Status   string                  `json:"status"`
	Version  string                  `json:"version"`
	Uptime   int                     `json:"uptime"`
	Database string                  `json:"database"`
	Redis    string                  `json:"redis"`
	Lists    map[string]ListInfo     `json:"lists"`
}

// ListInfo is per-source sanctions list metadata.
type ListInfo struct {
	Count      int     `json:"count"`
	LastSynced *string `json:"last_synced"`
}

// IsHealthy returns true if the API is fully operational.
func (h *HealthResponse) IsHealthy() bool {
	return h.Status == "ok"
}

// TotalEntities returns the sum of entities across all lists.
func (h *HealthResponse) TotalEntities() int {
	total := 0
	for _, l := range h.Lists {
		total += l.Count
	}
	return total
}
