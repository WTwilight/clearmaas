package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ============================================================================
// Filter
// ============================================================================

type SupplierStatsFilter struct {
	StartTimestamp int64
	EndTimestamp  int64
	SupplierId    int
	ChannelId     int
	ModelName     string
}

func buildTimeRange(startTs, endTs int64) (int64, int64) {
	if startTs == 0 {
		startTs = time.Now().AddDate(0, 0, -30).Unix()
	}
	if endTs == 0 {
		endTs = time.Now().Unix()
	}
	return startTs, endTs
}

// ============================================================================
// Shared helpers
// ============================================================================

// resolveSupplierNamesBySheetIds resolves supplier names given sheet IDs.
func resolveSupplierNamesBySheetIds(sheetIds []int) map[int]string {
	result := make(map[int]string)
	if len(sheetIds) == 0 {
		return result
	}
	var sheets []struct {
		Id         int `gorm:"column:id"`
		SupplierId int `gorm:"column:supplier_id"`
	}
	if err := model.LOG_DB.Table("supplier_pricing_sheets").
		Select("id, supplier_id").
		Where("id IN ?", sheetIds).
		Find(&sheets).Error; err != nil {
		return result
	}
	sheetToSupplierId := make(map[int]int)
	supplierIds := make([]int, 0)
	seen := make(map[int]bool)
	for _, s := range sheets {
		sheetToSupplierId[s.Id] = s.SupplierId
		if !seen[s.SupplierId] {
			supplierIds = append(supplierIds, s.SupplierId)
			seen[s.SupplierId] = true
		}
	}
	var suppliers []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := model.LOG_DB.Table("suppliers").
		Select("id, name").
		Where("id IN ?", supplierIds).
		Find(&suppliers).Error; err != nil {
		return result
	}
	supplierIdToName := make(map[int]string)
	for _, sup := range suppliers {
		supplierIdToName[sup.Id] = sup.Name
	}
	for sheetId, supId := range sheetToSupplierId {
		if name, ok := supplierIdToName[supId]; ok {
			result[sheetId] = name
		}
	}
	return result
}

// resolveChannelNames resolves channel names given channel IDs.
func resolveChannelNames(channelIds []int) map[int]string {
	result := make(map[int]string)
	if len(channelIds) == 0 {
		return result
	}
	var channels []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := model.LOG_DB.Table("channels").
		Select("id, name").
		Where("id IN ?", channelIds).
		Find(&channels).Error; err == nil {
		for _, ch := range channels {
			result[ch.Id] = ch.Name
		}
	}
	return result
}

// resolveSheetToSupplier resolves (sheetId -> supplierId, supplierName)
func resolveSheetToSupplier(sheetIds []int) (map[int]int, map[int]string) {
	sheetToSupplier := make(map[int]int)
	supplierNames := make(map[int]string)
	if len(sheetIds) == 0 {
		return sheetToSupplier, supplierNames
	}
	var sheets []struct {
		Id         int `gorm:"column:id"`
		SupplierId int `gorm:"column:supplier_id"`
	}
	if err := model.LOG_DB.Table("supplier_pricing_sheets").
		Select("id, supplier_id").
		Where("id IN ?", sheetIds).
		Find(&sheets).Error; err != nil {
		return sheetToSupplier, supplierNames
	}
	supplierIds := make([]int, 0)
	seen := make(map[int]bool)
	for _, s := range sheets {
		sheetToSupplier[s.Id] = s.SupplierId
		if !seen[s.SupplierId] {
			supplierIds = append(supplierIds, s.SupplierId)
			seen[s.SupplierId] = true
		}
	}
	var suppliers []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := model.LOG_DB.Table("suppliers").
		Select("id, name").
		Where("id IN ?", supplierIds).
		Find(&suppliers).Error; err != nil {
		return sheetToSupplier, supplierNames
	}
	for _, sup := range suppliers {
		supplierNames[sup.Id] = sup.Name
	}
	return sheetToSupplier, supplierNames
}

func computeProfit(totalCharge, totalCost float64) (grossProfit, profitMargin float64) {
	grossProfit = totalCharge - totalCost
	if totalCharge > 0 {
		profitMargin = grossProfit / totalCharge * 100
	}
	return grossProfit, profitMargin
}

// ============================================================================
// SQL dialect helpers
// ============================================================================

