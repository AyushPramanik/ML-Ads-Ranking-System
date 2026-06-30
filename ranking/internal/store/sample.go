package store

// SampleAds returns a representative ad catalog spanning every category. These
// rows mirror the seed data inserted into PostgreSQL (see deploy/postgres),
// keeping in-memory and database-backed behaviour consistent.
func SampleAds() []Ad {
	return []Ad{
		{ID: 1, CampaignID: 101, Title: "Summer Sneaker Sale", Category: "retail", HistoricalCTR: 0.082, CampaignBudget: 5000, CampaignStatus: "active"},
		{ID: 2, CampaignID: 101, Title: "Designer Bags 40% Off", Category: "retail", HistoricalCTR: 0.061, CampaignBudget: 5000, CampaignStatus: "active"},
		{ID: 3, CampaignID: 102, Title: "0% APR Balance Transfer", Category: "finance", HistoricalCTR: 0.019, CampaignBudget: 12000, CampaignStatus: "active"},
		{ID: 4, CampaignID: 102, Title: "High-Yield Savings 5.2%", Category: "finance", HistoricalCTR: 0.024, CampaignBudget: 12000, CampaignStatus: "active"},
		{ID: 5, CampaignID: 103, Title: "Raid Boss: Play Free", Category: "gaming", HistoricalCTR: 0.140, CampaignBudget: 8000, CampaignStatus: "active"},
		{ID: 6, CampaignID: 103, Title: "Build Your Empire Now", Category: "gaming", HistoricalCTR: 0.118, CampaignBudget: 8000, CampaignStatus: "active"},
		{ID: 7, CampaignID: 104, Title: "Maldives Getaway Deals", Category: "travel", HistoricalCTR: 0.072, CampaignBudget: 6500, CampaignStatus: "active"},
		{ID: 8, CampaignID: 104, Title: "Cheap Flights to Tokyo", Category: "travel", HistoricalCTR: 0.066, CampaignBudget: 6500, CampaignStatus: "active"},
		{ID: 9, CampaignID: 105, Title: "30-Minute Meal Kits", Category: "food", HistoricalCTR: 0.094, CampaignBudget: 4200, CampaignStatus: "active"},
		{ID: 10, CampaignID: 105, Title: "Late-Night Pizza Deal", Category: "food", HistoricalCTR: 0.101, CampaignBudget: 4200, CampaignStatus: "active"},
		{ID: 11, CampaignID: 106, Title: "Lease an EV Today", Category: "auto", HistoricalCTR: 0.031, CampaignBudget: 15000, CampaignStatus: "active"},
		{ID: 12, CampaignID: 106, Title: "Trade In, Trade Up", Category: "auto", HistoricalCTR: 0.028, CampaignBudget: 15000, CampaignStatus: "active"},
		{ID: 13, CampaignID: 107, Title: "Flagship Phone Pre-Order", Category: "tech", HistoricalCTR: 0.058, CampaignBudget: 9000, CampaignStatus: "active"},
		{ID: 14, CampaignID: 107, Title: "Noise-Cancelling Earbuds", Category: "tech", HistoricalCTR: 0.063, CampaignBudget: 9000, CampaignStatus: "active"},
		{ID: 15, CampaignID: 108, Title: "Online Therapy, $0 First Week", Category: "health", HistoricalCTR: 0.037, CampaignBudget: 7000, CampaignStatus: "active"},
		{ID: 16, CampaignID: 108, Title: "Daily Vitamins Subscription", Category: "health", HistoricalCTR: 0.041, CampaignBudget: 7000, CampaignStatus: "active"},
		{ID: 17, CampaignID: 109, Title: "Outlet Clearance Blowout", Category: "retail", HistoricalCTR: 0.070, CampaignBudget: 3000, CampaignStatus: "active"},
		{ID: 18, CampaignID: 110, Title: "Crypto Index Fund", Category: "finance", HistoricalCTR: 0.022, CampaignBudget: 11000, CampaignStatus: "active"},
		// Paused campaign: excluded from Ads(), honoured only via explicit IDs.
		{ID: 19, CampaignID: 111, Title: "Discontinued Console Bundle", Category: "gaming", HistoricalCTR: 0.090, CampaignBudget: 2000, CampaignStatus: "paused"},
		{ID: 20, CampaignID: 112, Title: "Weekend Road Trip Rentals", Category: "travel", HistoricalCTR: 0.055, CampaignBudget: 5500, CampaignStatus: "active"},
	}
}
