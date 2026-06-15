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
*/
import {
  useCallback,
  useDeferredValue,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import '@/styles/model-square-design.css'
import { useTranslation } from 'react-i18next'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { CLEARMAAS_DOCS_URL, MaasPublicHeader } from '@/features/home/maas-home'
import { CopyButton } from '@/components/copy-button'
import { getModelSquare } from './api'
import type { ModelSquareItem, ModelSquareVersion } from './types'

const categoryChips = ['All', 'Text', 'Image', 'Audio', 'Video', 'Free Trial']
const inputTypes = ['Text', 'Image', 'File', 'Audio', 'Video']
const discountRange = { min: 0, max: 100, step: 1 }
const defaultDiscountPercent = 100
const fallbackContextStops = [0, 128_000, 1_000_000, 2_000_000]
const fallbackContextRange = { min: 0, max: 2_000_000, step: 1000 }
const inputTypeIcons: Record<string, IconName> = {
  Text: 'text',
  Image: 'image',
  File: 'file',
  Audio: 'audio',
  Video: 'video',
}
const parameterIcons: Record<string, IconName> = {
  max_completion_tokens: 'hash',
  temperature: 'temperature',
  top_p: 'top-p',
  presence_penalty: 'plus',
  frequency_penalty: 'lines',
  reasoning_effort: 'sparkle',
}
const protocolIcons: Record<string, IconName> = {
  'OpenAI Chat Completions': 'chat',
  'OpenAI Responses': 'responses',
  'OpenAI Images': 'image',
  'Anthropic Messages': 'anthropic',
  'Google Gemini': 'sparkle',
  'Google Video': 'video',
}
const reasoningIcons: Record<string, IconName> = {
  'No reasoning': 'minus',
  'Switchable reasoning': 'lines',
  'Always-on reasoning': 'anthropic',
}
type IconName =
  | 'anthropic'
  | 'audio'
  | 'chat'
  | 'file'
  | 'hash'
  | 'image'
  | 'lines'
  | 'minus'
  | 'openai'
  | 'plus'
  | 'responses'
  | 'sparkle'
  | 'temperature'
  | 'text'
  | 'top-p'
  | 'video'
  | 'provider'
type SidebarFilterKey =
  | 'inputType'
  | 'developer'
  | 'vendor'
  | 'parameter'
  | 'protocol'
  | 'reasoning'
type TranslateFn = (key: string, options?: Record<string, unknown>) => string

const zhFallbackLabels: Record<string, string> = {
  All: '全部',
  Text: '文本',
  Image: '图像',
  File: '文件',
  Audio: '音频',
  Video: '视频',
  'Free Trial': '免费试用',
  'Model Square': '模型广场',
  'Input type': '输入类型',
  'Output type': '输出类型',
  'Context length': '上下文长度',
  Discount: '折扣',
  Developer: '开发商',
  Provider: '供应商',
  'Supported parameters': '支持参数',
  'Supported protocols': '支持协议',
  Reasoning: '推理',
  'Reasoning mode': '推理模式',
  Models: '模型',
  Model: '模型',
  Tags: '标签',
  Latency: '延迟',
  Success: '成功率',
  'Copy model ID': '复制模型 ID',
  'Copied model ID': '已复制模型 ID',
  Input: '输入',
  Output: '输出',
  'Max output': '最大输出',
  token: 'token',
  requests: '请求',
  Latest: '最新',
  'No reasoning': '无推理',
  'Switchable reasoning': '可切换推理',
  'Always-on reasoning': '固定推理',
  'OpenAI Chat Completions': 'OpenAI Chat Completions',
  'OpenAI Responses': 'OpenAI Responses',
  'OpenAI Images': 'OpenAI Images',
  'Anthropic Messages': 'Anthropic Messages',
  'Google Gemini': 'Google Gemini',
  'Google Video': 'Google Video',
  'Max Completion Tokens': '最大补全 tokens',
  Temperature: '温度',
  'Top P': 'Top P',
  'Presence Penalty': '存在惩罚',
  'Frequency Penalty': '频率惩罚',
  'Reasoning Effort': '推理强度',
  'All contexts': '全部上下文',
  'All discounts': '全部折扣',
  'All discount levels': '全部折扣',
  '{{count}} off or lower': '{{count}}折及以下',
  'Discount <= {{count}} off': '折扣 <= {{count}}折',
  'Expand more': '展开更多',
  Collapse: '收起',
  'No sidebar filters applied': '未应用侧栏筛选',
  'Default route': '默认路由',
  'Official route': '官方路由',
  'models, rebuilt with the project version view':
    '个模型，已按项目版本视图重构',
  '{{count}} versions': '{{count}} 个版本',
  '{{count}}% discount': '{{count}}% 折扣',
  '{{count}} off': '{{count}}折',
  '{{count}} channel versions, supporting discount and performance comparison':
    '共 {{count}} 个渠道版本，支持折扣与性能对比',
  'Available from {{count}} providers': '可用于 {{count}} 个供应商',
  'Available from {{count}} channels': '可用 {{count}} 个渠道',
  'Filtered {{summary}}': '已筛选 {{summary}}',
  'million tokens': '百万 tokens',
  Context: '上下文',
  'Tool Orchestration': '工具编排',
  'API Request': 'API 请求',
  Chat: '对话',
  'Best {{count}}% discount': '最低 {{count}}% 折扣',
  'Discount price': '优惠价',
  'Official price': '官方价',
  'No version data': '暂无版本数据',
  Price: '价格',
  'View versions': '查看版本',
  'Connect now': '立即接入',
  'Promo switch': '切换横幅',
  'Search models': '搜索模型',
  'Search model name, model ID, or version ID':
    '搜索模型名称、模型 ID 或版本 ID',
  'Toggle sort': '切换排序',
  'Loading...': '加载中...',
  'Unable to load models. Please try again later.':
    '无法加载模型，请稍后重试。',
  'No matching models': '没有匹配的模型',
  'Back to top': '返回顶部',
}
const zhForcedLabels = new Set([
  'Discount',
  'All discount levels',
  '{{count}} off or lower',
  'Discount <= {{count}} off',
])

function useModelSquareT(): TranslateFn {
  const { t, i18n } = useTranslation()
  return (key: string, options?: Record<string, unknown>) => {
    const translated = t(key, options)
    if (!i18n.language?.startsWith('zh')) {
      return translated
    }
    const fallback = zhFallbackLabels[key]
    if (fallback && zhForcedLabels.has(key)) {
      return interpolateFallback(fallback, options)
    }
    if (!fallback || translated !== key) {
      return translated
    }
    return interpolateFallback(fallback, options)
  }
}

function interpolateFallback(
  value: string,
  options?: Record<string, unknown>
): string {
  if (!options) {
    return value
  }
  return value.replace(/\{\{(\w+)}}/g, (match, name: string) => {
    const replacement = options[name]
    return replacement === undefined || replacement === null
      ? match
      : String(replacement)
  })
}

export function ModelSquare() {
  const mt = useModelSquareT()
  const { status } = useStatus()
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('All')
  const [sortBy, setSortBy] = useState<'updated' | 'discount'>('updated')
  const [contextMinTokens, setContextMinTokens] = useState(0)
  const [discountPercent, setDiscountPercent] = useState(defaultDiscountPercent)
  const deferredContextMinTokens = useDeferredValue(contextMinTokens)
  const deferredDiscountPercent = useDeferredValue(discountPercent)
  const [activeFilters, setActiveFilters] = useState<
    Record<SidebarFilterKey, Set<string>>
  >({
    inputType: new Set(),
    developer: new Set(),
    vendor: new Set(),
    parameter: new Set(),
    protocol: new Set(),
    reasoning: new Set(),
  })
  const [expandedLists, setExpandedLists] = useState<
    Record<'developer' | 'vendor' | 'parameter' | 'protocol', boolean>
  >({
    developer: false,
    vendor: false,
    parameter: false,
    protocol: false,
  })

  const query = useQuery({
    queryKey: ['model-square'],
    queryFn: () => getModelSquare(),
  })

  const data = query.data
  const contextStops = useMemo(
    () => buildContextStops(data?.models ?? []),
    [data?.models]
  )
  const contextRange = useMemo(
    () => buildContextRange(data?.models ?? []),
    [data?.models]
  )
  const contextLabels = useMemo(
    () => contextStops.map((value) => formatContextStopLabel(value, mt)),
    [contextStops, mt]
  )
  const developers = useMemo(
    () => uniqueValues((data?.models ?? []).map((model) => model.vendor_name)),
    [data?.models]
  )
  const vendorIconByName = useMemo(() => {
    const entries = new Map<string, string>()
    for (const vendor of data?.vendors ?? []) {
      if (vendor.name && vendor.icon) entries.set(vendor.name, vendor.icon)
    }
    for (const model of data?.models ?? []) {
      if (!model.vendor_name || entries.has(model.vendor_name)) {
        continue
      }
      const icon = model.vendor_icon || model.icon
      if (icon) {
        entries.set(model.vendor_name, icon)
      }
    }
    return entries
  }, [data?.models, data?.vendors])
  const availableParameters = useMemo(
    () =>
      uniqueValues(
        (data?.models ?? []).flatMap((model) => modelParameters(model))
      ),
    [data?.models]
  )
  const availableProtocols = useMemo(
    () =>
      uniqueValues(
        (data?.models ?? []).flatMap((model) => modelProtocols(model))
      ),
    [data?.models]
  )
  const availableReasoningModes = useMemo(
    () =>
      uniqueValues((data?.models ?? []).map((model) => modelReasoning(model))),
    [data?.models]
  )
  const categoryCounts = useMemo(
    () =>
      buildCategoryCounts(data?.models ?? [], {
        activeFilters,
        contextMinTokens: deferredContextMinTokens,
        discountPercent: deferredDiscountPercent,
        search: keyword.trim().toLowerCase(),
      }),
    [
      activeFilters,
      data?.models,
      deferredContextMinTokens,
      deferredDiscountPercent,
      keyword,
    ]
  )
  const filteredModels = useMemo(() => {
    const search = keyword.trim().toLowerCase()
    const models = (data?.models ?? []).filter((model) => {
      if (!matchesCategory(model, category)) {
        return false
      }
      return modelMatchesActiveFilters(
        model,
        activeFilters,
        deferredContextMinTokens,
        deferredDiscountPercent,
        search
      )
    })

    if (sortBy === 'discount') {
      return [...models].sort(
        (a, b) => modelBestDiscountRatioValue(a) - modelBestDiscountRatioValue(b)
      )
    }
    return [...models].sort(compareModelsByVendorLatest)
  }, [
    activeFilters,
    category,
    deferredContextMinTokens,
    deferredDiscountPercent,
    data?.models,
    keyword,
    sortBy,
  ])

  const toggleFilter = (key: SidebarFilterKey, value: string) => {
    setActiveFilters((current) => {
      const next = { ...current, [key]: new Set(current[key]) }
      if (next[key].has(value)) {
        next[key].delete(value)
      } else {
        next[key].add(value)
      }
      return next
    })
  }

  const filterSummary = buildFilterSummary(
    activeFilters,
    contextMinTokens,
    discountPercent,
    mt
  )

  return (
    <div className='model-square-page model-square-root'>
      <MaasPublicHeader
        active='model-square'
        brandTitle={mt('Model Square')}
        docsHref={CLEARMAAS_DOCS_URL}
      />

      <div className='shell'>
        <PromoBanner models={data?.models ?? []} />

        <div className='content-shell'>
          <aside className='sidebar'>
            <FilterGroup title={mt('Input type')}>
              {inputTypes.map((item) => (
                <FilterButton
                  key={item}
                  active={activeFilters.inputType.has(item)}
                  label={mt(item)}
                  icon={inputTypeIcons[item]}
                  onClick={() => toggleFilter('inputType', item)}
                />
              ))}
            </FilterGroup>

            <RangeFilter
              title={mt('Context length')}
              labels={contextLabels}
              value={contextMinTokens}
              min={contextRange.min}
              max={contextRange.max}
              step={contextRange.step}
              formatValue={(value) => formatContextLabel(value, mt)}
              onChange={setContextMinTokens}
            />

            <RangeFilter
              title={mt('Discount')}
              labels={[
                formatDiscountStopLabel(discountRange.min, mt),
                formatDiscountStopLabel(50, mt),
                formatDiscountStopLabel(discountRange.max, mt),
              ]}
              value={discountPercent}
              min={discountRange.min}
              max={discountRange.max}
              step={discountRange.step}
              formatValue={(value) =>
                mt('{{count}} off or lower', {
                  count: formatDiscountSliderValue(value),
                })
              }
              onChange={setDiscountPercent}
            />

            <FilterGroup title={mt('Developer')}>
              {renderExpandableFilters(
                developers,
                expandedLists.developer,
                (item) => (
                  <FilterButton
                    key={item}
                    active={activeFilters.developer.has(item)}
                    label={item}
                    iconName={toColorIconName(vendorIconByName.get(item))}
                    onClick={() => toggleFilter('developer', item)}
                    provider
                  />
                )
              )}
              <ExpandButton
                expanded={expandedLists.developer}
                onClick={() =>
                  setExpandedLists((current) => ({
                    ...current,
                    developer: !current.developer,
                  }))
                }
              />
            </FilterGroup>

            <FilterGroup title={mt('Provider')}>
              {renderExpandableFilters(
                (data?.vendors ?? []).map((item) => item.name),
                expandedLists.vendor,
                (item) => (
                  <FilterButton
                    key={item}
                    active={activeFilters.vendor.has(item)}
                    label={item}
                    iconName={toColorIconName(vendorIconByName.get(item))}
                    onClick={() => toggleFilter('vendor', item)}
                    provider
                  />
                )
              )}
              <ExpandButton
                expanded={expandedLists.vendor}
                onClick={() =>
                  setExpandedLists((current) => ({
                    ...current,
                    vendor: !current.vendor,
                  }))
                }
              />
            </FilterGroup>

            <FilterGroup title={mt('Supported parameters')}>
              {renderExpandableFilters(
                availableParameters,
                expandedLists.parameter,
                (item) => (
                  <FilterButton
                    key={item}
                    active={activeFilters.parameter.has(item)}
                    label={mt(formatParameterLabel(item))}
                    icon={parameterIcons[item] || 'provider'}
                    onClick={() => toggleFilter('parameter', item)}
                  />
                )
              )}
              <ExpandButton
                expanded={expandedLists.parameter}
                onClick={() =>
                  setExpandedLists((current) => ({
                    ...current,
                    parameter: !current.parameter,
                  }))
                }
              />
            </FilterGroup>

            <FilterGroup title={mt('Supported protocols')}>
              {renderExpandableFilters(
                availableProtocols,
                expandedLists.protocol,
                (item) => (
                  <FilterButton
                    key={item}
                    active={activeFilters.protocol.has(item)}
                    label={mt(item)}
                    icon={protocolIcons[item] || 'provider'}
                    onClick={() => toggleFilter('protocol', item)}
                  />
                )
              )}
              <ExpandButton
                expanded={expandedLists.protocol}
                onClick={() =>
                  setExpandedLists((current) => ({
                    ...current,
                    protocol: !current.protocol,
                  }))
                }
              />
            </FilterGroup>

            <FilterGroup title={mt('Reasoning')}>
              {availableReasoningModes.map((item) => (
                <FilterButton
                  key={item}
                  active={activeFilters.reasoning.has(item)}
                  label={mt(item)}
                  icon={reasoningIcons[item] || 'provider'}
                  onClick={() => toggleFilter('reasoning', item)}
                />
              ))}
            </FilterGroup>

            <p className='filter-summary'>{filterSummary}</p>
          </aside>

          <main className='main'>
            <section className='heading-panel'>
              <div className='heading-row'>
                <div className='page-title'>
                  <h2>{mt('Models')}</h2>
                  <p>
                    <span>{filteredModels.length}</span>{' '}
                    {mt('models, rebuilt with the project version view')}
                  </p>
                </div>
                <div className='search-tools'>
                  <label
                    className='search-box'
                    aria-label={mt('Search models')}
                  >
                    <svg viewBox='0 0 24 24' aria-hidden='true'>
                      <circle cx='11' cy='11' r='5' />
                      <path d='m20 20-4.2-4.2' />
                    </svg>
                    <input
                      type='text'
                      value={keyword}
                      placeholder={mt(
                        'Search model name, model ID, or version ID'
                      )}
                      onChange={(event) => setKeyword(event.target.value)}
                    />
                  </label>
                  <button
                    className='sort-button'
                    type='button'
                    aria-label={mt('Toggle sort')}
                    onClick={() =>
                      setSortBy((current) =>
                        current === 'updated' ? 'discount' : 'updated'
                      )
                    }
                  >
                    <svg viewBox='0 0 24 24' aria-hidden='true'>
                      <path d='M8 6v12' />
                      <path d='m5.5 8.5 2.5-2.5 2.5 2.5' />
                      <path d='M16 18V6' />
                      <path d='m13.5 15.5 2.5 2.5 2.5-2.5' />
                    </svg>
                    <span>{sortLabel(sortBy, mt)}</span>
                  </button>
                </div>
              </div>
              <div className='chips'>
                {categoryChips.map((item) => (
                  <button
                    key={item}
                    className={cn('chip', category === item && 'active')}
                    type='button'
                    onClick={() => setCategory(item)}
                  >
                    <span>{mt(item)}</span>
                    <span className='count'>
                      {categoryCounts[item] ?? 0}
                    </span>
                  </button>
                ))}
              </div>
            </section>

            <section className='model-stream'>
              {query.isLoading ? (
                <EmptyState title={mt('Loading...')} />
              ) : query.isError ? (
                <EmptyState
                  title={
                    query.error instanceof Error
                      ? query.error.message
                      : mt('Unable to load models. Please try again later.')
                  }
                />
              ) : filteredModels.length === 0 ? (
                <EmptyState title={mt('No matching models')} />
              ) : (
                filteredModels.map((model, index) => (
                  <ModelCard
                    key={model.name}
                    model={model}
                    index={index}
                    discountPercent={discountPercent}
                  />
                ))
              )}
            </section>
          </main>
        </div>
      </div>

      <div className='floating-actions'>
        <button
          type='button'
          aria-label={mt('Back to top')}
          onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
        >
          <svg viewBox='0 0 24 24'>
            <path d='M12 17V7' />
            <path d='m8.5 10.5 3.5-3.5 3.5 3.5' />
          </svg>
        </button>
        <Link to='/'>
          <svg viewBox='0 0 24 24'>
            <path d='m5 11 7-6 7 6' />
            <path d='M7.5 10.5V19h9v-8.5' />
          </svg>
        </Link>
      </div>
    </div>
  )
}

function PromoBanner(props: { models: ModelSquareItem[] }) {
  const mt = useModelSquareT()
  const [activeSlide, setActiveSlide] = useState(0)
  const sourceModels = newestModelByVendor(props.models)
  const slides =
    sourceModels.length > 0
      ? sourceModels.map((model) => ({
          pill:
            modelBestDiscount(model) > 0
              ? formatModelDiscountLabel(model, mt)
              : mt('{{count}} versions', { count: model.version_count }),
          title: model.display_name || model.name,
          copy: model.description || model.name,
          proof: [
            formatContext(model),
            modelReasoning(model) ? mt(modelReasoning(model)) : '-',
            mt('Available from {{count}} channels', {
              count: modelChannelCount(model),
            }),
          ],
          secondary: mt('View versions'),
          primary: mt('Connect now'),
        }))
      : [
          {
            pill: mt('Loading...'),
            title: mt('Model Square'),
            copy: mt('Loading...'),
            proof: [mt('Loading...')],
            secondary: mt('View versions'),
            primary: mt('Connect now'),
          },
        ]
  const slide = slides[activeSlide % slides.length]
  const currentSlideIndex = activeSlide % slides.length

  useEffect(() => {
    if (slides.length <= 1) return
    const timer = window.setInterval(() => {
      setActiveSlide((current) => (current + 1) % slides.length)
    }, 5200)
    return () => window.clearInterval(timer)
  }, [slides.length])

  return (
    <section className='promo-banner'>
      <article className='hero-card'>
        <div className='promo-slider'>
          <div className='promo-slide active'>
            <div className='hero-inner'>
              <div className='hero-copy-block'>
                <span className='promo-pill'>{slide.pill}</span>
                <h3>{slide.title}</h3>
                <div className='banner-copy'>
                  <p>{slide.copy}</p>
                  <div className='banner-proof'>
                    {slide.proof.map((item) => (
                      <span key={item}>{item}</span>
                    ))}
                  </div>
                </div>
                <div className='hero-actions'>
                  <Link className='cta-secondary' to='/dashboard'>
                    {slide.secondary}
                  </Link>
                  <Link className='cta-primary' to='/dashboard'>
                    {slide.primary}
                  </Link>
                </div>
              </div>
              <div className='hero-art' aria-hidden='true'>
                <span className='hero-orbit' />
              </div>
            </div>
          </div>
        </div>
        {slides.length > 1 ? (
          <div className='promo-dots' aria-label={mt('Promo switch')}>
            {slides.map((item, index) => (
              <button
                key={`${item.title}-${index}`}
                className={cn('promo-dot', currentSlideIndex === index && 'active')}
                type='button'
                aria-label={item.title}
                onClick={() => setActiveSlide(index)}
              />
            ))}
          </div>
        ) : null}
      </article>
    </section>
  )
}

function FilterGroup(props: { title: string; children: React.ReactNode }) {
  return (
    <section className='filter-group'>
      <div className='filter-header'>
        <span>{props.title}</span>
        <span>⌃</span>
      </div>
      <div className='filter-list'>{props.children}</div>
    </section>
  )
}

function FilterButton(props: {
  active?: boolean
  icon?: IconName
  iconName?: string
  label: string
  onClick?: () => void
  provider?: boolean
}) {
  return (
    <button
      className={cn(
        'check-item',
        props.provider && 'provider-filter-item',
        props.active && 'active'
      )}
      type='button'
      aria-pressed={Boolean(props.active)}
      onClick={props.onClick}
    >
      <span
        className={cn(
          props.provider ? 'provider-icon' : 'check-icon',
          props.iconName && 'system-icon'
        )}
      >
        {props.iconName ? (
          getLobeIcon(props.iconName, 16)
        ) : (
          <FilterIcon name={props.icon || 'provider'} />
        )}
      </span>
      <span className='check-label'>{props.label}</span>
      {props.active ? (
        <span className='filter-checkmark' aria-hidden='true'>
          <svg viewBox='0 0 24 24'>
            <path d='m5 12 4 4 10-10' />
          </svg>
        </span>
      ) : null}
    </button>
  )
}

function FilterIcon(props: { name: IconName }) {
  switch (props.name) {
    case 'text':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M5 6.5h14' />
          <path d='M12 6.5v11' />
          <path d='M8 17.5h8' />
        </svg>
      )
    case 'image':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <rect x='4' y='5' width='16' height='14' rx='2' />
          <circle cx='9' cy='10' r='1.5' />
          <path d='m7 16 3.5-3.5 2.5 2.5 2-2 2.5 3' />
        </svg>
      )
    case 'file':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M8 3.5h6l4 4V20H8z' />
          <path d='M14 3.5V8h4' />
        </svg>
      )
    case 'audio':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M10 8v8' />
          <path d='M14 6v12' />
          <path d='M7 10v4' />
          <path d='M17 10v4' />
        </svg>
      )
    case 'video':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <rect x='4' y='6' width='11' height='12' rx='2' />
          <path d='m15 10 5-3v10l-5-3z' />
        </svg>
      )
    case 'anthropic':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M6 18 12 5l6 13' />
          <path d='M8.5 13h7' />
        </svg>
      )
    case 'openai':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M12 4.5a4 4 0 0 1 3.8 2.8 4 4 0 0 1 3.3 5.3 4 4 0 0 1-2.5 6.6 4 4 0 0 1-7.2 0 4 4 0 0 1-4.8-5.5A4 4 0 0 1 8.2 7.3 4 4 0 0 1 12 4.5Z' />
        </svg>
      )
    case 'sparkle':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M12 4.5 14 10l5.5 2-5.5 2L12 19.5 10 14 4.5 12 10 10z' />
        </svg>
      )
    case 'hash':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M9 4 7 20' />
          <path d='M17 4 15 20' />
          <path d='M4 9h16' />
          <path d='M3 15h16' />
        </svg>
      )
    case 'temperature':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M12 5v9' />
          <path d='M9 8V5a3 3 0 1 1 6 0v3' />
          <path d='M8.5 15.5a3.5 3.5 0 1 0 7 0c0-1.4-.8-2.4-1.5-3.2V10' />
        </svg>
      )
    case 'top-p':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M7 16c2.2-4.8 7.8-4.8 10 0' />
          <path d='M7 12c2.2-3 7.8-3 10 0' />
          <path d='M12 6v12' />
        </svg>
      )
    case 'plus':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M12 5v14' />
          <path d='M5 12h14' />
        </svg>
      )
    case 'lines':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M7 8h10' />
          <path d='M7 12h10' />
          <path d='M7 16h10' />
        </svg>
      )
    case 'chat':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M5 7h14v8H9l-4 4z' />
        </svg>
      )
    case 'responses':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M6 7h12v10H8l-2 2z' />
          <path d='M9 11h6' />
          <path d='M9 14h4' />
        </svg>
      )
    case 'minus':
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <path d='M7 12h10' />
        </svg>
      )
    case 'provider':
    default:
      return (
        <svg viewBox='0 0 24 24' aria-hidden='true'>
          <rect x='5' y='6' width='14' height='12' rx='2' />
          <path d='M12 6v12' />
          <path d='M8.5 10h7' />
        </svg>
      )
  }
}

