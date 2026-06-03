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
import { memo, useEffect, useState, type ReactNode } from 'react'
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
  getTokenPricingModels,
} from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  apiKeyFormSchema,
  type ApiKeyFormValues,
  getApiKeyFormDefaultValues,
  transformFormDataToPayload,
  transformApiKeyToFormDefaults,
} from '../lib'
import { type ApiKey, type SelectablePricingModel, type PricingBinding, type SelectModel } from '../types'
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
  const hasDiscount = model.discount_ratio > 0 && model.discount_ratio < 1
  const inputOriginal = model.input_original_price ?? 0
  const outputOriginal = model.output_original_price ?? 0
  const inputDiscounted = model.input_discounted_price ?? 0
  const outputDiscounted = model.output_discounted_price ?? 0
  const hasPrice = inputOriginal > 0 || outputOriginal > 0
  const isFixedPrice = model.quota_type === 1
  const discountLabel = hasDiscount
    ? `${(model.discount_ratio * 10).toFixed(1)}折`
    : ''
  const unitLabel = isFixedPrice ? t('per request') : '/ 1M tokens'

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
        <div className='flex flex-col items-end gap-0.5'>
          {hasDiscount && (
            <span className='inline-flex items-center rounded bg-gradient-to-r from-amber-400 to-orange-400 px-1.5 py-0.5 text-[10px] font-bold text-white leading-none'>
              {discountLabel}
            </span>
          )}
          <span className='font-mono text-[11px] tabular-nums leading-tight'>
            {hasPrice ? (
              <>
                {inputOriginal > 0 && (
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
                <span className='text-foreground mx-1'>→</span>
                <span className='font-bold text-foreground'>
                  ${inputDiscounted.toFixed(4)}
                </span>
                {outputOriginal > 0 && (
                  <>
                    <span className='text-muted-foreground/40 mx-0.5'>/</span>
                    <span className='font-bold text-foreground'>
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
  const [pendingBindingModels, setPendingBindingModels] = useState<string[]>([])
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

  const selectableModels: SelectablePricingModel[] =
    pricingModelsData?.data || []
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
      Promise.all([
        getApiKey(currentRow.id),
        getTokenPricingModels(currentRow.id),
      ]).then(([apiKeyResult, bindingsResult]) => {
        if (apiKeyResult.success && apiKeyResult.data) {
          form.reset(transformApiKeyToFormDefaults(apiKeyResult.data))
        }
        if (bindingsResult.success && bindingsResult.data?.bindings) {
          const bindingModels = bindingsResult.data.bindings.map(
            (b: PricingBinding) => b.model
          )
          setPendingBindingModels(bindingModels)
        }
      })
    } else if (open && !isUpdate) {
      form.reset(getApiKeyFormDefaultValues(defaultUseAutoGroup && backendHasAuto))
    }
    return () => {
      setPendingBindingModels([])
    }
  }, [open, isUpdate, currentRow, form, defaultUseAutoGroup, backendHasAuto])

  // Auto-expand advanced settings when pricing sheet is available (create or edit mode)
  useEffect(() => {
    if (open && hasPricingSheet) {
      setAdvancedOpen(true)
    }
  }, [open, hasPricingSheet])

  // When pricing sheet exists, force group to 'default' and disable cross_group_retry
  // Also write pending binding models into model_limits once pricing sheet is ready
  useEffect(() => {
    if (hasPricingSheet) {
      form.setValue('group', 'default')
      form.setValue('cross_group_retry', false)
    }
    // Write pending binding models only after pricing sheet is available (so we know which selector to show)
    if (hasPricingSheet && pendingBindingModels.length > 0) {
      form.setValue('model_limits', pendingBindingModels)
    }
  }, [hasPricingSheet, form, pendingBindingModels])

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
        return pricingModel
          ? { model, pricing_sheet_id: pricingModel.sheet_id }
          : null
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
        className='bg-background flex !h-dvh !w-screen max-w-none gap-0 overflow-hidden p-0 sm:!w-full sm:!max-w-[620px]'
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
                                      <th className='px-3 py-2 text-center font-medium w-10'>
                                        <Checkbox
                                          checked={field.value.length === selectableModels.length && selectableModels.length > 0}
                                          indeterminate={field.value.length > 0 && field.value.length < selectableModels.length}
                                          onCheckedChange={(checked) => {
                                            if (checked) {
                                              field.onChange(selectableModels.map((m) => m.model))
                                            } else {
                                              field.onChange([])
                                            }
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
                                            field.onChange(current.filter((v) => v !== model))
                                          } else {
                                            field.onChange([...current, model])
                                          }
                                        }}
                                      />
                                    ))}
                                  </tbody>
                                </table>
                              </div>
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
                      <FormLabel>{quotaLabel}</FormLabel>
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
