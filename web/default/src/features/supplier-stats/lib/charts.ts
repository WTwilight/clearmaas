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
import { dataScheme as vchartDefaultDataScheme } from '@visactor/vchart/esm/theme/color-scheme/builtin/default'
import { getCurrencyDisplay } from '@/lib/currency'
import type {
  SupplierStatsBySupplierItem,
  SupplierStatsByChannelItem,
  ProcessedSupplierChartData,
} from '../types'

type TFunction = (key: string) => string

const THEME_CHART_COLOR_VARIABLES = [
  '--chart-1',
  '--chart-2',
  '--chart-3',
  '--chart-4',
  '--chart-5',
] as const

function getThemeChartColors(themeKey?: string): string[] {
  if (typeof document === 'undefined') return []
  void themeKey

  const bodyStyle = window.getComputedStyle(document.body)
  const rootStyle = window.getComputedStyle(document.documentElement)

  return THEME_CHART_COLOR_VARIABLES.map((name) => {
    return (
      bodyStyle.getPropertyValue(name) || rootStyle.getPropertyValue(name)
    ).trim()
  }).filter(Boolean)
}

function getVChartDefaultColors(domainLength: number, themeKey?: string) {
  const themeColors = getThemeChartColors(themeKey)
  if (themeColors.length > 0) {
    return Array.from(
      { length: Math.max(domainLength, themeColors.length) },
      (_, index) => themeColors[index % themeColors.length]
    )
  }

  const scheme =
    vchartDefaultDataScheme.find(
      (item) => !item.maxDomainLength || domainLength <= item.maxDomainLength
    ) ?? vchartDefaultDataScheme[vchartDefaultDataScheme.length - 1]

  return scheme.scheme
}

const CHART_COLOR_FALLBACKS = [
  '#5B8FF9',
  '#5AD8A6',
  '#F6BD16',
  '#E8684A',
  '#6DC8EC',
  '#9270CA',
  '#FF9D4D',
  '#269A99',
  '#FF99C3',
  '#5D7092',
]

function renderQuotaCompat(rawQuota: number, digits = 2): string {
  const { config, meta } = getCurrencyDisplay()
  if (meta.kind === 'tokens') return rawQuota.toLocaleString()
  const usd = rawQuota / config.quotaPerUnit
  const rate = 'exchangeRate' in meta ? meta.exchangeRate : 1
  const symbol = 'symbol' in meta ? meta.symbol : '$'
  const value = usd * rate
  return symbol + value.toFixed(digits)
}

function formatInt(value: number): string {
  return Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value)
}

function formatQuotaValue(value: number): string {
  return renderQuotaCompat(value, 4)
}

function formatQuotaTotal(value: number): string {
  return renderQuotaCompat(value, 2)
}