function RangeFilter({
  title,
  labels,
  value,
  min,
  max,
  step,
  formatValue,
  onChange,
}: {
  title: string
  labels: string[]
  value: number
  min: number
  max: number
  step: number
  formatValue: (value: number) => string
  onChange: (value: number) => void
}) {
  const railRef = useRef<HTMLDivElement | null>(null)
  const draggingRef = useRef(false)
  const draftValueRef = useRef(value)
  const commitTimerRef = useRef<number | null>(null)
  const frameRef = useRef<number | null>(null)
  const [draftValue, setDraftValue] = useState(value)

  const clampValue = useCallback(
    (nextValue: number) => {
      const safeMin = Number.isFinite(min) ? min : 0
      const safeMax = Number.isFinite(max) && max > safeMin ? max : safeMin
      const bounded = Math.max(safeMin, Math.min(safeMax, nextValue))
      if (step <= 0) return bounded
      const stepped = Math.round((bounded - safeMin) / step) * step + safeMin
      return Math.max(safeMin, Math.min(safeMax, stepped))
    },
    [max, min, step]
  )

  const displayedValue = draggingRef.current ? draftValue : value
  const rangeRatio =
    max > min ? (clampValue(displayedValue) - min) / (max - min) : 0

  useEffect(() => {
    if (draggingRef.current) return
    const bounded = clampValue(value)
    draftValueRef.current = bounded
    setDraftValue(bounded)
    railRef.current?.style.setProperty('--range-ratio', String(rangeRatio))
  }, [clampValue, rangeRatio, value])

  useEffect(() => {
    return () => {
      if (commitTimerRef.current !== null) {
        window.clearTimeout(commitTimerRef.current)
      }
      if (frameRef.current !== null) {
        window.cancelAnimationFrame(frameRef.current)
      }
    }
  }, [])

  const updateVisualValue = useCallback(
    (nextValue: number) => {
      const bounded = clampValue(nextValue)
      draftValueRef.current = bounded
      setDraftValue(bounded)
      if (frameRef.current !== null) {
        window.cancelAnimationFrame(frameRef.current)
      }
      frameRef.current = window.requestAnimationFrame(() => {
        const ratio = max > min ? (bounded - min) / (max - min) : 0
        railRef.current?.style.setProperty('--range-ratio', String(ratio))
        frameRef.current = null
      })
      return bounded
    },
    [clampValue, max, min]
  )

  const scheduleCommit = useCallback(
    (nextValue: number) => {
      if (commitTimerRef.current !== null) {
        window.clearTimeout(commitTimerRef.current)
      }
      commitTimerRef.current = window.setTimeout(() => {
        onChange(clampValue(nextValue))
        commitTimerRef.current = null
      }, 90)
    },
    [clampValue, onChange]
  )

  const commitValue = useCallback(
    (nextValue: number) => {
      if (commitTimerRef.current !== null) {
        window.clearTimeout(commitTimerRef.current)
        commitTimerRef.current = null
      }
      const bounded = clampValue(nextValue)
      updateVisualValue(bounded)
      onChange(bounded)
    },
    [clampValue, onChange, updateVisualValue]
  )

  const commitClientX = useCallback(
    (clientX: number) => {
      const rail = railRef.current
      if (!rail) return
      const rect = rail.getBoundingClientRect()
      const ratio = rect.width ? (clientX - rect.left) / rect.width : 0
      const nextValue = min + Math.max(0, Math.min(1, ratio)) * (max - min)
      const bounded = updateVisualValue(nextValue)
      scheduleCommit(bounded)
    },
    [max, min, scheduleCommit, updateVisualValue]
  )

  return (
    <section className='filter-group'>
      <div className='filter-header'>
        <span>{title}</span>
        <span>⌃</span>
      </div>
      <div className='range-track'>
        <div
          className='range-rail'
          ref={railRef}
          style={{ '--range-ratio': rangeRatio } as CSSProperties}
        >
          <div className='range-fill' />
          <div className='range-thumb' />
          <input
            className='range-input'
            type='range'
            min={min}
            max={max}
            step={step}
            value={clampValue(displayedValue)}
            aria-label={title}
            onPointerDown={(event) => {
              draggingRef.current = true
              event.currentTarget.setPointerCapture?.(event.pointerId)
              commitClientX(event.clientX)
            }}
            onClick={(event) => {
              commitClientX(event.clientX)
            }}
            onPointerMove={(event) => {
              if (!draggingRef.current) return
              commitClientX(event.clientX)
            }}
            onPointerUp={(event) => {
              draggingRef.current = false
              event.currentTarget.releasePointerCapture?.(event.pointerId)
              commitValue(draftValueRef.current)
            }}
            onPointerCancel={() => {
              draggingRef.current = false
              commitValue(draftValueRef.current)
            }}
            onChange={(event) => {
              const nextValue = Number(event.target.value)
              if (draggingRef.current) {
                const bounded = updateVisualValue(nextValue)
                scheduleCommit(bounded)
                return
              }
              commitValue(nextValue)
            }}
          />
        </div>
        <div className='range-labels'>
          {labels.map((label) => (
            <span key={label}>{label}</span>
          ))}
        </div>
        <div className='range-value'>{formatValue(displayedValue)}</div>
      </div>
    </section>
  )
}

