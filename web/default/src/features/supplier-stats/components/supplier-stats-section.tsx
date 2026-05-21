/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useMemo, useState, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Activity, Coins, TrendingUp, TrendingDown, Loader2, Warehouse, GitBranch, BarChart3 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getRollingDateRange } from '@/lib/time'
import { VCHART_OPTION } from '@/lib/vchart'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertTriangle } from 'lucide-react'
import { toast } from 'sonner'
import { StaggerContainer, StaggerItem } from '@/components/page-transition'
import {
  getSupplierStatsOverview,
  getSupplierStatsBySupplier,
  getSupplierStatsByChannel,
} from '../api'
import { StatCard } from '@/features/dashboard/components/ui/stat-card'
import { processSupplierChartData } from '../lib/charts'
import type {
  SupplierStatsOverviewResult,
  SupplierStatsBySupplierItem,
  SupplierStatsByChannelItem,
  ProcessedSupplierChartData,
} from '../types'

// ============================================================================
// Time Range Presets (matching User Analytics)
// ============================================================================

const TIME_RANGE_PRESETS = [
  { label: '1 Day', days: 1 },
  { label: '7 Days', days: 7 },
  { label: '14 Days', days: 14 },
  { label: '29 Days', days: 29 },
] as const

// ============================================================================
// Chart Definitions
// ============================================================================

const SUPPLIER_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedSupplierChartData
}[] = [
  {
    value: 'supplier-rank',
    labelKey: 'Supplier Consumption Ranking',
    specKey: 'spec_supplier_rank',
  },
  {
    value: 'channel-rank',
    labelKey: 'Channel Consumption Ranking',
    specKey: 'spec_channel_rank',
  },
  {
    value: 'popular-channel',
    labelKey: 'Popular Channels',
    specKey: 'spec_popular_channel',
  },
]

// ============================================================================
// Theme Setup (same pattern as user-charts.tsx)
// ============================================================================

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

// ============================================================================
// Overview Cards
// ============================================================================

interface OverviewCardProps {
  overview: SupplierStatsOverviewResult | null
  loading: boolean
}

function OverviewCards({ overview, loading }: OverviewCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className='bg-background/60 rounded-xl border p-3'>
            <Skeleton className='h-full min-h-32 w-full' />
          </div>
        ))}
      </div>
    )
  }

  if (!overview) return null

  const margin = overview.profit_margin
  const marginTone = margin >= 0 ? 'positive' : 'negative'
  const trendIcon = margin >= 0 ? TrendingUp : TrendingDown
  const profitTone: 'teal' | 'rose' =
    overview.gross_profit >= 0 ? 'teal' : 'rose'

  const items = [
    {
      key: 'total-cost',
      title: t('Total Cost'),
      value: formatQuota(overview.total_cost),
      description: t('Platform upstream spend'),
      icon: Activity,
      tone: 'gray' as const,
    },
    {
      key: 'total-revenue',
      title: t('Total Revenue'),
      value: formatQuota(overview.total_charge),
      description: t('Total amount charged from users'),
      icon: Coins,
      tone: 'gray' as const,
    },
    {
      key: 'gross-profit',
      title: t('Gross Profit'),
      value: formatQuota(overview.gross_profit),
      description: t('Revenue minus cost'),
      icon: trendIcon,
      tone: profitTone,
    },
    {
      key: 'profit-margin',
      title: t('Profit Margin'),
      value: `${margin.toFixed(1)}%`,
      description: t('Profit divided by revenue'),
      icon: trendIcon,
      tone: marginTone === 'positive' ? 'teal' : 'rose',
    },
    {
      key: 'request-count',
      title: t('Request Count'),
      value: overview.request_count.toLocaleString(),
      description: t('Total requests with supplier pricing'),
      icon: Activity,
      tone: 'gray' as const,
    },
    {
      key: 'supplier-count',
      title: t('Supplier Count'),
      value: String(overview.supplier_count),
      description: t('Active suppliers in period'),
      icon: Warehouse,
      tone: 'gray' as const,
    },
    {
      key: 'channel-count',
      title: t('Channel Count'),
      value: String(overview.channel_count),
      description: t('Active channels in period'),
      icon: GitBranch,
      tone: 'gray' as const,
    },
  ]

  return (
    <StaggerContainer className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {items.map((it) => (
        <StaggerItem key={it.key} className='bg-background/60 rounded-xl border p-3'>
          <StatCard
            title={it.title}
            value={it.value}
            description={it.description}
            icon={it.icon}
            tone={it.tone}
            loading={loading}
          />
        </StaggerItem>
      ))}
    </StaggerContainer>
  )
}

// ============================================================================
// Chart Panel
// ============================================================================

