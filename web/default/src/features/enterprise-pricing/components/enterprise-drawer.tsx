import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { createEnterprise, updateEnterprise, getEnterprise } from '../api'
import {
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  ENTERPRISE_STATUS,
  getEnterpriseStatusOptions,
} from '../constants'
import type { Enterprise } from '../types'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { z } from 'zod'

const enterpriseFormSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  status: z.number(),
  remark: z.string().optional(),
})

type EnterpriseFormValues = z.infer<typeof enterpriseFormSchema>

const DEFAULT_VALUES: EnterpriseFormValues = {
  name: '',
  status: ENTERPRISE_STATUS.ENABLED,
  remark: '',
}

type EnterpriseDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: Enterprise
}

export function EnterpriseDrawer({
  open,
  onOpenChange,
  currentRow,
}: EnterpriseDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = useEnterprisePricing()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<EnterpriseFormValues>({
    resolver: zodResolver(enterpriseFormSchema),
    defaultValues: DEFAULT_VALUES,
  })

  useEffect(() => {
    if (open && isUpdate && currentRow) {
      getEnterprise(currentRow.id).then((result) => {
        if (result.success && result.data) {
          form.reset({
            name: result.data.name,
            status: result.data.status,
            remark: result.data.remark || '',
          })
        }
      })
    } else if (open && !isUpdate) {
      form.reset(DEFAULT_VALUES)
    }
  }, [open, isUpdate, currentRow, form])

  const onSubmit = async (data: EnterpriseFormValues) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        status: data.status,
        remark: data.remark || undefined,
      }

      const result = isUpdate
        ? await updateEnterprise({ ...payload, id: currentRow!.id })
        : await createEnterprise(payload)

      if (result.success) {
        toast.success(
          isUpdate ? t(SUCCESS_MESSAGES.UPDATED) : t(SUCCESS_MESSAGES.CREATED)
        )
        onOpenChange(false)
        triggerRefresh()
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
              {isUpdate ? t('Update') : t('Create')} {t('Enterprise')}
            </SheetTitle>
            <SheetDescription>
              {isUpdate
                ? t('Update the enterprise by providing necessary info.')
                : t('Add a new enterprise by providing necessary info.')}
            </SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form
              id='enterprise-form'
              onSubmit={form.handleSubmit(onSubmit)}
              className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'
            >
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Enterprise Name')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder={t('Enter enterprise name')}
                      />
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
                      items={getEnterpriseStatusOptions(t)}
                      onValueChange={(value) =>
                        field.onChange(parseInt(value ?? '0'))
                      }
                      value={String(field.value)}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select status')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {getEnterpriseStatusOptions(t).map((opt) => (
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
                name='remark'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Remark')}</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        placeholder={t('Enter remark (optional)')}
                        rows={3}
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
            <Button form='enterprise-form' type='submit' disabled={isSubmitting}>
              {isSubmitting ? t('Saving...') : t('Save changes')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}
