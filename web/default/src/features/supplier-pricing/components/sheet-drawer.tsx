import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { useQuery } from '@tanstack/react-query';
import { Eye } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Sheet,
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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  createSupplierPricingSheet,
  updateSupplierPricingSheet,
  getSupplierPricingSheet,
  getSuppliers,
} from '../api';
import {
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  SHEET_STATUS,
  SHEET_STATUS_LABELS,
  getSheetStatusOptions,
} from '../constants';
import type { SupplierPricingSheet } from '../types';
import { useSupplierPricing } from './supplier-pricing-provider';
import { z } from 'zod';
import { formatTimestamp, formatTimestampForInput, parseTimestampFromInput } from '@/lib/format';

const sheetFormSchema = z.object({
  supplier_id: z.number().optional(),
  name: z.string().min(1, 'Name is required'),
  status: z.number(),
  start_time: z.number().optional(),
  end_time: z.number().optional(),
});

type SheetFormValues = z.infer<typeof sheetFormSchema>;

const DEFAULT_VALUES: SheetFormValues = {
  supplier_id: undefined,
  name: '',
  status: SHEET_STATUS.ACTIVE,
  start_time: Math.floor(Date.now() / 1000),
  end_time: 0,
};

type SheetDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  supplierId?: number;
  currentRow?: SupplierPricingSheet;
  mode?: 'create' | 'update' | 'view';
};

