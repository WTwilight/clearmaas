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
import { memo, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronDown,
  KeyRound,
  Settings2,
  WalletCards,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { getUserModels, getUserGroups } from '@/lib/api'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import { formatCurrencyUSD } from '@/lib/format'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { DateTimePicker } from '@/components/datetime-picker'
import { MultiSelect } from '@/components/multi-select'
import { Checkbox } from '@/components/ui/checkbox'
import {
  createApiKey,
  updateApiKey,
  getApiKey,
  getAvailablePricingModels,
  getGroupedAvailablePricingModels,
} from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  apiKeyFormSchema,
  type ApiKeyFormValues,
  getApiKeyFormDefaultValues,
  transformFormDataToPayload,
  transformApiKeyToFormDefaults,
} from '../lib'
import {
  type ApiKey,
  type SelectablePricingModel,
  type SelectablePricingModelGroup,
  type SelectablePricingModelVersion,
  type SelectModel,
} from '../types'
import {
  ApiKeyGroupCombobox,
  type ApiKeyGroupOption,
} from './api-key-group-combobox'
import { useApiKeys } from './api-keys-provider'

type ApiKeyMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: ApiKey
  side?: 'left' | 'right'
}

// ============================================================================
// Pricing Model Row — memoized to avoid re-rendering all rows when one changes
// ============================================================================

type PricingModelRowProps = {
  model: SelectablePricingModel
  fieldValue: string[]
  onToggle: (model: string) => void
}

const PricingModelRow = memo(function PricingModelRow({
  model,
  fieldValue,
  onToggle,
}: PricingModelRowProps) {
  const { t } = useTranslation()
  const isSelected = fieldValue.includes(model.model)
  return (
    <tr
      className={cn(
        'border-b last:border-0 transition-colors cursor-pointer',
        isSelected ? 'bg-primary/5' : 'hover:bg-muted/30'
      )}
      onClick={() => onToggle(model.model)}
    >
      <td className='px-3 py-2'>
        <div className='flex flex-col gap-0.5'>
          <span className='font-medium truncate max-w-[180px]'>{model.model}</span>
          <span className='text-muted-foreground text-[10px]'>{model.vendor_type}</span>
        </div>
      </td>
      <td className='px-3 py-2 text-right'>
        <PricingModelPrice model={model} />
      </td>
      <td className='px-3 py-2 text-center'>
        <Checkbox
          checked={isSelected}
          onCheckedChange={() => onToggle(model.model)}
          onClick={(e) => e.stopPropagation()}
        />
      </td>
    </tr>
  )
})

function PricingModelPrice({ model }: { model: SelectablePricingModel }) {
  const { t } = useTranslation()
  const hasRealDiscount = model.discount_ratio != null && model.discount_ratio < 1
  const hasDiscountBadge = model.discount_ratio != null
  const inputOriginal = model.input_original_price ?? 0
  const outputOriginal = model.output_original_price ?? 0
  const inputDiscounted = model.input_discounted_price ?? 0
  const outputDiscounted = model.output_discounted_price ?? 0
  const hasPrice = inputOriginal > 0 || outputOriginal > 0
  const showOriginalPrice =
    model.discount_ratio != null ||
    inputOriginal !== inputDiscounted ||
    outputOriginal !== outputDiscounted
  const isFixedPrice = model.quota_type === 1
  const discountLabel = hasRealDiscount
    ? `${(model.discount_ratio * 10).toFixed(1)}折`
    : hasDiscountBadge && model.discount_ratio === 1
    ? t('原价')
    : ''
  const unitLabel = isFixedPrice ? t('per request') : '/ 1M tokens'

  return (
    <div className='flex flex-col items-end gap-0.5'>
      {hasDiscountBadge && discountLabel && (
        <span className='inline-flex items-center rounded bg-gradient-to-r from-amber-400 to-orange-400 px-1.5 py-0.5 text-[10px] font-bold text-white leading-none'>
          {discountLabel}
        </span>
      )}
      <span className='font-mono text-[11px] tabular-nums leading-tight'>
        {hasPrice ? (
          <>
            {showOriginalPrice && inputOriginal > 0 && (
              <>
                <span className='text-muted-foreground/40 line-through'>
                  ${inputOriginal.toFixed(4)}
                </span>
                {outputOriginal > 0 && (
                  <>
                    <span className='text-muted-foreground/30 mx-0.5'>/</span>
                    <span className='text-muted-foreground/40 line-through'>
                      ${outputOriginal.toFixed(4)}
                    </span>
                  </>
                )}
              </>
            )}
            {showOriginalPrice &&
              (inputOriginal !== inputDiscounted ||
                outputOriginal !== outputDiscounted) && (
                <span className='text-foreground mx-1'>→</span>
              )}
            <span
              className={
                showOriginalPrice && inputOriginal !== inputDiscounted
                  ? 'font-bold text-foreground'
                  : ''
              }
            >
              ${inputDiscounted.toFixed(4)}
            </span>
            {outputOriginal > 0 && (
              <>
                <span className='text-muted-foreground/40 mx-0.5'>/</span>
                <span
                  className={
                    showOriginalPrice && outputOriginal !== outputDiscounted
                      ? 'font-bold text-foreground'
                      : ''
                  }
                >
                  ${outputDiscounted.toFixed(4)}
                </span>
              </>
            )}
          </>
        ) : (
          '—'
        )}
      </span>
      <span className='text-muted-foreground/50 text-[10px] leading-tight'>
        {unitLabel}
      </span>
    </div>
  )
}