// dialect returns the current SQL dialect: "sqlite", "mysql", or "postgresql".
func dialect() string {
	if common.UsingSQLite {
		return "sqlite"
	}
	if common.UsingPostgreSQL {
		return "postgresql"
	}
	return "mysql"
}

// jsonExtractCol returns the JSON extraction expression for l.other->key.
func jsonExtractCol(key string) string {
	if common.UsingPostgreSQL {
		return fmt.Sprintf("(l.other::json->>'%s')", key)
	}
	return fmt.Sprintf("json_extract(l.other, '$.%s')", key)
}

// castToInt returns the appropriate integer cast expression.
func castToInt(expr string) string {
	d := dialect()
	if d == "postgresql" {
		return expr + "::int"
	}
	if d == "sqlite" {
		return fmt.Sprintf("CAST(%s AS INTEGER)", expr)
	}
	return fmt.Sprintf("CAST(%s AS SIGNED)", expr)
}

// castToFloat returns the appropriate real/decimal cast expression.
func castToFloat(expr string) string {
	d := dialect()
	if d == "postgresql" {
		return expr + "::numeric"
	}
	if d == "sqlite" {
		return fmt.Sprintf("CAST(%s AS REAL)", expr)
	}
	return fmt.Sprintf("CAST(%s AS DECIMAL(20,10))", expr)
}

// costExpr returns the CASE expression for computing total_cost from Other JSON.
//
// Cost calculation by type:
//   - ratio:       original_price × supplier_cost  (supplier_cost is a multiplier, result already in quota units)
//   - fixed_price: supplier_cost  (already in quota units)
//   - per_call:    supplier_cost × QuotaPerUnit  (supplier_cost is in USD, convert to quota units)
func costExpr() string {
	costType := jsonExtractCol("supplier_cost_type")
	costVal := jsonExtractCol("supplier_cost")
	originalPrice := jsonExtractCol("original_price")
	groupRatio := jsonExtractCol("group_ratio")
	quotaPerUnit := "500000.0" // QuotaPerUnit constant in Go side

	ratioExpr := fmt.Sprintf(
		"%s * %s",
		// Fallback: original_price may be missing when model_price=-1 or in old logs.
		// We derive it as: quota × group_ratio (because quota = original_price × group_ratio).
		fmt.Sprintf("COALESCE(%s, %s * %s)",
			castToFloat(originalPrice),
			castToFloat("l.quota"),
			castToFloat(groupRatio)),
		castToFloat(costVal),
	)
	fixedExpr := castToFloat(costVal)
	perCallExpr := fmt.Sprintf("%s * %s", castToFloat(costVal), quotaPerUnit)

	if common.UsingSQLite {
		return fmt.Sprintf(`
			CASE %s
				WHEN 'ratio'       THEN %s
				WHEN 'fixed_price' THEN %s
				WHEN 'per_call'    THEN %s
				ELSE %s
			END`, costType, ratioExpr, fixedExpr, perCallExpr, fixedExpr)
	}
	if common.UsingPostgreSQL {
		return fmt.Sprintf(`
			CASE %s
				WHEN 'ratio'       THEN %s
				WHEN 'fixed_price' THEN %s
				WHEN 'per_call'    THEN %s
				ELSE %s
			END`, costType, ratioExpr, fixedExpr, perCallExpr, fixedExpr)
	}
	return fmt.Sprintf(`
		CASE CAST(%s AS CHAR)
			WHEN 'ratio'       THEN %s
			WHEN 'fixed_price' THEN %s
			WHEN 'per_call'    THEN %s
			ELSE %s
		END`, costType, ratioExpr, fixedExpr, perCallExpr, fixedExpr)
}

func sumCostExpr() string {
	return fmt.Sprintf("SUM(%s) AS total_cost", costExpr())
}

// supplierSheetIdCol returns the supplier_sheet_id column expression (as int).
func supplierSheetIdCol() string {
	return castToInt(jsonExtractCol("supplier_sheet_id"))
}

