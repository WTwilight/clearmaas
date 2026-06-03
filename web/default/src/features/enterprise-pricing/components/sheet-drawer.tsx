import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
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
import { createSheet, updateSheet, getSheet } from '../api'
import {
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  SHEET_STATUS,
  getSheetStatusOptions,
} from '../constants'
import type { PricingSheet } from '../types'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { z } from 'zod'

const sheetFormSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  status: z.number(),
  start_time: z.number().optional(),
  end_time: z.number().optional(),
})

type SheetFormValues = z.infer<typeof sheetFormSchema>

const DEFAULT_VALUES: SheetFormValues = {
  name: '',
  status: SHEET_STATUS.ACTIVE,
  start_time: undefined,
  end_time: undefined,
}

type SheetDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: PricingSheet
}

export function SheetDrawer({
  open,
  onOpenChange,
  currentRow,
}: SheetDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerSheetRefresh, selectedEnterpriseId } = useEnterprisePricing()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<SheetFormValues>({
    resolver: zodResolver(sheetFormSchema),
    defaultValues: DEFAULT_VALUES,
  })

  useEffect(() => {
    if (open && isUpdate && currentRow) {
      getSheet(currentRow.enterprise_id, currentRow.id).then((result) => {
        if (result.success && result.data) {
          form.reset({
            name: result.data.name,
            status: result.data.status,
            start_time: result.data.start_time,
            end_time: result.data.end_time,
          })
        }
      })
    } else if (open && !isUpdate) {
      form.reset(DEFAULT_VALUES)
    }
  }, [open, isUpdate, currentRow, form])

  const onSubmit = async (data: SheetFormValues) => {
    const targetEnterpriseId = isUpdate && currentRow
      ? currentRow.enterprise_id
      : selectedEnterpriseId
    if (!targetEnterpriseId) {
      toast.error(t('Please select an enterprise first'))
      return
    }
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        status: data.status,
        start_time: data.start_time,
        end_time: data.end_time,
      }

      const result = isUpdate
        ? await updateSheet(targetEnterpriseId, { ...payload, id: currentRow!.id })
        : await createSheet(targetEnterpriseId, payload)

      if (result.success) {
        toast.success(
          isUpdate ? t(SUCCESS_MESSAGES.UPDATED) : t(SUCCESS_MESSAGES.CREATED)
        )
        onOpenChange(false)
        triggerSheetRefresh()
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
              {isUpdate ? t('Update') : t('Create')} {t('Pricing Sheet')}
            </SheetTitle>
            <SheetDescription>
              {isUpdate
                ? t('Update the pricing sheet.')
                : t('Add a new pricing sheet.')}
            </SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form
              id='sheet-form'
              onSubmit={form.handleSubmit(onSubmit)}
              className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'
            >
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Sheet Name')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('Enter sheet name')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='status'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Status')}</FormLabel>
                    <Select
                      items={getSheetStatusOptions(t)}
                      onValueChange={(value) => field.onChange(parseInt(value ?? '0'))}
                      value={String(field.value)}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select status')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {getSheetStatusOptions(t).map((opt) => (
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
                name='start_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Start Time')}</FormLabel>
                    <FormControl>
                      <Input
                        type='datetime-local'
                        value={
                          field.value
                            ? new Date(field.value * 1000)
                                .toISOString()
                                .slice(0, 16)
                            : ''
                        }
                        onChange={(e) => {
                          const date = e.target.value
                          field.onChange(date ? Math.floor(new Date(date).getTime() / 1000) : undefined)
                        }}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='end_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('End Time')}</FormLabel>
                    <FormControl>
                      <Input
                        type='datetime-local'
                        value={
                          field.value
                            ? new Date(field.value * 1000)
                                .toISOString()
                                .slice(0, 16)
                            : ''
                        }
                        onChange={(e) => {
                          const date = e.target.value
                          field.onChange(date ? Math.floor(new Date(date).getTime() / 1000) : undefined)
                        }}
                      />
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
            <Button form='sheet-form' type='submit' disabled={isSubmitting}>
              {isSubmitting ? t('Saving...') : t('Save changes')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}
