package dto

type ModelSquareResponse struct {
	Success bool            `json:"success"`
	Data    ModelSquareData `json:"data"`
}

type ModelSquareData struct {
	Models  []ModelSquareItem `json:"models"`
	Vendors []ModelVendor     `json:"vendors"`
	Tags    []string          `json:"tags"`
	Total   int               `json:"total"`
}

type ModelSquareItem struct {
	ID              int                `json:"id"`
	Name            string             `json:"name"`
	DisplayName     string             `json:"display_name"`
	Description     string             `json:"description"`
	Icon            string             `json:"icon"`
	VendorID        int                `json:"vendor_id"`
	VendorName      string             `json:"vendor_name"`
	VendorIcon      string             `json:"vendor_icon"`
	Tags            []string           `json:"tags"`
	InputTypes      []string           `json:"input_types"`
	OutputTypes     []string           `json:"output_types"`
	Parameters      []string           `json:"parameters"`
	Protocols       []string           `json:"protocols"`
	Reasoning       string             `json:"reasoning"`
	ContextTokens   int64              `json:"context_tokens"`
	MaxOutput       int64              `json:"max_output"`
	DiscountPercent int                `json:"discount_percent"`
	RequestCount    int64              `json:"request_count"`
	UpdatedTime     int64              `json:"updated_time"`
	Versions        []ModelVersion     `json:"versions"`
	VersionCount    int                `json:"version_count"`
	PriceInfo       *PriceInfo         `json:"price_info,omitempty"`
	Capabilities    *ModelCapabilities `json:"capabilities,omitempty"`
}

type ModelVersion struct {
	ModelName       string     `json:"model_name"`
	UpstreamKey     string     `json:"upstream_key"`
	ChannelID       int        `json:"channel_id"`
	ChannelName     string     `json:"channel_name"`
	ChannelTags     []string   `json:"channel_tags,omitempty"`
	ChannelType     int        `json:"channel_type"`
	Status          string     `json:"status"`
	Priority        int        `json:"priority"`
	ModeDescription string     `json:"mode_description,omitempty"`
	DiscountPercent int        `json:"discount_percent"`
	DiscountRatio   float64    `json:"discount_ratio,omitempty"`
	RatioSource     string     `json:"ratio_source,omitempty"`
	TPS             float64    `json:"tps"`
	LatencySeconds  float64    `json:"latency_seconds"`
	SuccessRate     float64    `json:"success_rate"`
	RequestCount    int64      `json:"request_count"`
	UpdatedTime     int64      `json:"updated_time,omitempty"`
	PriceInfo       *PriceInfo `json:"price_info,omitempty"`
}

type PriceInfo struct {
	InputRatio            float64 `json:"input_ratio"`
	OutputRatio           float64 `json:"output_ratio"`
	QuotaType             int     `json:"quota_type"`
	ModelPrice            float64 `json:"model_price,omitempty"`
	DiscountRatio         float64 `json:"discount_ratio,omitempty"`
	InputOriginalPrice    float64 `json:"input_original_price,omitempty"`
	OutputOriginalPrice   float64 `json:"output_original_price,omitempty"`
	InputDiscountedPrice  float64 `json:"input_discounted_price,omitempty"`
	OutputDiscountedPrice float64 `json:"output_discounted_price,omitempty"`
	RatioSource           string  `json:"ratio_source,omitempty"`
}

type ModelCapabilities struct {
	SupportedEndpoints []string `json:"supported_endpoints"`
	Streaming          *bool    `json:"streaming,omitempty"`
	Vision             *bool    `json:"vision,omitempty"`
}

type ModelVendor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}