// supplierCostNotNullFilter returns the IS NOT NULL filter for supplier_cost.
func supplierCostNotNullFilter() string {
	if common.UsingSQLite {
		return fmt.Sprintf("%s IS NOT NULL AND %s > 0", castToFloat(jsonExtractCol("supplier_cost")), castToFloat(jsonExtractCol("supplier_cost")))
	}
	if common.UsingPostgreSQL {
		return fmt.Sprintf("%s IS NOT NULL AND %s > 0", jsonExtractCol("supplier_cost"), castToFloat(jsonExtractCol("supplier_cost")))
	}
	return fmt.Sprintf("json_extract(l.other, '$.supplier_cost') IS NOT NULL AND %s > 0", castToFloat(jsonExtractCol("supplier_cost")))
}

// ============================================================================
// API 0: Overview — grand totals across all supplier spend
// ============================================================================

type SupplierStatsOverviewResult struct {
	TotalCharge   float64 `json:"total_charge"`
	TotalCost     float64 `json:"total_cost"`
	GrossProfit   float64 `json:"gross_profit"`
	ProfitMargin  float64 `json:"profit_margin"`
	RequestCount  int64   `json:"request_count"`
	SupplierCount int     `json:"supplier_count"`
	ChannelCount  int     `json:"channel_count"`
}

func GetSupplierStatsOverview(filter SupplierStatsFilter) (*SupplierStatsOverviewResult, error) {
	startTs, endTs := buildTimeRange(filter.StartTimestamp, filter.EndTimestamp)
	cost := sumCostExpr()
	supplierCostFilter := supplierCostNotNullFilter()

	sqlStr := fmt.Sprintf(`
		SELECT
			SUM(l.quota) AS total_charge,
			%s,
			COUNT(*) AS request_count
		FROM logs l
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
	`, cost, supplierCostFilter)

	var row struct {
		TotalCharge  float64
		TotalCost    float64
		RequestCount int64
	}
	if err := model.LOG_DB.Raw(sqlStr, model.LogTypeConsume, startTs, endTs).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("failed to query supplier stats overview: %w", err)
	}

	var supplierCount, channelCount int64
	ssidCol := supplierSheetIdCol()

	// Count distinct suppliers that have at least one log entry with valid
	// supplier cost in the time range. This counts only suppliers that were
	// actually used, matching the same time filter as the other metrics.
	supplierCountSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT ss.supplier_id) AS cnt
		FROM logs l
		JOIN supplier_pricing_sheets ss ON ss.id = %s
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
	`, ssidCol, supplierCostFilter)
	if err := model.LOG_DB.Raw(supplierCountSQL, model.LogTypeConsume, startTs, endTs).Scan(&supplierCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count suppliers: %w", err)
	}

	// Count distinct channel IDs that have at least one log entry with valid
	// supplier cost in the time range.
	channelCountSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT l.channel_id) AS cnt
		FROM logs l
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
	`, supplierCostFilter)
	if err := model.LOG_DB.Raw(channelCountSQL, model.LogTypeConsume, startTs, endTs).Scan(&channelCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count channels: %w", err)
	}

	gp, pm := computeProfit(row.TotalCharge, row.TotalCost)
	return &SupplierStatsOverviewResult{
		TotalCharge:   row.TotalCharge,
		TotalCost:     row.TotalCost,
		GrossProfit:   gp,
		ProfitMargin:  pm,
		RequestCount:  row.RequestCount,
		SupplierCount: int(supplierCount),
		ChannelCount:  int(channelCount),
	}, nil
}

// ============================================================================
// API 1: By Supplier — one row per supplier
// ============================================================================

