package verifex

// ── Screening Types ─────────────────────────────────────────────────────────

// Match represents a single sanctions list match.
type Match struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Aliases     []string `json:"aliases"`
	Source      string  `json:"source"`
	EntityType  string  `json:"entity_type"`
	Nationality *string `json:"nationality"`
	DateOfBirth *string `json:"date_of_birth"`
	Reason      *string `json:"reason"`
	Confidence  int     `json:"confidence"`
	RiskLevel   string  `json:"risk_level"`
	MatchType   string  `json:"match_type"`
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
}

// IsClear returns true if no sanctions matches were found.
func (r *ScreenResult) IsClear() bool {
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