interface ChartPanelProps {
  chartData: ProcessedSupplierChartData
  isLoading: boolean
  themeReady: boolean
}

function ChartPanel({ chartData, isLoading, themeReady }: ChartPanelProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()

  return (
    <div className='grid gap-3'>
      {SUPPLIER_CHARTS.map((chart) => {
        const spec = chartData[chart.specKey]

        return (
          <div key={chart.value} className='overflow-hidden rounded-lg border'>
            <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
              <BarChart3 className='text-muted-foreground/60 size-4' />
              <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
            </div>

            <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
              {isLoading ? (
                <Skeleton className='h-full w-full' />
              ) : (
                themeReady &&
                spec && (
                  <VChart
                    key={`supplier-${chart.value}-${resolvedTheme}-${customization.preset}`}
                    spec={{
                      ...spec,
                      theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                      background: 'transparent',
                    }}
                    option={VCHART_OPTION}
                  />
                )
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ============================================================================
// Main Component
// ============================================================================

function formatQuota(value: number): string {
  const { config } = getCurrencyDisplay()
  const usd = value / config.quotaPerUnit
  const symbol = '$'

  if (Math.abs(usd) >= 1_000_000) {
    return `${symbol}${(usd / 1_000_000).toFixed(2)}M`
  }
  if (Math.abs(usd) >= 1_000) {
    return `${symbol}${(usd / 1_000).toFixed(2)}K`
  }
  return `${symbol}${usd.toFixed(2)}`
}

export function SupplierStatsSection() {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const [selectedRange, setSelectedRange] = useState(29)
  const [timeRange, setTimeRange] = useState(() => {
    const { start, end } = getRollingDateRange(29)
    return {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  })

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    updateTheme()
  }, [resolvedTheme])

  const handleRangeChange = (days: number) => {
    setSelectedRange(days)
    const { start, end } = getRollingDateRange(days)
    setTimeRange({
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    })
  }

  const { data: overviewData, isLoading: isLoadingOverview } = useQuery({
    queryKey: ['supplier-stats', 'overview', timeRange],
    queryFn: () =>
      getSupplierStatsOverview({
        start_timestamp: timeRange.start_timestamp,
        end_timestamp: timeRange.end_timestamp,
      }),
    select: (res) => (res.success ? res.data : null),
    staleTime: 60_000,
  })

  const { data: supplierData, isLoading: isLoadingSupplier } = useQuery({
    queryKey: ['supplier-stats', 'by-supplier', timeRange],
    queryFn: () =>
      getSupplierStatsBySupplier({
        start_timestamp: timeRange.start_timestamp,
        end_timestamp: timeRange.end_timestamp,
      }),
    select: (res) =>
      res.success && res.data ? res.data.items : ([] as SupplierStatsBySupplierItem[]),
    staleTime: 60_000,
  })

  const { data: channelData, isLoading: isLoadingChannel } = useQuery({
    queryKey: ['supplier-stats', 'by-channel', timeRange],
    queryFn: () =>
      getSupplierStatsByChannel({
        start_timestamp: timeRange.start_timestamp,
        end_timestamp: timeRange.end_timestamp,
      }),
    select: (res) =>
      res.success && res.data ? res.data.items : ([] as SupplierStatsByChannelItem[]),
    staleTime: 60_000,
  })

  const isLoading = isLoadingOverview || isLoadingSupplier || isLoadingChannel

  const chartData = useMemo(
    () =>
      processSupplierChartData(
        isLoadingSupplier ? [] : (supplierData ?? []),
        isLoadingChannel ? [] : (channelData ?? []),
        t,
        resolvedTheme
      ),
    [supplierData, channelData, isLoadingSupplier, isLoadingChannel, t, resolvedTheme]
  )

  if (overviewData === undefined && !isLoadingOverview) {
    return (
      <div className='flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3'>
        <AlertTriangle className='size-4 text-destructive' />
        <span className='text-sm text-destructive'>
          {t('Failed to load supplier stats')}
        </span>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {/* Overview Cards */}
      <OverviewCards overview={overviewData ?? null} loading={isLoadingOverview} />

      {/* Filter Bar */}
      <div className='flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2'>
        <div className='flex shrink-0 items-center gap-1.5 rounded-lg border p-0.5'>
          {TIME_RANGE_PRESETS.map((preset) => (
            <button
              key={preset.days}
              type='button'
              onClick={() => handleRangeChange(preset.days)}
              className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                selectedRange === preset.days
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              {t(preset.label)}
            </button>
          ))}
        </div>

        {isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      {/* Charts */}
      <ChartPanel
        chartData={chartData}
        isLoading={isLoading}
        themeReady={themeReady}
      />
    </div>
  )
}
