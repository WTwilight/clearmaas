import { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { useQuery } from '@tanstack/react-query';
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
  FormDescription,
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
import { MultiSelect, type Option } from '@/components/multi-select';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  getEnabledModels,
  createSupplierPricingItem,
  updateSupplierPricingItem,
} from '../api';
import {
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  DISCOUNT_TYPE,
  DISCOUNT_TYPE_OPTIONS,
  DISCOUNT_TYPE_LABELS,
  getVendorTypeOptions,
  getModelsByVendor,
  getVendorLabel,
  type VendorType,
} from '../constants';
import { useSupplierPricing } from './supplier-pricing-provider';
import type { SupplierPricingItem } from '../types';
import { z } from 'zod';

const itemFormSchema = z.object({
  vendor_type: z.string().min(1, 'Vendor type is required'),
  models: z.array(z.string()).min(1, 'At least one model is required'),
  discount_type: z.enum(['ratio', 'fixed_price', 'per_call']),
  discount_value: z.number().min(0, 'Value must be non-negative'),
  remark: z.string().optional(),
});

type ItemFormValues = z.infer<typeof itemFormSchema>;

const DEFAULT_VALUES: ItemFormValues = {
  vendor_type: '',
  models: [],
  discount_type: DISCOUNT_TYPE.RATIO,
  discount_value: 1,
  remark: '',
};

type ItemDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item?: SupplierPricingItem;
  mode?: 'create' | 'update' | 'view';
};