type SupplierStatsBySupplierItem struct {
	SupplierId   int     `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	TotalCharge  float64 `json:"total_charge"`
	TotalCost    float64 `json:"total_cost"`
	GrossProfit  float64 `json:"gross_profit"`
	ProfitMargin float64 `json:"profit_margin"`
	RequestCount int64   `json:"request_count"`
}

type SupplierStatsBySupplierResult struct {
	Items       []SupplierStatsBySupplierItem `json:"items"`
	TotalCharge float64                     `json:"total_charge"`
	TotalCost   float64                     `json:"total_cost"`
	TotalProfit float64                     `json:"total_profit"`
}

func GetSupplierStatsBySupplier(filter SupplierStatsFilter) (*SupplierStatsBySupplierResult, error) {
	startTs, endTs := buildTimeRange(filter.StartTimestamp, filter.EndTimestamp)
	cost := sumCostExpr()
	supplierCostFilter := supplierCostNotNullFilter()
	ssidCol := supplierSheetIdCol()

	sqlStr := fmt.Sprintf(`
		SELECT
			%s AS supplier_sheet_id,
			SUM(l.quota) AS total_charge,
			%s,
			COUNT(*) AS request_count
		FROM logs l
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
		GROUP BY supplier_sheet_id
		ORDER BY total_charge DESC
	`, ssidCol, cost, supplierCostFilter)

	var rows []struct {
		SupplierSheetId *int
		TotalCharge     float64
		TotalCost       float64
		RequestCount    int64
	}
	if err := model.LOG_DB.Raw(sqlStr, model.LogTypeConsume, startTs, endTs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query supplier stats by supplier: %w", err)
	}

	sheetIds := make([]int, 0)
	for _, r := range rows {
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 {
			sheetIds = append(sheetIds, *r.SupplierSheetId)
		}
	}
	sheetToSupplier, supplierNames := resolveSheetToSupplier(sheetIds)

	supplierAgg := make(map[int]struct {
		Name         string
		TotalCharge  float64
		TotalCost    float64
		RequestCount int64
	})
	for _, r := range rows {
		if r.SupplierSheetId == nil || *r.SupplierSheetId <= 0 {
			continue
		}
		sid := sheetToSupplier[*r.SupplierSheetId]
		agg := supplierAgg[sid]
		agg.Name = supplierNames[sid]
		agg.TotalCharge += r.TotalCharge
		agg.TotalCost += r.TotalCost
		agg.RequestCount += r.RequestCount
		supplierAgg[sid] = agg
	}

	items := make([]SupplierStatsBySupplierItem, 0, len(supplierAgg))
	var totalCharge, totalCost float64
	for sid, agg := range supplierAgg {
		gp, pm := computeProfit(agg.TotalCharge, agg.TotalCost)
		items = append(items, SupplierStatsBySupplierItem{
			SupplierId:   sid,
			SupplierName: agg.Name,
			TotalCharge:  agg.TotalCharge,
			TotalCost:    agg.TotalCost,
			GrossProfit:  gp,
			ProfitMargin: pm,
			RequestCount: agg.RequestCount,
		})
		totalCharge += agg.TotalCharge
		totalCost += agg.TotalCost
	}

	return &SupplierStatsBySupplierResult{
		Items:       items,
		TotalCharge: totalCharge,
		TotalCost:   totalCost,
		TotalProfit: totalCharge - totalCost,
	}, nil
}

// ============================================================================
// API 2: By Channel — one row per channel, with its supplier info
// ============================================================================

type SupplierStatsByChannelItem struct {
	SupplierId   int     `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	ChannelId    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	TotalCharge  float64 `json:"total_charge"`
	TotalCost    float64 `json:"total_cost"`
	GrossProfit  float64 `json:"gross_profit"`
	ProfitMargin float64 `json:"profit_margin"`
	RequestCount int64   `json:"request_count"`
}

type SupplierStatsByChannelResult struct {
	Items       []SupplierStatsByChannelItem `json:"items"`
	TotalCharge float64                     `json:"total_charge"`
	TotalCost   float64                     `json:"total_cost"`
	TotalProfit float64                     `json:"total_profit"`
}