export function SheetDrawer({
  open,
  onOpenChange,
  supplierId,
  currentRow,
  mode = 'update',
}: SheetDrawerProps) {
  const { t } = useTranslation();
  const { triggerSheetRefresh, selectedSupplierId } = useSupplierPricing();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const isView = mode === 'view';
  const isUpdate = mode === 'update';
  const isCreate = mode === 'create';

  const form = useForm<SheetFormValues>({
    resolver: zodResolver(sheetFormSchema),
    defaultValues: DEFAULT_VALUES,
  });

  const { data: suppliersData } = useQuery({
    queryKey: ['suppliers', 'drawer'],
    queryFn: async () => {
      const result = await getSuppliers({ p: 1, page_size: 1000 });
      return result.data?.items ?? [];
    },
    enabled: open && !isView,
    staleTime: 5 * 60 * 1000,
  });

  const { data: viewData } = useQuery({
    queryKey: ['supplier-sheet', 'view', currentRow?.supplier_id, currentRow?.id],
    queryFn: () =>
      getSupplierPricingSheet(currentRow!.supplier_id, currentRow!.id),
    enabled: open && isView && !!currentRow,
  });

  const effectiveSupplierId = supplierId ?? selectedSupplierId;

  useEffect(() => {
    if (open && isUpdate && currentRow) {
      getSupplierPricingSheet(currentRow.supplier_id, currentRow.id).then((result) => {
        form.reset({
          supplier_id: result.supplier_id,
          name: result.name,
          status: result.status,
          start_time: result.start_time,
          end_time: result.end_time,
        });
      });
    } else if (open && isCreate) {
      form.reset({
        ...DEFAULT_VALUES,
        supplier_id: effectiveSupplierId ?? undefined,
      });
    }
  }, [open, isUpdate, isCreate, currentRow, form, effectiveSupplierId]);

  const onSubmit = async (data: SheetFormValues) => {
    const sid = data.supplier_id ?? effectiveSupplierId;
    if (!sid) return;
    setIsSubmitting(true);
    try {
      const payload = {
        name: data.name,
        status: data.status,
        start_time: data.start_time,
        end_time: data.end_time,
      };

      if (isUpdate && currentRow) {
        const result = await updateSupplierPricingSheet(
          currentRow.supplier_id,
          currentRow.id,
          payload
        );
        if (result) {
          toast.success(t(SUCCESS_MESSAGES.UPDATED));
          onOpenChange(false);
          triggerSheetRefresh();
        } else {
          toast.error(t(ERROR_MESSAGES.UPDATE_FAILED));
        }
      } else {
        const result = await createSupplierPricingSheet(sid, payload);
        if (result) {
          toast.success(t(SUCCESS_MESSAGES.CREATED));
          onOpenChange(false);
          triggerSheetRefresh();
        } else {
          toast.error(t(ERROR_MESSAGES.CREATE_FAILED));
        }
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = (v: boolean) => {
    onOpenChange(v);
    if (!v) form.reset();
  };

  const getSupplierName = () => {
    if (isView && viewData) {
      return (
        suppliersData?.find((s: { id: number }) => s.id === viewData.supplier_id)
          ?.name ?? '-'
      );
    }
    return suppliersData?.find(
      (s: { id: number }) =>
        s.id === (isView ? viewData?.supplier_id : currentRow?.supplier_id)
    )?.name ?? '-';
  };

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className='flex h-dvh w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[500px]'>
        <SheetHeader className='border-b px-4 py-3 text-start sm:px-6 sm:py-4'>
          <SheetTitle>
            {isView
              ? t('View Pricing Sheet')
              : isCreate
                ? t('Add Pricing Sheet')
                : t('Edit Pricing Sheet')}
          </SheetTitle>
          <SheetDescription>
            {isView
              ? t('Pricing sheet details.')
              : isCreate
                ? t('Add a new pricing sheet.')
                : t('Update the pricing sheet.')}
          </SheetDescription>
        </SheetHeader>

        {isView ? (
          <div className='flex-1 space-y-4 overflow-y-auto px-4 py-4'>
            <ViewField label={t('Supplier')}>{getSupplierName()}</ViewField>
            <ViewField label={t('Sheet Name')}>
              {viewData?.name ?? '-'}
            </ViewField>
            <ViewField label={t('Status')}>
              {viewData
                ? SHEET_STATUS_LABELS[viewData.status] ?? String(viewData.status)
                : '-'}
            </ViewField>
            <ViewField label={t('Channel ID')}>
              {viewData?.channel_id && viewData.channel_id > 0
                ? viewData.channel_id
                : '-'}
            </ViewField>
            <ViewField label={t('Start Time')}>
              {viewData?.start_time
                ? formatTimestamp(viewData.start_time)
                : '-'}
            </ViewField>
            <ViewField label={t('End Time')}>
              {viewData?.end_time
                ? formatTimestamp(viewData.end_time)
                : '-'}
            </ViewField>
            <ViewField label={t('Created At')}>
              {viewData?.created_at
                ? formatTimestamp(viewData.created_at)
                : '-'}
            </ViewField>
            <ViewField label={t('Updated At')}>
              {viewData?.updated_at
                ? formatTimestamp(viewData.updated_at)
                : '-'}
            </ViewField>
          </div>
        ) : (
          <Form {...form}>
            <form
              id='sheet-form'
              onSubmit={form.handleSubmit(onSubmit)}
              className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'
            >
              <FormField
                control={form.control}
                name='supplier_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Supplier')}</FormLabel>
                    <Select
                      onValueChange={(value) => field.onChange(parseInt(value))}
                      value={field.value ? String(field.value) : ''}
                      disabled={isUpdate}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select supplier')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {suppliersData?.map((s: { id: number; name: string }) => (
                            <SelectItem key={s.id} value={String(s.id)}>
                              {s.name}
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
                      onValueChange={(value) => field.onChange(parseInt(value))}
                      value={String(field.value)}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue>
                            {getSheetStatusOptions(t).find((o) => o.value === String(field.value))?.label ?? '-'}
                          </SelectValue>
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {getSheetStatusOptions(t).map((opt) => (
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
                name='start_time'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Start Time')}</FormLabel>
                    <FormControl>
                      <Input
                        type='datetime-local'
                        value={formatTimestampForInput(field.value ?? 0)}
                        onChange={(e) => {
                          field.onChange(parseTimestampFromInput(e.target.value) || undefined);
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
                        value={formatTimestampForInput(field.value ?? 0)}
                        onChange={(e) => {
                          field.onChange(parseTimestampFromInput(e.target.value) || undefined);
                        }}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </form>
          </Form>
        )}

        <SheetFooter className='grid grid-cols-2 gap-2 border-t px-4 py-3 sm:flex sm:px-6 sm:py-4'>
          <Button variant='outline' onClick={() => handleClose(false)} type='button'>
            {t('Close')}
          </Button>
          {!isView && (
            <Button
              type='button'
              disabled={isSubmitting}
              onClick={form.handleSubmit(onSubmit)}
            >
              {isSubmitting
                ? t('Saving...')
                : isCreate
                  ? t('Create')
                  : t('Save changes')}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function ViewField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className='text-muted-foreground mb-1 text-xs font-medium uppercase tracking-wide'>
        {label}
      </div>
      <div className='text-sm'>{children}</div>
    </div>
  );
}