function toColorIconName(iconName?: string) {
  if (!iconName) return undefined
  return iconName.includes('.') ? iconName : `${iconName}.Color`
}

type GroupedPricingModelSelectorProps = {
  groups: SelectablePricingModelGroup[]
  selected: string[]
  onChange: (models: string[]) => void
}

function GroupedPricingModelSelector({
  groups,
  selected,
  onChange,
}: GroupedPricingModelSelectorProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [vendorFilter, setVendorFilter] = useState('all')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const selectedSet = useMemo(() => new Set(selected), [selected])
  const normalizedKeyword = keyword.trim().toLowerCase()
  const vendorOptions = useMemo(() => {
    const vendors = new Map<string, string>()
    for (const group of groups) {
      const vendor = getPricingModelGroupVendor(group)
      if (!vendor) continue
      const key = vendor.toLowerCase()
      if (!vendors.has(key)) vendors.set(key, vendor)
    }
    return [...vendors.entries()]
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label))
  }, [groups])
  const sortedGroups = useMemo(
    () =>
      [...groups]
        .map((group) => ({
          ...group,
          versions: [...group.versions].sort(comparePricingModelVersions),
        }))
        .sort(comparePricingModelGroups),
    [groups]
  )
  const filteredGroups = useMemo(() => {
    const vendorFilteredGroups =
      vendorFilter === 'all'
        ? sortedGroups
        : sortedGroups.filter(
            (group) =>
              getPricingModelGroupVendor(group).toLowerCase() === vendorFilter
          )
    if (!normalizedKeyword) return vendorFilteredGroups
    return vendorFilteredGroups
      .map((group) => {
        const groupMatched = [
          group.name,
          group.display_name,
          group.vendor_name ?? '',
          ...(group.tags ?? []),
        ]
          .join(' ')
          .toLowerCase()
          .includes(normalizedKeyword)
        const versions = group.versions.filter((version) =>
          [
            version.model,
            version.upstream_key ?? '',
            version.channel_name ?? '',
            version.vendor_type,
            ...(version.channel_tags ?? []),
          ]
            .join(' ')
            .toLowerCase()
            .includes(normalizedKeyword)
        )
        return groupMatched ? group : { ...group, versions }
      })
      .filter((group) => group.versions.length > 0)
  }, [normalizedKeyword, sortedGroups, vendorFilter])

  useEffect(() => {
    if (normalizedKeyword) {
      setExpanded(new Set(filteredGroups.map((group) => group.name)))
    }
  }, [filteredGroups, normalizedKeyword])

  const allModels = uniquePricingModelNames(
    groups.flatMap((group) => group.versions.map((version) => version.model))
  )
  const filteredModels = uniquePricingModelNames(
    filteredGroups.flatMap((group) =>
      group.versions.map((version) => version.model)
    )
  )
  const filteredSelectedCount = filteredModels.filter((model) =>
    selectedSet.has(model)
  ).length
  const allSelected =
    filteredModels.length > 0 && filteredSelectedCount === filteredModels.length
  const partiallySelected = filteredSelectedCount > 0 && !allSelected

  const toggleAll = (checked: boolean) => {
    const modelSet = new Set(selected)
    for (const model of filteredModels) {
      if (checked) {
        modelSet.add(model)
      } else {
        modelSet.delete(model)
      }
    }
    onChange([...modelSet])
  }

  const toggleGroup = (group: SelectablePricingModelGroup, checked: boolean) => {
    const modelSet = new Set(selected)
    for (const version of group.versions) {
      if (checked) {
        modelSet.add(version.model)
      } else {
        modelSet.delete(version.model)
      }
    }
    onChange([...modelSet])
  }

  const toggleVersion = (model: string) => {
    const modelSet = new Set(selected)
    if (modelSet.has(model)) {
      modelSet.delete(model)
    } else {
      modelSet.add(model)
    }
    onChange([...modelSet])
  }

  const toggleExpanded = (name: string) => {
    setExpanded((current) => {
      const next = new Set(current)
      if (next.has(name)) {
        next.delete(name)
      } else {
        next.add(name)
      }
      return next
    })
  }

  return (
    <div className='mt-2 overflow-hidden rounded-xl border'>
      <div className='flex flex-col gap-2 border-b bg-muted/30 p-3 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <div className='text-sm font-medium'>
            {t('{{selected}} / {{total}} selected', {
              selected: selected.length,
              total: allModels.length,
            })}
          </div>
          <div className='text-muted-foreground text-xs'>
            {t('Grouped by model with version and channel tags')}
          </div>
        </div>
        <div className='flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center'>
          <Select value={vendorFilter} onValueChange={setVendorFilter}>
            <SelectTrigger className='h-8 w-full sm:w-40' aria-label={t('Select vendor')}>
              <SelectValue placeholder={t('Select vendor')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectItem value='all'>{t('All providers')}</SelectItem>
              {vendorOptions.map((vendor) => (
                <SelectItem key={vendor.value} value={vendor.value}>
                  {vendor.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Input
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            placeholder={t('Search model, version, or channel tag')}
            className='h-8 w-full sm:w-72'
          />
          <div className='flex h-8 items-center justify-end sm:w-8'>
            <Checkbox
              checked={allSelected ? true : partiallySelected ? 'indeterminate' : false}
              onCheckedChange={(checked) => toggleAll(checked === true)}
              onClick={(event) => event.stopPropagation()}
              aria-label={t('Select all models')}
            />
          </div>
        </div>
      </div>

      <div className='max-h-[460px] overflow-y-auto'>
        {filteredGroups.length === 0 ? (
          <div className='text-muted-foreground p-6 text-center text-sm'>
            {t('No models match your search')}
          </div>
        ) : (
          filteredGroups.map((group) => {
            const versionModels = uniquePricingModelNames(
              group.versions.map((version) => version.model)
            )
            const selectedCount = versionModels.filter((model) =>
              selectedSet.has(model)
            ).length
            const groupSelected =
              versionModels.length > 0 && selectedCount === versionModels.length
            const groupPartial = selectedCount > 0 && !groupSelected
            const isExpanded = expanded.has(group.name) || normalizedKeyword !== ''
            const bestRatio = bestGroupDiscountRatio(group)
            const iconNode = getLobeIcon(
              toColorIconName(group.icon || group.vendor_icon) || 'Bot',
              22
            )

            return (
              <div key={group.name} className='border-b last:border-0'>
                <div className='flex items-center gap-3 bg-background px-3 py-2.5'>
                  <Checkbox
                    checked={groupSelected ? true : groupPartial ? 'indeterminate' : false}
                    onCheckedChange={(checked) => toggleGroup(group, checked === true)}
                    onClick={(event) => event.stopPropagation()}
                    aria-label={t('Select model group')}
                  />
                  <button
                    type='button'
                    className='hover:bg-muted focus-visible:border-ring focus-visible:ring-ring/50 flex size-7 shrink-0 items-center justify-center rounded-md border border-transparent p-0 outline-none transition-colors focus-visible:ring-3'
                    aria-expanded={isExpanded}
                    aria-label={isExpanded ? t('Collapse') : t('Expand')}
                    onClick={(event) => {
                      event.preventDefault()
                      event.stopPropagation()
                      toggleExpanded(group.name)
                    }}
                  >
                    <ChevronDown
                      className={cn(
                        'text-muted-foreground size-4 shrink-0 transition-transform',
                        !isExpanded && '-rotate-90'
                      )}
                    />
                  </button>
                  <div className='flex min-w-0 flex-1 items-center gap-2'>
                    <span className='flex size-8 shrink-0 items-center justify-center rounded-md border bg-background'>
                      {iconNode}
                    </span>
                    <div className='min-w-0 flex-1'>
                      <div className='flex min-w-0 flex-wrap items-center gap-2'>
                        <span className='truncate text-sm font-semibold'>
                          {group.display_name || group.name}
                        </span>
                        {group.vendor_name && (
                          <span className='text-muted-foreground text-[11px]'>
                            {group.vendor_name}
                          </span>
                        )}
                        <span className='rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground'>
                          {t('{{count}} versions', {
                            count: group.versions.length,
                          })}
                        </span>
                        {bestRatio && bestRatio < 1 && (
                          <span className='rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'>
                            {(bestRatio * 10).toFixed(1)}折
                          </span>
                        )}
                      </div>
                      <div className='text-muted-foreground mt-0.5 truncate text-[11px]'>
                        {group.name}
                      </div>
                    </div>
                  </div>
                </div>
                {isExpanded && (
                  <div className='divide-y bg-muted/10'>
                    {group.versions.map((version) => {
                      const checked = selectedSet.has(version.model)
                      return (
                        <div
                          key={version.model}
                          role='button'
                          tabIndex={0}
                          className={cn(
                            'grid w-full cursor-pointer grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-4 py-2 text-left transition-colors hover:bg-muted/40 focus-visible:bg-muted/40 focus-visible:outline-none sm:grid-cols-[auto_minmax(0,1fr)_minmax(150px,auto)]',
                            checked && 'bg-primary/5'
                          )}
                          onClick={() => toggleVersion(version.model)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter' || event.key === ' ') {
                              event.preventDefault()
                              toggleVersion(version.model)
                            }
                          }}
                        >
                          <Checkbox
                            checked={checked}
                            onCheckedChange={(nextChecked) => {
                              const modelSet = new Set(selected)
                              if (nextChecked === true) {
                                modelSet.add(version.model)
                              } else {
                                modelSet.delete(version.model)
                              }
                              onChange([...modelSet])
                            }}
                            onClick={(event) => event.stopPropagation()}
                            aria-label={version.model}
                          />
                          <div className='min-w-0'>
                            <div className='truncate font-mono text-xs'>
                              {version.model}
                            </div>
                            <div className='mt-1 flex flex-wrap gap-1'>
                              {(version.channel_tags?.length
                                ? version.channel_tags
                                : version.channel_name
                                  ? [version.channel_name]
                                  : []
                              ).map((tag) => (
                                <span
                                  key={tag}
                                  className='rounded bg-sky-50 px-1.5 py-0.5 text-[10px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300'
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          </div>
                          <div className='hidden sm:block'>
                            <PricingModelPrice model={version} />
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}

function bestGroupDiscountRatio(group: SelectablePricingModelGroup) {
  const ratios = group.versions
    .map((version) => version.discount_ratio)
    .filter((value): value is number => typeof value === 'number' && value > 0)
  if (ratios.length === 0) return undefined
  return Math.min(...ratios)
}

function uniquePricingModelNames(models: string[]) {
  return [...new Set(models.filter(Boolean))]
}

function getPricingModelGroupVendor(group: SelectablePricingModelGroup) {
  return (group.vendor_name || group.vendor || '').trim()
}

function comparePricingModelGroups(
  a: SelectablePricingModelGroup,
  b: SelectablePricingModelGroup
) {
  const vendorCompare = pricingModelVendorSortKey(a).localeCompare(
    pricingModelVendorSortKey(b)
  )
  if (vendorCompare !== 0) return vendorCompare

  const releaseCompare =
    pricingModelReleaseSortValue(b) - pricingModelReleaseSortValue(a)
  if (releaseCompare !== 0) return releaseCompare

  return (a.display_name || a.name).localeCompare(b.display_name || b.name)
}

function pricingModelVendorSortKey(group: SelectablePricingModelGroup) {
  return (
    getPricingModelGroupVendor(group).toLowerCase() ||
    (group.vendor_id ? String(group.vendor_id).padStart(8, '0') : '') ||
    group.name.toLowerCase()
  )
}

function comparePricingModelVersions(
  a: SelectablePricingModelVersion,
  b: SelectablePricingModelVersion
) {
  const releaseCompare =
    pricingModelNameReleaseSortValue(b.model) -
    pricingModelNameReleaseSortValue(a.model)
  if (releaseCompare !== 0) return releaseCompare
  return a.model.localeCompare(b.model)
}

function pricingModelReleaseSortValue(group: SelectablePricingModelGroup) {
  const names = [
    group.name,
    group.display_name,
    ...group.versions.map((version) => version.model),
  ]
  return Math.max(...names.map(pricingModelNameReleaseSortValue), 0)
}

const pricingModelDatePattern =
  /(?:^|[^0-9])((?:20\d{2})(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01]))(?:[^0-9]|$)/
const pricingModelVersionPattern =
  /(?:^|[^0-9])(\d+)(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[^0-9]|$)/g

function pricingModelNameReleaseSortValue(name: string) {
  const dateMatch = name.match(pricingModelDatePattern)
  if (dateMatch?.[1]) {
    return Number(dateMatch[1])
  }

  let best = 0
  for (const match of name.toLowerCase().matchAll(pricingModelVersionPattern)) {
    let value = 0
    for (let index = 1; index <= 4; index++) {
      const rawPart = match[index]
      const part = rawPart ? Number(rawPart) : 0
      if (index > 1 && rawPart && rawPart.length > 2) {
        value = 0
        break
      }
      if (index === 1 && part >= 2000) {
        value = 0
        break
      }
      value = value * 1000 + part
    }
    best = Math.max(best, value)
  }

  return best > 0 ? 10_000_000_000 + best : 0
}

type ApiKeyFormSectionProps = {
  title: string
  description: string
  icon: LucideIcon
  children: ReactNode
}

function ApiKeyFormSection(props: ApiKeyFormSectionProps) {
  const Icon = props.icon

  return (
    <section className='bg-card rounded-lg border'>
      <div className='flex items-center gap-2.5 border-b px-3 py-2.5 sm:gap-3 sm:px-4 sm:py-3'>
        <div className='bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-lg border sm:size-10'>
          <Icon className='size-4 sm:size-5' />
        </div>
        <div className='min-w-0'>
          <h3 className='text-sm leading-none font-medium'>{props.title}</h3>
          <p className='text-muted-foreground mt-0.5 text-xs sm:mt-1'>
            {props.description}
          </p>
        </div>
      </div>
      <div className='space-y-3 p-3 sm:space-y-4 sm:p-4'>{props.children}</div>
    </section>
  )
}

export function ApiKeysMutateDrawer({
  open,
  onOpenChange,
  currentRow,
  side = 'right',
}: ApiKeyMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = useApiKeys()
  const { status } = useStatus()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const defaultUseAutoGroup = status?.default_use_auto_group === true

  // Fetch models
  const { data: modelsData } = useQuery({
    queryKey: ['user-models'],
    queryFn: getUserModels,
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  })

  // Fetch selectable pricing models (platform or enterprise sheet models)
  const { data: pricingModelsData } = useQuery({
    queryKey: ['available-pricing-models'],
    queryFn: getAvailablePricingModels,
    staleTime: 5 * 60 * 1000,
  })

  const { data: groupedPricingModelsData } = useQuery({
    queryKey: ['available-pricing-models-grouped'],
    queryFn: getGroupedAvailablePricingModels,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  const selectableModels: SelectablePricingModel[] =
    pricingModelsData?.data || []
  const groupedSelectableModels: SelectablePricingModelGroup[] =
    groupedPricingModelsData?.data?.models || []
  const hasGroupedPricingModels =
    groupedPricingModelsData?.success === true && groupedSelectableModels.length > 0
  const hasPricingSheet = selectableModels.length > 0
  const pricingSheetName =
    selectableModels[0]?.sheet_name || ''
  const pricingSheetSource = selectableModels[0]?.source || ''

  // Fetch groups
  const { data: groupsData } = useQuery({
    queryKey: ['user-groups'],
    queryFn: getUserGroups,
    staleTime: 5 * 60 * 1000,
  })

  const models = modelsData?.data || []
  const groupsRaw = groupsData?.data || {}
  const groups: ApiKeyGroupOption[] = Object.entries(groupsRaw).map(
    ([key, info]) => ({
      value: key,
      label: key,
      desc: info.desc || key,
      ratio: info.ratio,
    })
  )
  const backendHasAuto = groups.some((g) => g.value === 'auto')

  const form = useForm<ApiKeyFormValues>({
    resolver: zodResolver(apiKeyFormSchema),
    defaultValues: getApiKeyFormDefaultValues(defaultUseAutoGroup),
  })

  // Load existing data when updating
  useEffect(() => {
    if (open && isUpdate && currentRow) {
      getApiKey(currentRow.id).then((apiKeyResult) => {
        if (apiKeyResult.success && apiKeyResult.data) {
          form.reset(transformApiKeyToFormDefaults(apiKeyResult.data))
        }
      })
    } else if (open && !isUpdate) {
      form.reset(getApiKeyFormDefaultValues(defaultUseAutoGroup && backendHasAuto))
    }
  }, [open, isUpdate, currentRow, form, defaultUseAutoGroup, backendHasAuto])

  // Auto-expand advanced settings when pricing sheet is available (create or edit mode)
  useEffect(() => {
    if (open && hasPricingSheet) {
      setAdvancedOpen(true)
    }
  }, [open, hasPricingSheet])

  // When pricing sheet exists, force group to 'default' and disable cross_group_retry
  useEffect(() => {
    if (hasPricingSheet) {
      form.setValue('group', 'default')
      form.setValue('cross_group_retry', false)
    }
  }, [hasPricingSheet, form])

  // Correct group after groups load: if the form value is not in available groups, fall back
  useEffect(() => {
    if (groups.length === 0) return
    const currentGroup = form.getValues('group')
    if (currentGroup && !groups.some((g) => g.value === currentGroup)) {
      const fallback = groups.find((g) => g.value === 'default')?.value ?? groups[0]?.value ?? ''
      form.setValue('group', fallback)
      if (currentGroup === 'auto') {
        form.setValue('cross_group_retry', false)
      }
    }
  }, [groups, form])

  const buildSelectModels = (modelLimits: string[]): SelectModel[] =>
    (modelLimits || [])
      .map((model) => {
        const pricingModel = selectableModels.find((m) => m.model === model)
        if (pricingModel) {
          return { model, pricing_sheet_id: pricingModel.sheet_id }
        }
        const groupedVersion = groupedSelectableModels
          .flatMap((group) => group.versions)
          .find((version) => version.model === model)
        const sheetId =
          groupedVersion?.pricing_sheet_id ?? groupedVersion?.sheet_id
        return sheetId ? { model, pricing_sheet_id: sheetId } : null
      })
      .filter((x): x is SelectModel => x !== null)

  const onSubmit = async (data: ApiKeyFormValues) => {
    console.log('[api-keys-mutate-drawer] onSubmit called, isUpdate:', isUpdate)
    setIsSubmitting(true)
    try {
      const basePayload = transformFormDataToPayload(data)
      const selectModels = buildSelectModels(data.model_limits)

      if (isUpdate && currentRow) {
        const result = await updateApiKey({
          ...basePayload,
          id: currentRow.id,
          select_models: selectModels,
        })
        if (result.success) {
          toast.success(t(SUCCESS_MESSAGES.API_KEY_UPDATED))
          onOpenChange(false)
          triggerRefresh()
        } else {
          toast.error(result.message || t(ERROR_MESSAGES.UPDATE_FAILED))
        }
      } else {
        // Create mode - handle batch creation
        const count = data.tokenCount || 1
        let successCount = 0

        for (let i = 0; i < count; i++) {
          const result = await createApiKey({
            ...basePayload,
            name:
              i === 0 && data.name
                ? data.name
                : `${data.name || 'default'}-${Math.random().toString(36).slice(2, 8)}`,
            select_models: selectModels,
          })
          if (result.success) {
            successCount++
          } else {
            toast.error(result.message || t(ERROR_MESSAGES.CREATE_FAILED))
            break
          }
        }

        if (successCount > 0) {
          toast.success(
            t('Successfully created {{count}} API Key(s)', {
              count: successCount,
            })
          )
          onOpenChange(false)
          triggerRefresh()
        }
      }
    } catch (_error) {
      console.error('[api-keys-mutate-drawer] onSubmit error:', _error)
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleSetExpiry = (months: number, days: number, hours: number) => {
    if (months === 0 && days === 0 && hours === 0) {
      form.setValue('expired_time', undefined)
      return
    }

    const now = new Date()
    now.setMonth(now.getMonth() + months)
    now.setDate(now.getDate() + days)
    now.setHours(now.getHours() + hours)

    form.setValue('expired_time', now)
  }

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'
  const quotaLabel = t('Quota ({{currency}})', { currency: currencyLabel })
  const quotaPlaceholder = tokensOnly
    ? t('Enter quota in tokens')
    : t('Enter quota in {{currency}}', { currency: currencyLabel })
  const selectedGroup = form.watch('group')
  const unlimitedQuota = form.watch('unlimited_quota')

  // When unlimited quota is enabled, reset daily/monthly limits to 0
  useEffect(() => {
    if (unlimitedQuota) {
      form.setValue('quota_limit_daily', 0, { shouldValidate: false })
      form.setValue('quota_limit_monthly', 0, { shouldValidate: false })
    }
  }, [unlimitedQuota, form])

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v)
        if (!v) {
          form.reset()
          setPendingBindingModels([])
        }
      }}
    >
      <SheetContent
        side={side}
        className='bg-background flex !h-dvh !w-screen max-w-none gap-0 overflow-hidden p-0 sm:!w-full sm:!max-w-[860px] xl:!max-w-[980px]'
      >
        <SheetHeader className='bg-background border-b px-4 py-3 text-start sm:px-5 sm:py-4'>
          <SheetTitle className='text-base sm:text-lg'>
            {isUpdate ? t('Update API Key') : t('Create API Key')}
          </SheetTitle>
          <SheetDescription className='pr-6 text-xs sm:text-sm'>
            {isUpdate
              ? t('Update the API key by providing necessary info.')
              : t('Add a new API key by providing necessary info.')}{' '}
            {t("Click save when you're done.")}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='api-key-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className='min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain px-3 py-3 sm:space-y-4 sm:px-4 sm:py-4'
          >
            <ApiKeyFormSection
              title={t('Basic Information')}
              description={t('Set API key basic information')}
              icon={KeyRound}
            >
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Name')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('Enter a name')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!hasPricingSheet && (
                <FormField
                  control={form.control}
                  name='group'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Group')}</FormLabel>
                      <FormControl>
                        <ApiKeyGroupCombobox
                          options={groups}
                          value={field.value}
                          onValueChange={field.onChange}
                          placeholder={t('Select a group')}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {selectedGroup === 'auto' && !hasPricingSheet && (
                <FormField
                  control={form.control}
                  name='cross_group_retry'
                  render={({ field }) => (
                    <FormItem className='flex min-h-16 flex-row items-center justify-between gap-3 rounded-lg border px-3 py-2.5 sm:min-h-20 sm:gap-4 sm:px-4 sm:py-3'>
                      <div className='space-y-0.5'>
                        <FormLabel className='text-sm'>
                          {t('Cross-group retry')}
                        </FormLabel>
                        <FormDescription className='line-clamp-2 text-xs sm:line-clamp-none'>
                          {t(
                            'When enabled, if channels in the current group fail, it will try channels in the next group in order.'
                          )}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={!!field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              )}

              <FormField
                control={form.control}
                name='expired_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Expiration Time')}</FormLabel>
                    <div className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center'>
                      <FormControl>
                        <DateTimePicker
                          value={field.value}
                          onChange={field.onChange}
                          placeholder={t('Never expires')}
                          className='min-w-0 [&_input[type=time]]:w-24 sm:[&_input[type=time]]:w-32'
                        />
                      </FormControl>
                      <div className='grid grid-cols-4 gap-2 sm:flex'>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          className='px-2 text-xs sm:px-3 sm:text-sm'
                          onClick={() => handleSetExpiry(0, 0, 0)}
                        >
                          {t('Never')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          className='px-2 text-xs sm:px-3 sm:text-sm'
                          onClick={() => handleSetExpiry(1, 0, 0)}
                        >
                          {t('1 Month')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          className='px-2 text-xs sm:px-3 sm:text-sm'
                          onClick={() => handleSetExpiry(0, 1, 0)}
                        >
                          {t('1 Day')}
                        </Button>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          className='px-2 text-xs sm:px-3 sm:text-sm'
                          onClick={() => handleSetExpiry(0, 0, 1)}
                        >
                          {t('1 Hour')}
                        </Button>
                      </div>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!isUpdate && (
                <FormField
                  control={form.control}
                  name='tokenCount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Quantity')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min='1'
                          placeholder={t('Number of keys to create')}
                          onChange={(e) =>
                            field.onChange(parseInt(e.target.value, 10) || 1)
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Create multiple API keys at once (random suffix will be added to names)'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </ApiKeyFormSection>

            <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
              <section className='bg-card rounded-lg border'>
                <CollapsibleTrigger
                  render={
                    <button
                      type='button'
                      className='hover:bg-muted/50 flex w-full items-center gap-2.5 px-3 py-2.5 text-left transition-colors sm:gap-3 sm:px-4 sm:py-3'
                    />
                  }
                >
                  <div className='bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-lg border sm:size-10'>
                    <Settings2 className='size-4 sm:size-5' />
                  </div>
                  <div className='min-w-0 flex-1'>
                    <h3 className='text-sm leading-none font-medium'>
                      {t('Advanced Settings')}
                    </h3>
                    <p className='text-muted-foreground mt-1 text-xs'>
                      {t('Set API key access restrictions')}
                    </p>
                  </div>
                  <ChevronDown
                    className={cn(
                      'text-muted-foreground size-4 shrink-0 transition-transform',
                      advancedOpen && 'rotate-180'
                    )}
                  />
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <div className='space-y-3 border-t p-3 sm:space-y-4 sm:p-4'>
                    {hasPricingSheet ? (
                      <FormField
                        control={form.control}
                        name='model_limits'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t('Available Models')}
                            </FormLabel>
                            <FormDescription>
                              {t(
                                'Select models from the pricing sheet. Unselected models will not be accessible with this key.'
                              )}
                            </FormDescription>
                            <FormControl>
                              {hasGroupedPricingModels ? (
                                <GroupedPricingModelSelector
                                  groups={groupedSelectableModels}
                                  selected={field.value || []}
                                  onChange={field.onChange}
                                />
                              ) : (
                                <div className='mt-2 overflow-hidden rounded-lg border'>
                                  <table className='w-full text-xs'>
                                    <thead>
                                      <tr className='border-b bg-muted/50'>
                                        <th className='px-3 py-2 text-left font-medium'>
                                          {t('Model')}
                                        </th>
                                        <th className='px-3 py-2 text-right font-medium'>
                                          {t('Price')}
                                        </th>
                                        <th className='w-10 px-3 py-2 text-center font-medium'>
                                          <Checkbox
                                            checked={
                                              field.value.length === selectableModels.length &&
                                              selectableModels.length > 0
                                                ? true
                                                : field.value.length > 0 &&
                                                    field.value.length < selectableModels.length
                                                  ? 'indeterminate'
                                                  : false
                                            }
                                            onCheckedChange={(checked) => {
                                              field.onChange(
                                                checked === true
                                                  ? selectableModels.map((m) => m.model)
                                                  : []
                                              )
                                            }}
                                            onClick={(e) => e.stopPropagation()}
                                          />
                                        </th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {selectableModels.map((m) => (
                                        <PricingModelRow
                                          key={m.model}
                                          model={m}
                                          fieldValue={field.value || []}
                                          onToggle={(model) => {
                                            const current = field.value || []
                                            if (current.includes(model)) {
                                              field.onChange(
                                                current.filter((v) => v !== model)
                                              )
                                            } else {
                                              field.onChange([...current, model])
                                            }
                                          }}
                                        />
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    ) : (
                      <FormField
                        control={form.control}
                        name='model_limits'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Model Limits')}</FormLabel>
                            <FormControl>
                              <MultiSelect
                                options={models.map((m) => ({
                                  label: m,
                                  value: m,
                                }))}
                                selected={field.value}
                                onChange={field.onChange}
                                placeholder={t(
                                  'Select models (empty for allow all)'
                                )}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                'Limit which models can be used with this key'
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    )}

                    <FormField
                      control={form.control}
                      name='allow_ips'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('IP Whitelist (supports CIDR)')}
                          </FormLabel>
                          <FormControl>
                            <Textarea
                              {...field}
                              className='min-h-20 resize-none'
                              placeholder={t(
                                'One IP per line (empty for no restriction)'
                              )}
                              rows={3}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              'Do not over-trust this feature. IP may be spoofed. Please use with nginx, CDN and other gateways.'
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </CollapsibleContent>
              </section>
            </Collapsible>

            <ApiKeyFormSection
              title={t('Quota Settings')}
              description={t('Set quota amount and limits')}
              icon={WalletCards}
            >
              {!unlimitedQuota && (
                <FormField
                  control={form.control}
                  name='remain_quota_dollars'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Totle Quota Limit')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step={tokensOnly ? 1 : 0.01}
                          placeholder={quotaPlaceholder}
                          onChange={(e) =>
                            field.onChange(parseFloat(e.target.value) || 0)
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {tokensOnly
                          ? t('Enter the quota amount in tokens')
                          : t('Enter the quota amount in {{currency}}', {
                              currency: currencyLabel,
                            })}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              <FormField
                control={form.control}
                name='quota_limit_daily'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Daily Quota Limit')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='number'
                        step={1}
                        min={0}
                        placeholder={t('0 = unlimited')}
                        disabled={unlimitedQuota}
                        onChange={(e) =>
                          field.onChange(parseInt(e.target.value, 10) || 0)
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Used: {{used}} / Remaining: {{remaining}}', {
                        used: formatCurrencyUSD(form.watch('quota_used_daily') ?? 0),
                        remaining: formatCurrencyUSD(
                          Math.max(0, (form.watch('quota_limit_daily') ?? 0) - (form.watch('quota_used_daily') ?? 0))
                        ),
                      })}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='quota_limit_monthly'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Monthly Quota Limit')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='number'
                        step={1}
                        min={0}
                        placeholder={t('0 = unlimited')}
                        disabled={unlimitedQuota}
                        onChange={(e) =>
                          field.onChange(parseInt(e.target.value, 10) || 0)
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Used: {{used}} / Remaining: {{remaining}}', {
                        used: formatCurrencyUSD(form.watch('quota_used_monthly') ?? 0),
                        remaining: formatCurrencyUSD(
                          Math.max(0, (form.watch('quota_limit_monthly') ?? 0) - (form.watch('quota_used_monthly') ?? 0))
                        ),
                      })}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='unlimited_quota'
                render={({ field }) => (
                  <FormItem className='flex min-h-16 flex-row items-center justify-between gap-3 rounded-lg border px-3 py-2.5 sm:min-h-20 sm:gap-4 sm:px-4 sm:py-3'>
                    <div className='space-y-0.5'>
                      <FormLabel className='text-sm'>
                        {t('Unlimited Quota')}
                      </FormLabel>
                      <FormDescription className='text-xs'>
                        {t('Enable unlimited quota for this API key')}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            </ApiKeyFormSection>

          </form>
        </Form>
        <SheetFooter className='bg-background grid grid-cols-2 gap-2 border-t px-3 py-3 sm:flex sm:flex-row sm:justify-end sm:px-5 sm:py-4'>
          <SheetClose
            render={<Button variant='outline' className='w-full sm:w-auto' />}
          >
            {t('Close')}
          </SheetClose>
          <Button
            form='api-key-form'
            type='submit'
            disabled={isSubmitting}
            className='w-full sm:w-auto'
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