func GetSupplierStatsByChannel(filter SupplierStatsFilter) (*SupplierStatsByChannelResult, error) {
	startTs, endTs := buildTimeRange(filter.StartTimestamp, filter.EndTimestamp)
	cost := sumCostExpr()
	supplierCostFilter := supplierCostNotNullFilter()
	ssidCol := supplierSheetIdCol()

	sqlStr := fmt.Sprintf(`
		SELECT
			%s AS supplier_sheet_id,
			l.channel_id,
			SUM(l.quota) AS total_charge,
			%s,
			COUNT(*) AS request_count
		FROM logs l
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
		GROUP BY supplier_sheet_id, l.channel_id
		ORDER BY total_charge DESC
	`, ssidCol, cost, supplierCostFilter)

	var rows []struct {
		SupplierSheetId *int
		ChannelId       int
		TotalCharge     float64
		TotalCost       float64
		RequestCount    int64
	}
	if err := model.LOG_DB.Raw(sqlStr, model.LogTypeConsume, startTs, endTs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query supplier stats by channel: %w", err)
	}

	sheetIds := make([]int, 0)
	channelIds := make([]int, 0)
	seenSheet := make(map[int]bool)
	seenChannel := make(map[int]bool)
	for _, r := range rows {
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 && !seenSheet[*r.SupplierSheetId] {
			sheetIds = append(sheetIds, *r.SupplierSheetId)
			seenSheet[*r.SupplierSheetId] = true
		}
		if r.ChannelId > 0 && !seenChannel[r.ChannelId] {
			channelIds = append(channelIds, r.ChannelId)
			seenChannel[r.ChannelId] = true
		}
	}
	sheetToSupplier, supplierNames := resolveSheetToSupplier(sheetIds)
	channelNames := resolveChannelNames(channelIds)

	items := make([]SupplierStatsByChannelItem, 0, len(rows))
	var totalCharge, totalCost float64
	for _, r := range rows {
		gp, pm := computeProfit(r.TotalCharge, r.TotalCost)
		sid := 0
		sname := ""
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 {
			sid = sheetToSupplier[*r.SupplierSheetId]
			sname = supplierNames[sid]
		}
		items = append(items, SupplierStatsByChannelItem{
			SupplierId:   sid,
			SupplierName: sname,
			ChannelId:    r.ChannelId,
			ChannelName:  channelNames[r.ChannelId],
			TotalCharge:  r.TotalCharge,
			TotalCost:    r.TotalCost,
			GrossProfit:  gp,
			ProfitMargin: pm,
			RequestCount: r.RequestCount,
		})
		totalCharge += r.TotalCharge
		totalCost += r.TotalCost
	}

	return &SupplierStatsByChannelResult{
		Items:       items,
		TotalCharge: totalCharge,
		TotalCost:   totalCost,
		TotalProfit: totalCharge - totalCost,
	}, nil
}

// ============================================================================
// API 3: By Model — one row per (supplier, channel, model, date)
// ============================================================================