function ExpandButton(props: { expanded: boolean; onClick: () => void }) {
  const mt = useModelSquareT()
  return (
    <button
      className={cn('expand-link', props.expanded && 'is-open')}
      type='button'
      onClick={props.onClick}
    >
      {props.expanded ? mt('Collapse') : mt('Expand more')}
    </button>
  )
}

function renderExpandableFilters<T>(
  items: T[],
  expanded: boolean,
  render: (item: T) => React.ReactNode
) {
  return items.slice(0, expanded ? items.length : 3).map(render)
}

function ModelCard(props: {
  model: ModelSquareItem
  index: number
  discountPercent: number
}) {
  const mt = useModelSquareT()
  const model = props.model
  const title = model.display_name || model.name
  const versions = filterVersionsByDiscount(
    model.versions ?? [],
    props.discountPercent
  )
  const discount = modelBestDiscount(model)
  const inputTypesForDisplay = modelInputTypes(model)
  const outputTypesForDisplay = modelOutputTypes(model)

  return (
    <article className='model-card'>
      <div className='card-top'>
        <div className='card-name'>
          <span className='logo-orb'>
            {getSystemIcon(toColorIconName(model.icon || model.vendor_icon), 24)}
          </span>
          <div className='name-block'>
            <h3>{title}</h3>
            <div className='slug-row'>
              <code>{model.name}</code>
              <span className='mini-icon'>
                <svg viewBox='0 0 24 24'>
                  <rect x='9' y='9' width='10' height='10' rx='2' />
                  <rect x='5' y='5' width='10' height='10' rx='2' />
                </svg>
              </span>
              <span className='mini-icon'>
                <svg viewBox='0 0 24 24'>
                  <path d='M12 5v14' />
                  <path d='M5 12h14' />
                </svg>
              </span>
            </div>
          </div>
        </div>
        <div className='badge-stack'>
          {discount > 0 ? (
            <span className='status-badge discount'>
              {formatModelDiscountLabel(model, mt)}
            </span>
          ) : null}
          <span className='status-badge'>
            {mt('{{count}} versions', { count: model.version_count })}
          </span>
          {(model.tags ?? []).slice(0, 1).map((tag) => (
            <span key={tag} className='status-badge'>
              {mt(tag)}
            </span>
          ))}
        </div>
      </div>

      {model.description ? (
        <p className='card-desc'>{model.description}</p>
      ) : null}

      <div className='meta-grid'>
        <div className='meta-item'>
          <strong>{mt('Input type')}</strong>
          <span className='input-tags'>
            {inputTypesForDisplay.length > 0
              ? typeSquares(inputTypesForDisplay)
              : '-'}
          </span>
        </div>
        <div className='meta-item'>
          <strong>{mt('Output type')}</strong>
          <span className='input-tags'>
            {outputTypesForDisplay.length > 0
              ? typeSquares(outputTypesForDisplay)
              : '-'}
          </span>
        </div>
        <div className='meta-item'>
          <strong>{mt('Official price')}</strong>
          <span>{formatPricePair(model.price_info, 'original', mt)}</span>
        </div>
        <div className='meta-item'>
          <strong>{mt('Context')}</strong>
          <span>{formatContext(model)}</span>
        </div>
        <div className='meta-item'>
          <strong>{mt('Max output')}</strong>
          <span>
            {model.max_output ? formatTokenAmount(model.max_output) : '-'}
          </span>
        </div>
      </div>

      <VersionPanel versions={versions} />
      <ModelFooter model={model} />
    </article>
  )
}