export function ItemDrawer({
  open,
  onOpenChange,
  item,
  mode = 'update',
}: ItemDrawerProps) {
  const { t } = useTranslation();
  const isView = mode === 'view';
  const isUpdate = mode === 'update';
  const isCreate = mode === 'create';
  const { selectedSupplierId, selectedSheetId, triggerItemRefresh } = useSupplierPricing();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<ItemFormValues>({
    resolver: zodResolver(itemFormSchema),
    defaultValues: DEFAULT_VALUES,
  });

  const { data: allModels, isLoading: isLoadingModels } = useQuery({
    queryKey: ['models', 'enabled'],
    queryFn: getEnabledModels,
    staleTime: 10 * 60 * 1000,
  });

  const selectedVendor = form.watch('vendor_type');
  const vendorModelOptions: Option[] = useMemo(() => {
    if (!selectedVendor || !allModels) return [];
    const models = getModelsByVendor(selectedVendor as VendorType, allModels);
    return models.map((m) => ({ value: m, label: m }));
  }, [selectedVendor, allModels]);

  useEffect(() => {
    if (open) {
      if (isUpdate && item) {
        form.reset({
          vendor_type: item.vendor_type || '',
          models: item.models ?? [],
          discount_type: item.discount_type as 'ratio' | 'fixed_price' | 'per_call',
          discount_value: item.discount_value,
          remark: item.remark || '',
        });
      } else if (isCreate) {
        form.reset(DEFAULT_VALUES);
      }
    }
  }, [open, isUpdate, isCreate, item, form]);

  const handleVendorChange = (vendor: string) => {
    form.setValue('vendor_type', vendor);
    form.setValue('models', []);
  };

  const onSubmit = async (values: ItemFormValues) => {
    if (!selectedSupplierId || !selectedSheetId) {
      toast.error(t('No supplier or sheet selected'));
      return;
    }

    setIsSubmitting(true);
    try {
      if (isUpdate && item) {
        await updateSupplierPricingItem(
          selectedSupplierId,
          selectedSheetId,
          item.id,
          {
            models: values.models,
            discount_type: values.discount_type,
            discount_value: values.discount_value,
            remark: values.remark || undefined,
          }
        );
        toast.success(t(SUCCESS_MESSAGES.UPDATED));
        onOpenChange(false);
        triggerItemRefresh();
      } else {
        await createSupplierPricingItem(selectedSupplierId, selectedSheetId, {
          vendor_type: values.vendor_type,
          models: values.models,
          discount_type: values.discount_type,
          discount_value: values.discount_value,
          remark: values.remark || undefined,
        });
        toast.success(t(SUCCESS_MESSAGES.CREATED));
        onOpenChange(false);
        triggerItemRefresh();
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

  const formatValue = (type: string, value: number) => {
    if (type === 'ratio') return `${(value * 100).toFixed(0)}%`;
    if (type === 'per_call') return `$${value.toFixed(4)}/call`;
    return `$${value.toFixed(6)}`;
  };

  const resolvedVendor = item?.vendor_type ? getVendorLabel(item.vendor_type) : null;

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className='flex h-dvh w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[500px]'>
        <SheetHeader className='border-b px-4 py-3 text-start sm:px-6 sm:py-4'>
          <SheetTitle>
            {isView
              ? t('View Pricing Item')
              : isCreate
                ? t('Add Pricing Item')
                : t('Edit Pricing Item')}
          </SheetTitle>
          <SheetDescription>
            {isView
              ? t('Pricing item details.')
              : isCreate
                ? t('Select models to add to this pricing sheet.')
                : t('Update the pricing item.')}
          </SheetDescription>
        </SheetHeader>

        {isView && item ? (
          <div className='flex-1 space-y-4 overflow-y-auto px-4 py-4'>
            <ViewField label={t('Vendor')}>{resolvedVendor ?? '-'}</ViewField>
            <ViewField label={t('Models')}>{item.models?.join(', ')}</ViewField>
            <ViewField label={t('Cost Type')}>
              {DISCOUNT_TYPE_LABELS[item.discount_type] ?? item.discount_type}
            </ViewField>
            <ViewField label={t('Cost Value')}>
              {formatValue(item.discount_type, item.discount_value)}
            </ViewField>
            <ViewField label={t('Remark')}>
              {item.remark || '-'}
            </ViewField>
          </div>
        ) : (
          <Form {...form}>
            <form onSubmit={(e) => e.preventDefault()} className='flex-1 space-y-4 overflow-y-auto px-3 py-3 pb-4 sm:space-y-6 sm:px-4'>
              <FormField
                control={form.control}
                name='vendor_type'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Vendor')}</FormLabel>
                    <Select
                      onValueChange={handleVendorChange}
                      value={field.value}
                      disabled={isUpdate}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('Select vendor')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectGroup>
                          {getVendorTypeOptions(t).map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {t('Select the vendor to filter available models.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='models'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Models')}</FormLabel>
                    <FormControl>
                      {isLoadingModels ? (
                        <div className='flex h-10 items-center text-sm text-muted-foreground'>
                          {t('Loading models...')}
                        </div>
                      ) : (
                        <Controller
                          control={form.control}
                          name='models'
                          render={({ field: controllerField }) => (
                            <MultiSelect
                              options={vendorModelOptions}
                              selected={controllerField.value}
                              onChange={controllerField.onChange}
                              placeholder={
                                selectedVendor
                                  ? t('Select models...')
                                  : t('Select vendor first')
                              }
                            />
                          )}
                        />
                      )}
                    </FormControl>
                    <FormDescription>
                      {selectedVendor
                        ? `${vendorModelOptions.length} ${t('models available for this vendor.')}`
                        : t('Choose a vendor above to see available models.')}
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
                    <FormLabel>{t('Cost Type')}</FormLabel>
                    <Select
                      onValueChange={(value) =>
                        field.onChange(value as 'ratio' | 'fixed_price' | 'per_call')
                      }
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue>
                            {DISCOUNT_TYPE_OPTIONS.find((o) => o.value === field.value)?.label ?? '-'}
                          </SelectValue>
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {DISCOUNT_TYPE_OPTIONS.map((opt) => (
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
                  const discountType = form.watch('discount_type');
                  return (
                    <FormItem>
                      <FormLabel>
                        {discountType === DISCOUNT_TYPE.RATIO
                          ? t('Ratio')
                          : discountType === DISCOUNT_TYPE.PER_CALL
                            ? t('Price per Call')
                            : t('Fixed Price')}
                      </FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          step='0.000001'
                          {...field}
                          onChange={(e) =>
                            field.onChange(parseFloat(e.target.value) || 0)
                          }
                          placeholder={
                            discountType === DISCOUNT_TYPE.RATIO
                              ? 'e.g., 0.8 (80% of base price)'
                              : discountType === DISCOUNT_TYPE.PER_CALL
                                ? 'e.g., 0.05 ($/call)'
                                : 'e.g., 0.001'
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {discountType === DISCOUNT_TYPE.RATIO
                          ? t('Ratio to base price (e.g., 0.8 = pay 80%)')
                          : discountType === DISCOUNT_TYPE.PER_CALL
                            ? t('Fixed price per API call (USD).')
                            : t('Fixed unit price.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  );
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

              <SheetFooter className='grid grid-cols-2 gap-2 sm:flex sm:gap-2'>
                <Button variant='outline' type='button' onClick={() => handleClose(false)}>
                  {t('Close')}
                </Button>
                <Button
                  type='button'
                  disabled={isSubmitting || !selectedSheetId}
                  onClick={form.handleSubmit(onSubmit)}
                >
                  {isSubmitting
                    ? t('Saving...')
                    : isCreate
                      ? t('Create')
                      : t('Save changes')}
                </Button>
              </SheetFooter>
            </form>
          </Form>
        )}

        {isView && (
          <SheetFooter className='grid grid-cols-1 gap-2 border-t px-4 py-3 sm:flex sm:px-6 sm:py-4'>
            <Button variant='outline' onClick={() => handleClose(false)} type='button'>
              {t('Close')}
            </Button>
          </SheetFooter>
        )}
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
