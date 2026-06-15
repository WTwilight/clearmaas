package dto

type AvailablePricingModelGroupsData struct {
	Models []AvailablePricingModelGroup `json:"models"`
	Total  int                          `json:"total"`
}

type AvailablePricingModelGroup struct {
	Name              string                         `json:"name"`
	DisplayName       string                         `json:"display_name"`
	Icon              string                         `json:"icon"`
	VendorID          int                            `json:"vendor_id,omitempty"`
	Vendor            string                         `json:"vendor"`
	VendorName        string                         `json:"vendor_name"`
	VendorIcon        string                         `json:"vendor_icon"`
	Tags              []string                       `json:"tags"`
	Context           int64                          `json:"context"`
	ContextTokens     int64                          `json:"context_tokens"`
	MaxOutput         int64                          `json:"max_output"`
	BestDiscountRatio float64                        `json:"best_discount_ratio,omitempty"`
	Versions          []AvailablePricingModelVersion `json:"versions"`
}

type AvailablePricingModelVersion struct {
	Model                 string    `json:"model"`
	UpstreamKey           string    `json:"upstream_key"`
	ChannelID             int       `json:"channel_id"`
	ChannelName           string    `json:"channel_name"`
	ChannelTags           []string  `json:"channel_tags"`
	PricingSheetID        int       `json:"pricing_sheet_id"`
	SheetID               int       `json:"sheet_id"`
	SheetName             string    `json:"sheet_name"`
	Source                string    `json:"source"`
	VendorType            string    `json:"vendor_type"`
	QuotaType             int       `json:"quota_type"`
	Original              PricePair `json:"original"`
	InputOriginalPrice    float64   `json:"input_original_price"`
	OutputOriginalPrice   float64   `json:"output_original_price"`
	Discounted            PricePair `json:"discounted"`
	DiscountRatio         float64   `json:"discount_ratio"`
	InputDiscountedPrice  float64   `json:"input_discounted_price"`
	OutputDiscountedPrice float64   `json:"output_discounted_price"`
}

type PricePair struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}