function VersionPanel(props: { versions: ModelSquareVersion[] }) {
  const mt = useModelSquareT()
  return (
    <div className='version-panel'>
      <div className='version-head'>
        <div>
          <span>
            {mt(
              '{{count}} channel versions, supporting discount and performance comparison',
              {
                count: props.versions.length,
              }
            )}
          </span>
        </div>
      </div>
      <div className='version-columns'>
        <span>{mt('Model')}</span>
        <span>{mt('Discount')}</span>
        <span>{mt('Price')}</span>
        <span>{mt('Tags')}</span>
        <span>{mt('Latency')}</span>
      </div>
      <div className='version-list'>
        {props.versions.length > 0 ? (
          props.versions.map((version, index) => (
            <div
              key={`${version.channel_id}-${version.model_name}-${index}`}
              className='version-row'
            >
              <div className='version-id' data-label={mt('Model')}>
                <code>{version.model_name}</code>
                <CopyButton
                  value={version.model_name}
                  className='version-copy-button'
                  iconClassName='version-copy-icon'
                  tooltip={mt('Copy model ID')}
                  successTooltip={mt('Copied model ID')}
                  aria-label={mt('Copy model ID')}
                />
              </div>
              <div className='version-discount-cell' data-label={mt('Discount')}>
                {renderVersionDiscount(version, mt)}
              </div>
              <div className='version-price' data-label={mt('Price')}>
                {renderVersionPrice(version, mt)}
              </div>
              <div className='version-tags' data-label={mt('Tags')}>
                {versionTagLabels(version, mt).map((tag) => (
                  <span key={tag} className='version-tag'>
                    {tag}
                  </span>
                ))}
              </div>
              <div className='metric-cell' data-label={mt('Latency')}>
                <strong>{formatLatency(version.latency_seconds)}</strong>
              </div>
            </div>
          ))
        ) : (
          <div className='version-empty'>{mt('No version data')}</div>
        )}
      </div>
    </div>
  )
}

