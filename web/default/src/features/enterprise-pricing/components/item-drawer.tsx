import { useEffect, useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
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
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ComboboxInput, type ComboboxInputOption } from '@/components/ui/combobox-input'
import { getPricingItems, createPricingItem, updatePricingItem, getEnabledModels } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES, DISCOUNT_TYPE, getDiscountTypeOptions } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'

const itemFormSchema = z.object({
  model: z.string().min(1, 'Model is required'),
  discount_type: z.enum(['ratio', 'fixed_price', 'per_call']),
  discount_value: z.number().min(0, 'Value must be non-negative'),
  remark: z.string().optional(),
})

type ItemFormValues = z.infer<typeof itemFormSchema>

const DEFAULT_VALUES: ItemFormValues = {
  model: '',
  discount_type: DISCOUNT_TYPE.RATIO,
  discount_value: 1,
  remark: '',
}

type ItemDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentItemId: number | null
  sheetId: number | null
}

export function ItemDrawer({
  open,
  onOpenChange,
  currentItemId,
  sheetId,
}: ItemDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentItemId
  const { triggerItemRefresh } = useEnterprisePricing()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<ItemFormValues>({
    resolver: zodResolver(itemFormSchema),
    defaultValues: DEFAULT_VALUES,
  })

  // Fetch enabled models from the system (used for billing)
  const { data: enabledModels } = useQuery({
    queryKey: ['enabled-models'],
    queryFn: getEnabledModels,
    staleTime: 10 * 60 * 1000, // 10 minutes
  })

  const modelOptions: ComboboxInputOption[] = useMemo(() => {
    return (enabledModels || []).map((model) => ({
      value: model,
      label: model,
    }))
  }, [enabledModels])

  // Fetch item data when updating
  const { data: itemsData } = useQuery({
    queryKey: ['pricing-items', sheetId],
    queryFn: async () => {
      if (!sheetId) return []
      const result = await getPricingItems(sheetId)
      return result.data || []
    },
    enabled: !!sheetId,
  })

  useEffect(() => {
    if (open && isUpdate && currentItemId && itemsData) {
      const item = itemsData.find((i) => i.id === currentItemId)
      if (item) {
        form.reset({
          model: item.model,
          discount_type: item.discount_type,
          discount_value: item.discount_value,
          remark: item.remark || '',
        })
      }
    } else if (open && !isUpdate) {
      form.reset(DEFAULT_VALUES)
    }
  }, [open, isUpdate, currentItemId, itemsData, form])

  const onSubmit = async (data: ItemFormValues) => {
    if (!sheetId) return
    setIsSubmitting(true)
    try {
      const payload = {
        model: data.model,
        discount_type: data.discount_type,
        discount_value: data.discount_value,
        remark: data.remark || undefined,
      }

      const result = isUpdate
        ? await updatePricingItem(sheetId, { ...payload, id: currentItemId! })
        : await createPricingItem(sheetId, payload)

      if (result.success) {
        toast.success(
          isUpdate ? t(SUCCESS_MESSAGES.UPDATED) : t(SUCCESS_MESSAGES.CREATED)
        )
        onOpenChange(false)
        triggerItemRefresh()
      } else {
        toast.error(
          result.message ||
            (isUpdate ? t(ERROR_MESSAGES.UPDATE_FAILED) : t(ERROR_MESSAGES.CREATE_FAILED))
        )
      }
    } catch (_error) {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <Sheet
        open={open}
        onOpenChange={(v) => {
          onOpenChange(v)
          if (!v) form.reset()
        }}
      >
        <SheetContent className='flex h-dvh w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[500px]'>
          <SheetHeader className='border-b px-4 py-3 text-start sm:px-6 sm:py-4'>
            <SheetTitle>
              {isUpdate ? t('Update') : t('Create')} {t('Pricing Item')}
            </SheetTitle>
            <SheetDescription>
              {isUpdate
                ? t('Update the pricing item.')
                : t('Add a new model discount to this pricing sheet.')}
            </SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form
              id='item-form'
              onSubmit={form.handleSubmit(onSubmit)}
              className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'
            >
              <FormField
                control={form.control}
                name='model'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Model')}</FormLabel>
                    <FormControl>
                      <ComboboxInput
                        options={modelOptions}
                        value={field.value}
                        onValueChange={field.onChange}
                        placeholder={t('Search or select model...')}
                        emptyText={t('No model found')}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Select a model to apply this discount.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='discount_type'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Discount Type')}</FormLabel>
                    <Select
                      items={getDiscountTypeOptions(t)}
                      onValueChange={(value) =>
                        field.onChange(value as 'ratio' | 'fixed_price' | 'per_call')
                      }
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select type')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {getDiscountTypeOptions(t).map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='discount_value'
                render={({ field }) => {
                  const discountType = form.watch('discount_type')
                  return (
                    <FormItem>
                      <FormLabel>
                        {discountType === DISCOUNT_TYPE.RATIO
                          ? t('Discount Ratio')
                          : discountType === DISCOUNT_TYPE.PER_CALL
                          ? t('Price per Call')
                          : t('Fixed Price')}
                      </FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          step='0.01'
                          {...field}
                          onChange={(e) =>
                            field.onChange(parseFloat(e.target.value) || 0)
                          }
                          placeholder={
                            discountType === DISCOUNT_TYPE.RATIO
                              ? 'e.g., 0.8 for 20% off'
                              : discountType === DISCOUNT_TYPE.PER_CALL
                              ? 'e.g., 0.05'
                              : 'e.g., 0.001'
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {discountType === DISCOUNT_TYPE.RATIO
                          ? t('Enter ratio (e.g., 0.8 = 80% of original price)')
                          : discountType === DISCOUNT_TYPE.PER_CALL
                          ? t('Fixed price per API call (USD). Bypasses model token billing.')
                          : t('Enter fixed price per unit')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )
                }}
              />

              <FormField
                control={form.control}
                name='remark'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Remark')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('Optional remark')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </form>
          </Form>
          <SheetFooter className='grid grid-cols-2 gap-2 border-t px-4 py-3 sm:flex sm:px-6 sm:py-4'>
            <SheetClose render={<Button variant='outline' />}>
              {t('Close')}
            </SheetClose>
            <Button
              form='item-form'
              type='submit'
              disabled={isSubmitting || !sheetId}
            >
              {isSubmitting ? t('Saving...') : t('Save changes')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}
