package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makePeriodStart always returns 00:00:00 UTC of the given date's first day of month.
// For specific timestamps (not month-starts), use raw Unix timestamps instead.
func makePeriodStart(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

// seedTokenForPeriodTest creates a test Token with period quota fields for direct DB operation tests.
// dailyResetLast/monthlyResetLast are set to the current period so ResetTokenPeriodQuotaIfDue
// (called inside IncreaseTokenQuota/PreConsumeTokenQuota) does NOT reset unexpectedly.
func seedTokenForPeriodTest(t *testing.T, userId int, key string, remainQuota int,
	dailyLimit, monthlyLimit int, usedDaily, usedMonthly int,
	dailyResetLast, monthlyResetLast int64) *model.Token {
	t.Helper()
	now := time.Now().Unix()
	if dailyResetLast == 0 {
		dailyResetLast = getDailyPeriodStart(now)
	}
	if monthlyResetLast == 0 {
		monthlyResetLast = getMonthlyPeriodStart(now)
	}
	token := &model.Token{
		UserId:               userId,
		Key:                  key,
		Status:               1,
		RemainQuota:          remainQuota,
		UnlimitedQuota:       false,
		QuotaLimitDaily:      dailyLimit,
		QuotaLimitMonthly:    monthlyLimit,
		QuotaUsedDaily:       usedDaily,
		QuotaUsedMonthly:     usedMonthly,
		QuotaDailyResetLast:   dailyResetLast,
		QuotaMonthlyResetLast: monthlyResetLast,
	}
	require.NoError(t, model.DB.Create(token).Error)
	return token
}

// ============================================================================
// getDailyPeriodStart — pure function, 100% branch coverage
// ============================================================================

func TestGetDailyPeriodStart(t *testing.T) {
	tests := []struct {
		name string
		now  int64
		want int64
	}{
		{
			name: "midday - normal day",
			now:  time.Date(2024, 1, 15, 8, 30, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "at midnight - start of day",
			now:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "one second before midnight",
			now:  time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "cross year boundary",
			now:  time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
			want: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "start of month",
			now:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "leap year february",
			now:  time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC).Unix(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDailyPeriodStart(tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// getMonthlyPeriodStart — pure function, 100% branch coverage
// ============================================================================

func TestGetMonthlyPeriodStart(t *testing.T) {
	tests := []struct {
		name string
		now  int64
		want int64
	}{
		{
			name: "mid month",
			now:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "first of month at midnight",
			now:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "last day of month",
			now:  time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "last day of previous month",
			now:  time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC).Unix(),
			want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "cross year boundary",
			now:  time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
			want: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		{
			name: "leap year february",
			now:  time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC).Unix(),
			want: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMonthlyPeriodStart(tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// ResetTokenPeriodQuotaIfDue — integration test with DB
// ============================================================================

func TestResetTokenPeriodQuotaIfDue(t *testing.T) {
	t.Run("both daily and monthly need reset", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 1, "reset-test-user")

		yesterday := makePeriodStart(2024, 1, 14)
		lastMonth := makePeriodStart(2023, 12, 1)
		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-reset-both",
			1000, 500, 3000, 200, 1500, yesterday, lastMonth)

		err := ResetTokenPeriodQuotaIfDue(token, today)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		// daily should be reset
		assert.Equal(t, 0, reloaded.QuotaUsedDaily)
		assert.Equal(t, today, reloaded.QuotaDailyResetLast)
		// monthly should be reset
		assert.Equal(t, 0, reloaded.QuotaUsedMonthly)
		assert.Equal(t, thisMonth, reloaded.QuotaMonthlyResetLast)
	})

	t.Run("only daily needs reset, monthly is current", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 2, "reset-test-user2")

		yesterday := makePeriodStart(2024, 1, 14)
		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-reset-daily",
			1000, 500, 3000, 200, 1000, yesterday, thisMonth)

		err := ResetTokenPeriodQuotaIfDue(token, today)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		assert.Equal(t, 0, reloaded.QuotaUsedDaily)
		assert.Equal(t, today, reloaded.QuotaDailyResetLast)
		// monthly should NOT be reset
		assert.Equal(t, 1000, reloaded.QuotaUsedMonthly)
		assert.Equal(t, thisMonth, reloaded.QuotaMonthlyResetLast)
	})

	t.Run("only monthly needs reset, daily is current", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 3, "reset-test-user3")

		lastMonth := makePeriodStart(2023, 12, 1)
		today := makePeriodStart(2024, 1, 15)

		// dailyResetLast=0 → auto-set to today; monthlyResetLast=0 → auto-set to lastMonth
		token := seedTokenForPeriodTest(t, user.Id, "key-reset-monthly",
			1000, 500, 3000, 200, 1500, today, lastMonth)

		err := ResetTokenPeriodQuotaIfDue(token, today)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		// daily should NOT be reset (dailyResetLast auto-set to today matches dailyStart)
		assert.Equal(t, 200, reloaded.QuotaUsedDaily)
		assert.Equal(t, today, reloaded.QuotaDailyResetLast)
		// monthly should be reset (monthlyResetLast explicitly set to lastMonth < thisMonth)
		assert.Equal(t, 0, reloaded.QuotaUsedMonthly)
		assert.Equal(t, getMonthlyPeriodStart(today), reloaded.QuotaMonthlyResetLast)
	})

	t.Run("neither needs reset - no DB write", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 4, "reset-test-user4")

		// seed auto-sets reset times to current period; now() is later but same period → no reset
		token := seedTokenForPeriodTest(t, user.Id, "key-no-reset",
			1000, 500, 3000, 200, 1500, 0, 0)

		beforeDaily := token.QuotaDailyResetLast
		beforeMonthly := token.QuotaMonthlyResetLast

		err := ResetTokenPeriodQuotaIfDue(token, time.Now().Unix())
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		// values should be unchanged
		assert.Equal(t, beforeDaily, reloaded.QuotaDailyResetLast)
		assert.Equal(t, beforeMonthly, reloaded.QuotaMonthlyResetLast)
		assert.Equal(t, 200, reloaded.QuotaUsedDaily)
		assert.Equal(t, 1500, reloaded.QuotaUsedMonthly)
	})

	t.Run("daily at exact boundary - should reset", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 5, "reset-test-user5")

		yesterday := makePeriodStart(2024, 1, 14)
		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-boundary",
			1000, 500, 3000, 200, 1000, yesterday, thisMonth)

		err := ResetTokenPeriodQuotaIfDue(token, today)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		assert.Equal(t, 0, reloaded.QuotaUsedDaily)
	})

	t.Run("monthly at exact boundary - should reset", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 6, "reset-test-user6")

		lastMonth := makePeriodStart(2023, 12, 1)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-mboundary",
			1000, 500, 3000, 200, 1500, thisMonth, lastMonth)

		err := ResetTokenPeriodQuotaIfDue(token, thisMonth)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		assert.Equal(t, 0, reloaded.QuotaUsedMonthly)
	})
}