function ModelFooter(props: { model: ModelSquareItem }) {
  const mt = useModelSquareT()
  const channels = getModelChannels(props.model)
  const uptime = modelUptime(props.model)
  const activeUptimeBars =
    typeof uptime === 'number' ? Math.round((uptime / 100) * 31) : -1
  return (
    <div className='card-footer'>
      <div className='footer-left'>
        <span>
          {mt('Available from {{count}} channels', {
            count: channels.length,
          })}
        </span>
      </div>
      <div className='footer-right'>
        <div className='uptime-header'>
          <span>{formatUpdatedTime(props.model.updated_time)}</span>
          <strong>{formatSuccess(uptime)}</strong>
        </div>
        <div className='uptime-graph' aria-label={formatSuccess(uptime)}>
          {Array.from({ length: 32 }).map((_, index) => (
            <span
              key={index}
              className={cn(
                'uptime-bar',
                index <= activeUptimeBars && 'active'
              )}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

function getModelChannels(model: ModelSquareItem) {
  const channels = new Map<string, { key: string; name: string; icon?: string }>()
  for (const version of model.versions ?? []) {
    const key =
      version.channel_id > 0
        ? String(version.channel_id)
        : version.channel_name?.trim() || version.model_name
    if (!key || channels.has(key)) continue
    const name = version.channel_name?.trim() || version.model_name
    channels.set(key, {
      key,
      name,
      icon: toColorIconName(model.vendor_icon || model.icon),
    })
  }
  return [...channels.values()]
}

function newestModelByVendor(models: ModelSquareItem[]) {
  const selected = new Map<string, ModelSquareItem>()
  const sortedModels = [...models].sort(compareModelsByLatest)
  for (const model of sortedModels) {
    const key =
      model.vendor_name ||
      (model.vendor_id ? String(model.vendor_id) : '') ||
      model.name
    if (!selected.has(key)) {
      selected.set(key, model)
    }
  }
  return [...selected.values()]
}

function compareModelsByLatest(a: ModelSquareItem, b: ModelSquareItem) {
  const latestDiff = modelLatestSortValue(b) - modelLatestSortValue(a)
  if (latestDiff !== 0) return latestDiff
  const updatedDiff = (b.updated_time ?? 0) - (a.updated_time ?? 0)
  if (updatedDiff !== 0) return updatedDiff
  const versionDiff = (b.version_count ?? 0) - (a.version_count ?? 0)
  if (versionDiff !== 0) return versionDiff
  return a.name.localeCompare(b.name)
}

function compareModelsByVendorLatest(a: ModelSquareItem, b: ModelSquareItem) {
  const vendorCompare = modelVendorSortKey(a).localeCompare(modelVendorSortKey(b))
  if (vendorCompare !== 0) return vendorCompare
  return compareModelsByLatest(a, b)
}

function modelVendorSortKey(model: ModelSquareItem) {
  return (
    model.vendor_name?.trim().toLowerCase() ||
    (model.vendor_id ? String(model.vendor_id).padStart(8, '0') : '') ||
    model.name.toLowerCase()
  )
}

function modelChannelCount(model: ModelSquareItem) {
  return getModelChannels(model).length || model.version_count || 0
}

function getSystemIcon(iconName: string | undefined, size: number) {
  return getLobeIcon(iconName || 'Bot', size)
}

function toColorIconName(iconName: string | undefined) {
  if (!iconName || iconName.includes('.')) return iconName
  return `${iconName}.Color`
}

function EmptyState(props: { title: string }) {
  return (
    <div className='empty-state'>
      <strong>{props.title}</strong>
    </div>
  )
}

function typeSquares(types: string[]) {
  return types.map((type, index) => (
    <span key={`${type}-${index}`} className='tiny-square' title={type}>
      <svg viewBox='0 0 24 24'>
        {type === 'Text' ? (
          <>
            <path d='M5 6.5h14' />
            <path d='M12 6.5v11' />
            <path d='M8 17.5h8' />
          </>
        ) : type === 'Image' ? (
          <>
            <rect x='4' y='5' width='16' height='14' rx='2' />
            <circle cx='9' cy='10' r='1.5' />
            <path d='m7 16 3.5-3.5 2.5 2.5 2-2 2.5 3' />
          </>
        ) : type === 'File' ? (
          <>
            <path d='M8 3.5h6l4 4V20H8z' />
            <path d='M14 3.5V8h4' />
          </>
        ) : type === 'Audio' ? (
          <>
            <path d='M10 8v8' />
            <path d='M14 6v12' />
            <path d='M7 10v4' />
            <path d='M17 10v4' />
          </>
        ) : (
          <>
            <rect x='4' y='6' width='11' height='12' rx='2' />
            <path d='m15 10 5-3v10l-5-3z' />
          </>
        )}
      </svg>
    </span>
  ))
}

function formatRatio(value: number | undefined, t: TranslateFn) {
  if (typeof value !== 'number') return '-'
  return `$${value.toLocaleString(undefined, { maximumFractionDigits: 4 })}/${t('million tokens')}`
}

function formatPricePair(
  priceInfo: ModelSquareItem['price_info'] | ModelSquareVersion['price_info'],
  kind: 'original' | 'discounted',
  t: TranslateFn
) {
  if (!priceInfo) return '-'
  const input =
    kind === 'original'
      ? priceInfo.input_original_price
      : priceInfo.input_discounted_price
  const output =
    kind === 'original'
      ? priceInfo.output_original_price
      : priceInfo.output_discounted_price
  if (priceInfo.quota_type === 1) {
    return formatUsd(input)
  }
  if (typeof input !== 'number' || input <= 0) return '-'
  const inputText = formatUsd(input)
  const outputText =
    typeof output === 'number' && output > 0 ? formatUsd(output) : '-'
  return `${inputText} / ${outputText}/${t('million tokens')}`
}

function renderVersionPrice(version: ModelSquareVersion, t: TranslateFn) {
  const priceInfo = version.price_info
  if (!priceInfo) {
    return <span className='version-price-empty'>-</span>
  }
  const original = formatShortPricePair(priceInfo, 'original')
  const discounted = formatShortPricePair(priceInfo, 'discounted')
  const hasDiscount =
    typeof priceInfo.discount_ratio === 'number' &&
    priceInfo.discount_ratio > 0 &&
    priceInfo.discount_ratio < 1 &&
    original !== discounted

  return (
    <>
      <span className='version-price-line'>
        {hasDiscount ? (
          <>
            <span className='version-price-original'>{original}</span>
            <span className='version-price-arrow'>→</span>
            <strong>{discounted}</strong>
          </>
        ) : (
          <strong>{discounted}</strong>
        )}
      </span>
      <span className='version-price-unit'>
        {priceInfo.quota_type === 1 ? t('requests') : `/ ${t('million tokens')}`}
      </span>
    </>
  )
}

function renderVersionDiscount(version: ModelSquareVersion, t: TranslateFn) {
  const priceInfo = version.price_info
  const label = formatDiscountRatio(
    version.discount_ratio ?? priceInfo?.discount_ratio,
    t
  )
  if (!label) {
    return <span className='version-discount is-zero'>{t('Original price')}</span>
  }
  return <span className='version-discount'>{label}</span>
}

function formatShortPricePair(
  priceInfo: NonNullable<ModelSquareVersion['price_info']>,
  kind: 'original' | 'discounted'
) {
  const input =
    kind === 'original'
      ? priceInfo.input_original_price
      : priceInfo.input_discounted_price
  const output =
    kind === 'original'
      ? priceInfo.output_original_price
      : priceInfo.output_discounted_price
  if (priceInfo.quota_type === 1) {
    return formatUsd(input)
  }
  if (typeof input !== 'number' || input <= 0) return '-'
  if (typeof output !== 'number' || output <= 0) return formatUsd(input)
  return `${formatUsd(input)} / ${formatUsd(output)}`
}

function formatUsd(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `$${stripTrailingZeros(value.toLocaleString(undefined, {
    maximumFractionDigits: 6,
  }))}`
}

function stripTrailingZeros(value: string) {
  return value.replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.0+$/, '')
}

function formatDiscountRatio(value: number | undefined, t: TranslateFn) {
  if (typeof value !== 'number' || value <= 0 || value >= 1) {
    return ''
  }
  return t('{{count}} off', {
    count: stripTrailingZeros((value * 10).toFixed(1)),
  })
}

function uniqueValues(values: Array<string | undefined>) {
  return [...new Set(values.filter(Boolean) as string[])].sort((a, b) =>
    a.localeCompare(b)
  )
}

function matchesSet(values: string[], selected: Set<string>) {
  if (selected.size === 0) return true
  return values.some((value) => selected.has(value))
}

function matchesCategory(model: ModelSquareItem, category: string) {
  if (category === 'All') return true
  if (category === 'Free Trial') {
    return model.tags.some((tag) => /free|trial|免费/i.test(tag))
  }
  return modelCategoryTypes(model).includes(category)
}

function modelMatchesActiveFilters(
  model: ModelSquareItem,
  activeFilters: Record<SidebarFilterKey, Set<string>>,
  contextMinTokens: number,
  discountPercent: number,
  search: string
) {
  if (!matchesSet(modelInputTypes(model), activeFilters.inputType)) {
    return false
  }
  if (!matchesSet([model.vendor_name || ''], activeFilters.developer)) {
    return false
  }
  if (!matchesSet([model.vendor_name || ''], activeFilters.vendor)) {
    return false
  }
  if (!matchesSet(modelParameters(model), activeFilters.parameter)) {
    return false
  }
  if (!matchesSet(modelProtocols(model), activeFilters.protocol)) {
    return false
  }
  if (
    activeFilters.reasoning.size > 0 &&
    !matchesSet([modelReasoning(model)], activeFilters.reasoning)
  ) {
    return false
  }
  if (contextMinTokens > 0) {
    const contextTokens = modelContextTokens(model)
    if (!contextTokens || contextTokens < contextMinTokens) return false
  }
  if (!hasVersionsAtDiscount(model, discountPercent)) return false
  if (!search) return true
  return modelSearchText(model).includes(search)
}

function modelSearchText(model: ModelSquareItem) {
  return [
    model.name,
    model.display_name,
    model.description ?? '',
    model.vendor_name ?? '',
    ...model.tags,
    ...model.versions.flatMap((version) => [
      version.model_name,
      version.upstream_key,
      version.channel_name,
      ...(version.channel_tags ?? []),
      version.mode_description ?? '',
    ]),
  ]
    .join(' ')
    .toLowerCase()
}

function buildCategoryCounts(
  models: ModelSquareItem[],
  filters: {
    activeFilters: Record<SidebarFilterKey, Set<string>>
    contextMinTokens: number
    discountPercent: number
    search: string
  }
) {
  const counts: Record<string, number> = {}
  for (const chip of categoryChips) {
    counts[chip] = models.filter(
      (model) =>
        matchesCategory(model, chip) &&
        modelMatchesActiveFilters(
          model,
          filters.activeFilters,
          filters.contextMinTokens,
          filters.discountPercent,
          filters.search
        )
    ).length
  }
  return counts
}

function modelCategoryTypes(model: ModelSquareItem) {
  const values = new Set<string>([
    ...modelInputTypes(model),
    ...modelOutputTypes(model),
  ])
  for (const tag of model.tags ?? []) {
    if (/^(image|vision|图片|视觉)$/i.test(tag)) values.add('Image')
    if (/^(audio|音频)$/i.test(tag)) values.add('Audio')
    if (/^(video|视频)$/i.test(tag)) values.add('Video')
    if (/^(text|文本)$/i.test(tag)) values.add('Text')
  }
  if (values.size === 0) values.add('Text')
  return [...values]
}

function modelInputTypes(model: ModelSquareItem) {
  if (model.input_types?.length) {
    return model.input_types
  }
  return []
}

function modelOutputTypes(model: ModelSquareItem) {
  if (model.output_types?.length) {
    return model.output_types
  }
  return []
}

function modelParameters(model: ModelSquareItem) {
  if (model.parameters?.length) {
    return model.parameters
  }
  return []
}

function modelProtocols(model: ModelSquareItem) {
  if (model.protocols?.length) {
    return model.protocols
  }
  return model.capabilities?.supported_endpoints ?? []
}

function modelReasoning(model: ModelSquareItem) {
  if (model.reasoning) {
    return model.reasoning
  }
  return ''
}

function modelDiscount(model: ModelSquareItem) {
  const ratioDiscounts = (model.versions ?? [])
    .map((version) =>
      discountPercentFromRatio(
        version.discount_ratio ?? version.price_info?.discount_ratio
      )
    )
    .filter((value) => value > 0)
  if (ratioDiscounts.length > 0) {
    return Math.max(...ratioDiscounts)
  }
  const versionDiscounts = (model.versions ?? [])
    .map((version) => version.discount_percent)
    .filter((value): value is number => typeof value === 'number' && value > 0)
  if (versionDiscounts.length > 0) {
    return Math.max(...versionDiscounts)
  }
  const modelRatioDiscount = discountPercentFromRatio(model.price_info?.discount_ratio)
  if (modelRatioDiscount > 0) return modelRatioDiscount
  if (typeof model.discount_percent === 'number') {
    return model.discount_percent
  }
  const base = model.price_info?.input_ratio
  if (typeof base !== 'number' || base <= 0) return 0
  const versionRatios = (model.versions ?? [])
    .map((version) => version.price_info?.input_ratio)
    .filter((value): value is number => typeof value === 'number' && value > 0)
  if (versionRatios.length === 0) return 0
  const lowest = Math.min(...versionRatios, base)
  return Math.max(0, Math.min(100, Math.round((1 - lowest / base) * 100)))
}

function modelBestDiscount(model: ModelSquareItem) {
  return modelDiscount(model)
}

function discountPercentFromRatio(ratio: number | undefined) {
  if (typeof ratio !== 'number' || ratio <= 0 || ratio >= 1) return 0
  return Math.max(0, Math.min(100, Math.round((1 - ratio) * 100)))
}

function modelBestDiscountRatio(model: ModelSquareItem) {
  const ratios = (model.versions ?? [])
    .map((version) => version.discount_ratio ?? version.price_info?.discount_ratio)
    .filter((value): value is number => typeof value === 'number' && value > 0)
  if (ratios.length > 0) {
    return Math.min(...ratios)
  }
  const ratio = model.price_info?.discount_ratio
  return typeof ratio === 'number' && ratio > 0 ? ratio : undefined
}

function modelBestDiscountRatioValue(model: ModelSquareItem) {
  return modelBestDiscountRatio(model) ?? 1
}

function hasVersionsAtDiscount(model: ModelSquareItem, discountPercent: number) {
  return filterVersionsByDiscount(model.versions ?? [], discountPercent).length > 0
}

function filterVersionsByDiscount(
  versions: ModelSquareVersion[],
  discountPercent: number
) {
  if (discountPercent >= defaultDiscountPercent) return versions
  const thresholdRatio = Math.max(0, Math.min(100, discountPercent)) / 100
  return versions.filter((version) => {
    const ratio = version.discount_ratio ?? version.price_info?.discount_ratio
    return typeof ratio === 'number' && ratio > 0 && ratio <= thresholdRatio
  })
}

function modelLatestSortValue(model: ModelSquareItem) {
  const values = [
    releaseDateValue(model.name),
    ...(model.versions ?? []).map((version) =>
      releaseDateValue(version.model_name)
    ),
  ].filter((value) => value > 0)
  if (values.length > 0) return Math.max(...values)
  const semanticVersions = [
    semanticVersionValue(model.name),
    ...(model.versions ?? []).map((version) =>
      semanticVersionValue(version.model_name)
    ),
  ].filter((value) => value > 0)
  if (semanticVersions.length > 0) return Math.max(...semanticVersions)
  return model.updated_time ?? 0
}

function releaseDateValue(name: string | undefined) {
  const match = name?.match(
    /(?:^|[^0-9])((?:20\d{2})(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01]))(?:[^0-9]|$)/
  )
  return match ? Number(match[1]) : 0
}

function semanticVersionValue(name: string | undefined) {
  const matches = name
    ?.toLowerCase()
    .matchAll(
      /(?:^|[^0-9])(\d+)(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[^0-9]|$)/g
    )
  if (!matches) return 0
  let best = 0
  for (const match of matches) {
    const rawParts = [match[1], match[2], match[3], match[4]]
    if (Number(rawParts[0]) >= 2000) continue
    let value = 0
    for (const [index, part] of rawParts.entries()) {
      if (index > 0 && part && part.length > 2) break
      value = value * 1000 + (part ? Number(part) : 0)
    }
    best = Math.max(best, value)
  }
  return best > 0 ? 10_000_000_000 + best : 0
}

function formatModelDiscountLabel(
  model: ModelSquareItem,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  const ratio = modelBestDiscountRatio(model)
  if (typeof ratio === 'number' && ratio > 0 && ratio < 1) {
    return t('Best {{count}} off', {
      count: stripTrailingZeros((ratio * 10).toFixed(1)),
    })
  }
  return t('{{count}}% discount', { count: modelBestDiscount(model) })
}

function buildContextStops(models: ModelSquareItem[]) {
  const values = uniqueNumberValues([
    0,
    ...models.map((model) => modelContextTokens(model)),
  ])
  if (values.length <= 1 || values[values.length - 1] <= 0) {
    return fallbackContextStops
  }
  if (values.length <= 4) return values
  return [
    0,
    values[Math.max(1, Math.floor((values.length - 1) / 3))],
    values[Math.max(1, Math.floor(((values.length - 1) * 2) / 3))],
    values[values.length - 1],
  ]
}

function buildContextRange(models: ModelSquareItem[]) {
  const max = Math.max(0, ...models.map((model) => modelContextTokens(model)))
  if (max <= 0) {
    return fallbackContextRange
  }
  return { min: 0, max, step: contextStep(max) }
}

function contextStep(max: number) {
  if (max >= 100_000) return 1000
  if (max >= 10_000) return 100
  return 1
}

function uniqueNumberValues(values: number[]) {
  return [...new Set(values.filter((value) => Number.isFinite(value)))].sort(
    (a, b) => a - b
  )
}

function formatContext(model: ModelSquareItem) {
  if (model.context_tokens && model.context_tokens > 0) {
    return formatTokenAmount(model.context_tokens)
  }
  return '-'
}

function modelContextTokens(model: ModelSquareItem) {
  if (model.context_tokens && model.context_tokens > 0) {
    return model.context_tokens
  }
  return 0
}

function formatTokenAmount(tokens: number) {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(2)}M`
  }
  if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(2)}K`
  }
  return `${tokens}`
}

function formatContextStopLabel(
  tokens: number,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (tokens <= 0) return t('All')
  return formatTokenAmount(tokens)
}

function formatParameterLabel(value: string) {
  return value
    .split('_')
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(' ')
}

function formatLatency(value: number | undefined) {
  if (typeof value !== 'number' || value <= 0) {
    return '-'
  }
  return `${value.toFixed(2)}s`
}

function formatSuccess(value: number | undefined) {
  if (typeof value !== 'number' || value < 0) {
    return '-'
  }
  return `${value.toFixed(2)}%`
}

function versionTagLabels(version: ModelSquareVersion, t: TranslateFn) {
  const tags = (version.channel_tags ?? [])
    .map((tag) => tag.trim())
    .filter(Boolean)
  if (tags.length > 0) return uniqueValues(tags)
  return [version.channel_name || t('Default route')]
}

function formatUpdatedTime(value: number | undefined) {
  if (!value) {
    return '-'
  }
  return new Date(value * 1000).toLocaleString(undefined, {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function modelUptime(model: ModelSquareItem) {
  const rates = (model.versions ?? [])
    .map((version) => version.success_rate)
    .filter((value): value is number => typeof value === 'number' && value >= 0)
  if (rates.length === 0) {
    return undefined
  }
  return rates.reduce((sum, value) => sum + value, 0) / rates.length
}

function formatContextLabel(
  tokens: number,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (tokens <= 0) return t('All contexts')
  if (tokens >= 1_000_000) return '>= 1M'
  return `>= ${Math.round(tokens / 1_000)}K`
}

function formatDiscountSliderValue(value: number) {
  return stripTrailingZeros((value / 10).toFixed(1))
}

function formatDiscountStopLabel(
  value: number,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  return t('{{count}} off', {
    count: formatDiscountSliderValue(value),
  })
}

function sortLabel(
  sortBy: 'updated' | 'discount',
  t: (key: string) => string
) {
  if (sortBy === 'discount') return t('Discount')
  return t('Latest')
}

function buildFilterSummary(
  activeFilters: Record<SidebarFilterKey, Set<string>>,
  contextMinTokens: number,
  discountPercent: number,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  const labels: string[] = []
  for (const [key, values] of Object.entries(activeFilters)) {
    if (values.size > 0) {
      labels.push(
        `${t(filterGroupLabel(key as SidebarFilterKey))}: ${[...values].join(' / ')}`
      )
    }
  }
  if (contextMinTokens > 0) labels.push(formatContextLabel(contextMinTokens, t))
  if (discountPercent < defaultDiscountPercent) {
    labels.push(
      t('Discount <= {{count}} off', {
        count: formatDiscountSliderValue(discountPercent),
      })
    )
  }
  return labels.length
    ? t('Filtered {{summary}}', { summary: labels.join(' · ') })
    : t('No sidebar filters applied')
}

function filterGroupLabel(key: SidebarFilterKey) {
  const labels: Record<SidebarFilterKey, string> = {
    inputType: 'Input type',
    developer: 'Developer',
    vendor: 'Provider',
    parameter: 'Supported parameters',
    protocol: 'Supported protocols',
    reasoning: 'Reasoning mode',
  }
  return labels[key]
}
