package entity

type CampaignEntity struct {
	ID string `json:"id"`
	CampaignName string `json:"campaign_name"`
	CampaignStartDate string `json:"campaign_start_date"`
	DailyBudget string `json:"daily_budget"`
	MatchType string `json:"match_type"`
	Bid string `json:"bid"`
	SKU string `json:"sku"`
	TotalKeywords uint `json:"total_keywords"`
	Keywords []string `json:"keywords"`

	NegativeMatchType string `json:"negative_match_type"`
	NegativeKeywords []string `json:"negative_keywords"`
}