export function processSupplierChartData(
  supplierItems: SupplierStatsBySupplierItem[],
  channelItems: SupplierStatsByChannelItem[],
  t?: TFunction,
  themeKey?: string
): ProcessedSupplierChartData {
  const tt: TFunction = t ?? ((x) => x)
  const themeColors = getThemeChartColors(themeKey)
  const chartColorRange =
    themeColors.length > 0
      ? Array.from(
          { length: Math.max(20, themeColors.length) },
          (_, index) => themeColors[index % themeColors.length]
        )
      : CHART_COLOR_FALLBACKS

  const emptyResult: ProcessedSupplierChartData = {
    spec_supplier_rank: buildEmptyBarSpec(tt('Supplier Consumption Ranking')),
    spec_channel_rank: buildEmptyBarSpec(tt('Channel Consumption Ranking')),
    spec_popular_channel: buildEmptyBarSpec(tt('Popular Channels')),
  }

  if ((!supplierItems || supplierItems.length === 0) && (!channelItems || channelItems.length === 0)) {
    return emptyResult
  }

  // --- Supplier Consumption Ranking ---
  const supplierRankValues = supplierItems.map((item) => ({
    Supplier: item.supplier_name || '-',
    total_charge: item.total_charge,
    total_cost: item.total_cost,
    gross_profit: item.gross_profit,
    request_count: item.request_count,
    rawCharge: item.total_charge,
    Usage: Number((item.total_charge / getCurrencyDisplay().config.quotaPerUnit).toFixed(4)),
  }))

  const supplierColorMap: Record<string, string> = {}
  supplierRankValues.forEach((v, i) => {
    supplierColorMap[v.Supplier] = chartColorRange[i % chartColorRange.length]
  })

  const supplierTotalCharge = supplierRankValues.reduce(
    (s, v) => s + v.total_charge,
    0
  )

  // --- Channel Consumption Ranking ---
  // Deduplicate: same channel may appear under multiple suppliers; sum them
  const channelMap = new Map<string, { total_charge: number; request_count: number; supplier: string; channel_name: string }>()
  channelItems.forEach((item) => {
    const key = `${item.channel_id}`
    const existing = channelMap.get(key)
    if (existing) {
      existing.total_charge += item.total_charge
      existing.request_count += item.request_count
    } else {
      channelMap.set(key, {
        total_charge: item.total_charge,
        request_count: item.request_count,
        supplier: item.supplier_name || '-',
        channel_name: item.channel_name || `#${item.channel_id}`,
      })
    }
  })

  const channelRankValues = Array.from(channelMap.values())
    .sort((a, b) => b.total_charge - a.total_charge)
    .map((item) => ({
      Channel: `${item.supplier} / ${item.channel_name}`,
      total_charge: item.total_charge,
      request_count: item.request_count,
      rawCharge: item.total_charge,
      Usage: Number((item.total_charge / getCurrencyDisplay().config.quotaPerUnit).toFixed(4)),
    }))

  const channelColorMap: Record<string, string> = {}
  channelRankValues.forEach((v, i) => {
    channelColorMap[v.Channel] = chartColorRange[i % chartColorRange.length]
  })

  const channelTotalCharge = channelRankValues.reduce(
    (s, v) => s + v.total_charge,
    0
  )

  // --- Popular Channels ---
  const popularChannelValues = Array.from(channelMap.values())
    .sort((a, b) => b.request_count - a.request_count)
    .map((item) => ({
      Channel: `${item.supplier} / ${item.channel_name}`,
      request_count: item.request_count,
      total_charge: item.total_charge,
      rawCharge: item.total_charge,
      Usage: Number((item.total_charge / getCurrencyDisplay().config.quotaPerUnit).toFixed(4)),
    }))

  const totalRequests = popularChannelValues.reduce(
    (s, v) => s + v.request_count,
    0
  )

  return {
    spec_supplier_rank: buildSupplierRankSpec(
      supplierRankValues,
      supplierColorMap,
      supplierTotalCharge,
      tt,
    ),
    spec_channel_rank: buildChannelRankSpec(
      channelRankValues,
      channelColorMap,
      channelTotalCharge,
      tt,
    ),
    spec_popular_channel: buildPopularChannelSpec(
      popularChannelValues,
      channelColorMap,
      totalRequests,
      tt,
    ),
  }
}

function buildEmptyBarSpec(title: string) {
  return {
    type: 'bar',
    data: [{ id: 'emptyData', values: [] }],
    xField: 'value',
    yField: 'label',
    direction: 'horizontal',
    title: {
      visible: true,
      text: title,
      subtext: 'No data available',
    },
    legends: { visible: false },
    background: { fill: 'transparent' },
  }
}