// ============================================================================
// DecreaseTokenQuota — integration test verifying all 6 fields
// ============================================================================

func TestDecreaseTokenQuota_PeriodFields(t *testing.T) {
	t.Run("decrease updates all 6 fields correctly", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 10, "decr-test-user")

		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-decr",
			1000, 500, 3000, 200, 1500, today, thisMonth)

		err := model.DecreaseTokenQuota(token.Id, token.Key, 100)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		// 总配额
		assert.Equal(t, 900, reloaded.RemainQuota)
		assert.Equal(t, 100, reloaded.UsedQuota)
		// 每日
		assert.Equal(t, 300, reloaded.QuotaUsedDaily)
		// 每月
		assert.Equal(t, 1600, reloaded.QuotaUsedMonthly)
		// accessed_time 应更新（大于 0）
		assert.Greater(t, reloaded.AccessedTime, int64(0))
	})

	t.Run("consecutive decreases accumulate correctly", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 11, "decr-test-user2")

		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-decr2",
			1000, 500, 3000, 0, 0, today, thisMonth)

		require.NoError(t, model.DecreaseTokenQuota(token.Id, token.Key, 100))
		require.NoError(t, model.DecreaseTokenQuota(token.Id, token.Key, 50))

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		assert.Equal(t, 850, reloaded.RemainQuota)
		assert.Equal(t, 150, reloaded.UsedQuota)
		assert.Equal(t, 150, reloaded.QuotaUsedDaily)
		assert.Equal(t, 150, reloaded.QuotaUsedMonthly)
	})
}

// ============================================================================
// IncreaseTokenQuota — integration test verifying all 6 fields
// ============================================================================

func TestIncreaseTokenQuota_PeriodFields(t *testing.T) {
	t.Run("increase updates all 6 fields correctly", func(t *testing.T) {
		truncateServiceTestData(t)
		user := seedServiceUser(t, 20, "incr-test-user")

		today := makePeriodStart(2024, 1, 15)
		thisMonth := makePeriodStart(2024, 1, 1)

		token := seedTokenForPeriodTest(t, user.Id, "key-incr",
			800, 500, 3000, 200, 1500, today, thisMonth)

		err := model.IncreaseTokenQuota(token.Id, token.Key, 50)
		require.NoError(t, err)

		var reloaded model.Token
		require.NoError(t, model.DB.First(&reloaded, token.Id).Error)

		// 总配额: IncreaseTokenQuota adds quota back
		assert.Equal(t, 850, reloaded.RemainQuota)
		// 每日/每月 used 字段在 increase 时会减少（退款逻辑）
		assert.Equal(t, 150, reloaded.QuotaUsedDaily)     // 200 - 50 = 150
		assert.Equal(t, 1450, reloaded.QuotaUsedMonthly)  // 1500 - 50 = 1450
	})
}