type SupplierStatsByModelItem struct {
	SupplierId   int     `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	ChannelId    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	ModelName    string  `json:"model_name"`
	Date         string  `json:"date"`
	TotalCharge  float64 `json:"total_charge"`
	TotalCost    float64 `json:"total_cost"`
	GrossProfit  float64 `json:"gross_profit"`
	ProfitMargin float64 `json:"profit_margin"`
	RequestCount int64   `json:"request_count"`
}

type SupplierStatsByModelResult struct {
	Items       []SupplierStatsByModelItem `json:"items"`
	TotalCharge float64                   `json:"total_charge"`
	TotalCost   float64                   `json:"total_cost"`
	TotalProfit float64                   `json:"total_profit"`
}

func GetSupplierStatsByModel(filter SupplierStatsFilter) (*SupplierStatsByModelResult, error) {
	startTs, endTs := buildTimeRange(filter.StartTimestamp, filter.EndTimestamp)
	cost := sumCostExpr()
	supplierCostFilter := supplierCostNotNullFilter()
	ssidCol := supplierSheetIdCol()

	var sqlStr string
	if common.UsingSQLite {
		sqlStr = fmt.Sprintf(`
			SELECT
				%s AS supplier_sheet_id,
				l.channel_id,
				l.model_name,
				DATE(l.created_at, 'unixepoch') AS date,
				SUM(l.quota) AS total_charge,
				%s,
				COUNT(*) AS request_count
			FROM logs l
			WHERE l.type = ?
				AND l.created_at >= ?
				AND l.created_at <= ?
				AND %s
			GROUP BY supplier_sheet_id, l.channel_id, l.model_name, date
			ORDER BY date DESC, total_charge DESC
		`, ssidCol, cost, supplierCostFilter)
	} else if common.UsingPostgreSQL {
		sqlStr = fmt.Sprintf(`
			SELECT
				%s AS supplier_sheet_id,
				l.channel_id,
				l.model_name,
				DATE(TO_TIMESTAMP(l.created_at)) AS date,
				SUM(l.quota) AS total_charge,
				%s,
				COUNT(*) AS request_count
			FROM logs l
			WHERE l.type = ?
				AND l.created_at >= ?
				AND l.created_at <= ?
				AND %s
			GROUP BY supplier_sheet_id, l.channel_id, l.model_name, date
			ORDER BY date DESC, total_charge DESC
		`, ssidCol, cost, supplierCostFilter)
	} else {
		sqlStr = fmt.Sprintf(`
			SELECT
				%s AS supplier_sheet_id,
				l.channel_id,
				l.model_name,
				DATE(FROM_UNIXTIME(l.created_at)) AS date,
				SUM(l.quota) AS total_charge,
				%s,
				COUNT(*) AS request_count
			FROM logs l
			WHERE l.type = ?
				AND l.created_at >= ?
				AND l.created_at <= ?
				AND %s
			GROUP BY supplier_sheet_id, l.channel_id, l.model_name, date
			ORDER BY date DESC, total_charge DESC
		`, ssidCol, cost, supplierCostFilter)
	}

	var rows []struct {
		SupplierSheetId *int
		ChannelId       int
		ModelName       string
		Date            string
		TotalCharge     float64
		TotalCost       float64
		RequestCount    int64
	}
	if err := model.LOG_DB.Raw(sqlStr, model.LogTypeConsume, startTs, endTs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query supplier stats by model: %w", err)
	}

	sheetIds := make([]int, 0)
	channelIds := make([]int, 0)
	seenSheet := make(map[int]bool)
	seenChannel := make(map[int]bool)
	for _, r := range rows {
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 && !seenSheet[*r.SupplierSheetId] {
			sheetIds = append(sheetIds, *r.SupplierSheetId)
			seenSheet[*r.SupplierSheetId] = true
		}
		if r.ChannelId > 0 && !seenChannel[r.ChannelId] {
			channelIds = append(channelIds, r.ChannelId)
			seenChannel[r.ChannelId] = true
		}
	}
	sheetToSupplier, supplierNames := resolveSheetToSupplier(sheetIds)
	channelNames := resolveChannelNames(channelIds)

	items := make([]SupplierStatsByModelItem, 0, len(rows))
	var totalCharge, totalCost float64
	for _, r := range rows {
		gp, pm := computeProfit(r.TotalCharge, r.TotalCost)
		sid := 0
		sname := ""
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 {
			sid = sheetToSupplier[*r.SupplierSheetId]
			sname = supplierNames[sid]
		}
		items = append(items, SupplierStatsByModelItem{
			SupplierId:   sid,
			SupplierName:  sname,
			ChannelId:     r.ChannelId,
			ChannelName:   channelNames[r.ChannelId],
			ModelName:     r.ModelName,
			Date:          r.Date,
			TotalCharge:   r.TotalCharge,
			TotalCost:     r.TotalCost,
			GrossProfit:   gp,
			ProfitMargin:  pm,
			RequestCount:  r.RequestCount,
		})
		totalCharge += r.TotalCharge
		totalCost += r.TotalCost
	}

	return &SupplierStatsByModelResult{
		Items:       items,
		TotalCharge: totalCharge,
		TotalCost:   totalCost,
		TotalProfit: totalCharge - totalCost,
	}, nil
}

// ============================================================================
// Legacy: detail-level (by date) — kept for backward compatibility
// ============================================================================

type SupplierStatItem struct {
	ChannelId      int     `json:"channel_id"`
	ChannelName    string  `json:"channel_name"`
	ModelName      string  `json:"model_name"`
	TotalCharge    float64 `json:"total_charge"`
	TotalCost      float64 `json:"total_cost"`
	GrossProfit    float64 `json:"gross_profit"`
	ProfitMargin   float64 `json:"profit_margin"`
	RequestCount   int64   `json:"request_count"`
	Date           string  `json:"date"`
	SupplierName   string  `json:"supplier_name"`
	SupplierSheet  string  `json:"supplier_sheet"`
}

type SupplierStatsResult struct {
	Items       []SupplierStatItem `json:"items"`
	TotalCharge float64           `json:"total_charge"`
	TotalCost   float64           `json:"total_cost"`
	TotalProfit float64           `json:"total_profit"`
}

func getSupplierStatSQL() string {
	cost := sumCostExpr()
	supplierCostFilter := supplierCostNotNullFilter()
	ssidCol := supplierSheetIdCol()

	if common.UsingSQLite {
		return fmt.Sprintf(`
			SELECT
				l.channel_id,
				l.model_name,
				DATE(l.created_at, 'unixepoch') AS date,
				%s AS supplier_sheet_id,
				CAST(json_extract(l.other, '$.supplier_sheet_name') AS TEXT) AS supplier_sheet,
				SUM(l.quota) AS total_charge,
				%s,
				COUNT(*) AS request_count
			FROM logs l
			WHERE l.type = ?
				AND l.created_at >= ?
				AND l.created_at <= ?
				AND %s
			GROUP BY l.channel_id, l.model_name, date, supplier_sheet_id, supplier_sheet
			ORDER BY date DESC, total_charge DESC
		`, ssidCol, cost, supplierCostFilter)
	}
	if common.UsingPostgreSQL {
		return fmt.Sprintf(`
			SELECT
				l.channel_id,
				l.model_name,
				DATE(TO_TIMESTAMP(l.created_at)) AS date,
				%s AS supplier_sheet_id,
				(l.other::json->>'supplier_sheet_name') AS supplier_sheet,
				SUM(l.quota) AS total_charge,
				%s,
				COUNT(*) AS request_count
			FROM logs l
			WHERE l.type = ?
				AND l.created_at >= ?
				AND l.created_at <= ?
				AND %s
			GROUP BY l.channel_id, l.model_name, date, supplier_sheet_id, supplier_sheet
			ORDER BY date DESC, total_charge DESC
		`, ssidCol, cost, supplierCostFilter)
	}
	return fmt.Sprintf(`
		SELECT
			l.channel_id,
			l.model_name,
			DATE(FROM_UNIXTIME(l.created_at)) AS date,
			%s AS supplier_sheet_id,
			CAST(json_extract(l.other, '$.supplier_sheet_name') AS CHAR) AS supplier_sheet,
			SUM(l.quota) AS total_charge,
			%s,
			COUNT(*) AS request_count
		FROM logs l
		WHERE l.type = ?
			AND l.created_at >= ?
			AND l.created_at <= ?
			AND %s
		GROUP BY l.channel_id, l.model_name, date, supplier_sheet_id, supplier_sheet
		ORDER BY date DESC, total_charge DESC
	`, ssidCol, cost, supplierCostFilter)
}

func GetSupplierStats(filter SupplierStatsFilter) (*SupplierStatsResult, error) {
	startTs, endTs := buildTimeRange(filter.StartTimestamp, filter.EndTimestamp)
	sqlStr := getSupplierStatSQL()

	var rows []struct {
		ChannelId       int
		ModelName       string
		Date            string
		SupplierSheetId *int
		SupplierSheet   *string
		TotalCharge     float64
		TotalCost       float64
		RequestCount    int64
	}
	if err := model.LOG_DB.Raw(sqlStr, model.LogTypeConsume, startTs, endTs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query supplier stats: %w", err)
	}

	channelIds := make([]int, 0)
	sheetIds := make([]int, 0)
	seenChannels := make(map[int]bool)
	seenSheets := make(map[int]bool)
	for _, r := range rows {
		if r.ChannelId > 0 && !seenChannels[r.ChannelId] {
			channelIds = append(channelIds, r.ChannelId)
			seenChannels[r.ChannelId] = true
		}
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 && !seenSheets[*r.SupplierSheetId] {
			sheetIds = append(sheetIds, *r.SupplierSheetId)
			seenSheets[*r.SupplierSheetId] = true
		}
	}
	channelNames := resolveChannelNames(channelIds)
	sheetToSupplierName := resolveSupplierNamesBySheetIds(sheetIds)

	items := make([]SupplierStatItem, 0, len(rows))
	var totalCharge, totalCost float64
	for _, r := range rows {
		gp, pm := computeProfit(r.TotalCharge, r.TotalCost)
		supplierName := ""
		if r.SupplierSheetId != nil && *r.SupplierSheetId > 0 {
			supplierName = sheetToSupplierName[*r.SupplierSheetId]
		}
		sheetName := ""
		if r.SupplierSheet != nil {
			sheetName = *r.SupplierSheet
		}
		items = append(items, SupplierStatItem{
			ChannelId:     r.ChannelId,
			ChannelName:   channelNames[r.ChannelId],
			ModelName:     r.ModelName,
			TotalCharge:   r.TotalCharge,
			TotalCost:     r.TotalCost,
			GrossProfit:   gp,
			ProfitMargin:  pm,
			RequestCount:  r.RequestCount,
			Date:          r.Date,
			SupplierName:  supplierName,
			SupplierSheet: sheetName,
		})
		totalCharge += r.TotalCharge
		totalCost += r.TotalCost
	}

	return &SupplierStatsResult{
		Items:       items,
		TotalCharge: totalCharge,
		TotalCost:   totalCost,
		TotalProfit: totalCharge - totalCost,
	}, nil
}
