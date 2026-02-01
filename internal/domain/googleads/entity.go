package googleads

// Campaign represents a Google Ads campaign
type Campaign struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	CustomerID   string  `json:"customer_id"`
	CampaignID   string  `json:"campaign_id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	BudgetAmount float64 `json:"budget_amount"`
	TargetCPA    float64 `json:"target_cpa"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// AdGroup represents a Google Ads ad group
type AdGroup struct {
	ID         string  `json:"id"`
	CampaignID string  `json:"campaign_id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	CPCBid     float64 `json:"cpc_bid"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// Keyword represents a Google Ads keyword
type Keyword struct {
	ID        string `json:"id"`
	AdGroupID string `json:"ad_group_id"`
	Text      string `json:"text"`
	MatchType string `json:"match_type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PerformanceMetrics represents campaign performance data
type PerformanceMetrics struct {
	CampaignID  string  `json:"campaign_id"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	Cost        float64 `json:"cost"`
	Conversions int64   `json:"conversions"`
	CTR         float64 `json:"ctr"`
	CPC         float64 `json:"cpc"`
	CPM         float64 `json:"cpm"`
	ConvRate    float64 `json:"conversion_rate"`
	CostPerConv float64 `json:"cost_per_conversion"`
	Date        string  `json:"date"`
}

// AccountInfo represents Google Ads account information
type AccountInfo struct {
	CustomerID   string `json:"customer_id"`
	Name         string `json:"name"`
	CurrencyCode string `json:"currency_code"`
	TimeZone     string `json:"time_zone"`
	Status       string `json:"status"`
}
