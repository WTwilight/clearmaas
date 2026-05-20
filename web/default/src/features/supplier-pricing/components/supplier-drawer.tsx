import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { createSupplier, updateSupplier, getSupplier } from '../api';
import {
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  SUPPLIER_STATUS,
  SHEET_STATUS_OPTIONS,
  getSheetStatusOptions,
} from '../constants';
import type { Supplier } from '../types';
import { useSupplierPricing } from './supplier-pricing-provider';
import { z } from 'zod';

const supplierFormSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  status: z.number(),
  remark: z.string().optional(),
});

type SupplierFormValues = z.infer<typeof supplierFormSchema>;

const DEFAULT_VALUES: SupplierFormValues = {
  name: '',
  status: SUPPLIER_STATUS.ENABLED,
  remark: '',
};

type SupplierDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentRow?: Supplier;
};

export function SupplierDrawer({
  open,
  onOpenChange,
  currentRow,
}: SupplierDrawerProps) {
  const { t } = useTranslation();
  const isUpdate = !!currentRow;
  const { triggerSupplierRefresh } = useSupplierPricing();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<SupplierFormValues>({
    resolver: zodResolver(supplierFormSchema),
    defaultValues: DEFAULT_VALUES,
  });

  useEffect(() => {
    if (open && isUpdate && currentRow) {
      getSupplier(currentRow.id).then((result) => {
        if (result) {
          form.reset({
            name: result.name,
            status: result.status,
            remark: result.remark || '',
          });
        }
      });
    } else if (open && !isUpdate) {
      form.reset(DEFAULT_VALUES);
    }
  }, [open, isUpdate, currentRow, form]);

  const onSubmit = async (data: SupplierFormValues) => {
    setIsSubmitting(true);
    try {
      const payload = {
        name: data.name,
        status: data.status,
        remark: data.remark || undefined,
      };

      const result = isUpdate
        ? await updateSupplier(currentRow!.id, payload)
        : await createSupplier(payload);

      if (result) {
        toast.success(
          t(isUpdate ? SUCCESS_MESSAGES.UPDATED : SUCCESS_MESSAGES.CREATED)
        );
        onOpenChange(false);
        triggerSupplierRefresh();
      } else {
        toast.error(
          isUpdate ? t(ERROR_MESSAGES.UPDATE_FAILED) : t(ERROR_MESSAGES.CREATE_FAILED)
        );
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v);
        if (!v) form.reset();
      }}
    >
      <SheetContent className='flex h-dvh w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[500px]'>
        <SheetHeader className='border-b px-4 py-3 text-start sm:px-6 sm:py-4'>
          <SheetTitle>
            {isUpdate ? t('Edit Supplier') : t('Add Supplier')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the supplier by providing necessary info.')
              : t('Add a new supplier by providing necessary info.')}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='supplier-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'
          >
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Supplier Name')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder={t('Enter supplier name')}
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
                    onValueChange={(value) => field.onChange(parseInt(value))}
                    value={String(field.value)}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue>
                          {SHEET_STATUS_OPTIONS.find((o) => o.value === field.value)?.label ?? '-'}
                        </SelectValue>
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {SHEET_STATUS_OPTIONS.map((opt) => (
                          <SelectItem key={opt.value} value={String(opt.value)}>
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
          <Button form='supplier-form' type='submit' disabled={isSubmitting}>
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