function buildSupplierRankSpec(
  values: Array<{
    Supplier: string
    rawCharge: number
    Usage: number
  }>,
  colorMap: Record<string, string>,
  totalCharge: number,
  tt: TFunction
) {
  const label = tt('Supplier Consumption Ranking')
  const subtext = `${tt('Total:')} ${formatQuotaTotal(totalCharge)}`

  return {
    type: 'bar',
    data: [{ id: 'supplierRankData', values }],
    xField: 'rawCharge',
    yField: 'Supplier',
    seriesField: 'Supplier',
    direction: 'horizontal',
    title: {
      visible: true,
      text: label,
      subtext,
    },
    legends: { visible: false },
    bar: {
      state: { hover: { stroke: '#000', lineWidth: 1 } },
    },
    label: {
      visible: true,
      position: 'outside',
      formatMethod: (value: number) => formatQuotaValue(value),
      style: { fontSize: 11 },
    },
    axes: [
      { orient: 'left', type: 'band' },
      { orient: 'bottom', type: 'linear', visible: false },
    ],
    tooltip: {
      mark: {
        content: [
          {
            key: (datum: Record<string, unknown>) => datum?.Supplier,
            value: (datum: Record<string, unknown>) =>
              formatQuotaValue(Number(datum?.rawCharge) || 0),
          },
        ],
      },
    },
    color: { specified: colorMap },
    background: { fill: 'transparent' },
    animation: true,
  }
}

function buildChannelRankSpec(
  values: Array<{
    Channel: string
    rawCharge: number
    Usage: number
  }>,
  colorMap: Record<string, string>,
  totalCharge: number,
  tt: TFunction
) {
  const label = tt('Channel Consumption Ranking')
  const subtext = `${tt('Total:')} ${formatQuotaTotal(totalCharge)}`

  return {
    type: 'bar',
    data: [{ id: 'channelRankData', values }],
    xField: 'rawCharge',
    yField: 'Channel',
    seriesField: 'Channel',
    direction: 'horizontal',
    title: {
      visible: true,
      text: label,
      subtext,
    },
    legends: { visible: false },
    bar: {
      state: { hover: { stroke: '#000', lineWidth: 1 } },
    },
    label: {
      visible: true,
      position: 'outside',
      formatMethod: (value: number) => formatQuotaValue(value),
      style: { fontSize: 11 },
    },
    axes: [
      { orient: 'left', type: 'band' },
      { orient: 'bottom', type: 'linear', visible: false },
    ],
    tooltip: {
      mark: {
        content: [
          {
            key: (datum: Record<string, unknown>) => datum?.Channel,
            value: (datum: Record<string, unknown>) =>
              formatQuotaValue(Number(datum?.rawCharge) || 0),
          },
        ],
      },
    },
    color: { specified: colorMap },
    background: { fill: 'transparent' },
    animation: true,
  }
}

function buildPopularChannelSpec(
  values: Array<{
    Channel: string
    request_count: number
    rawCharge: number
  }>,
  colorMap: Record<string, string>,
  totalRequests: number,
  tt: TFunction
) {
  const label = tt('Popular Channels')
  const subtext = `${tt('Total:')} ${formatInt(totalRequests)} requests`

  return {
    type: 'bar',
    data: [{ id: 'popularChannelData', values }],
    xField: 'request_count',
    yField: 'Channel',
    seriesField: 'Channel',
    direction: 'horizontal',
    title: {
      visible: true,
      text: label,
      subtext,
    },
    legends: { visible: false },
    bar: {
      state: { hover: { stroke: '#000', lineWidth: 1 } },
    },
    label: {
      visible: true,
      position: 'outside',
      formatMethod: (value: number) => formatInt(value),
      style: { fontSize: 11 },
    },
    axes: [
      { orient: 'left', type: 'band' },
      { orient: 'bottom', type: 'linear', visible: false },
    ],
    tooltip: {
      mark: {
        content: [
          {
            key: (datum: Record<string, unknown>) => datum?.Channel,
            value: (datum: Record<string, unknown>) =>
              formatInt(Number(datum?.request_count) || 0),
          },
        ],
      },
    },
    color: { specified: colorMap },
    background: { fill: 'transparent' },
    animation: true,
  }
}
