import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useFormContext } from 'react-hook-form';
import { Link2, Unlink } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Combobox } from '@/components/ui/combobox';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import { updateSupplierPricingSheet } from '../api';
import { useSupplierPricing } from './supplier-pricing-provider';
import { z } from 'zod';
import { api as axiosApi } from '@/lib/api';

const bindFormSchema = z.object({
  channel_id: z.string().min(1, 'Please select a channel'),
});

type BindFormValues = z.infer<typeof bindFormSchema>;

interface ChannelInfo {
  id: number;
  name: string;
  type: number;
  status: number;
}

export function SheetBindDialog() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { sheetBindDialog, closeSheetBindDialog } = useSupplierPricing();

  const sheet = sheetBindDialog.sheet;
  const currentChannelId = sheet?.channel_id ?? 0;

  const form = useForm<BindFormValues>({
    resolver: zodResolver(bindFormSchema),
    defaultValues: {
      channel_id: '',
    },
  });

  const { data: allChannels, isLoading: isLoadingChannels } = useQuery({
    queryKey: ['channels', 'all'],
    queryFn: async () => {
      const res = await axiosApi.get('/api/channel', {
        params: { p: 1, page_size: 500, status: 1 },
      });
      return (res.data.data?.items ?? []) as ChannelInfo[];
    },
    enabled: sheetBindDialog.open,
    staleTime: 5 * 60 * 1000,
  });

  const currentChannel =
    currentChannelId > 0
      ? allChannels?.find((ch) => ch.id === currentChannelId)
      : null;

  const channelOptions =
    (allChannels ?? []).map((ch) => ({
      value: String(ch.id),
      label: `${ch.name} (${ch.type})`,
    })) ?? [];

  const handleClose = (open: boolean) => {
    if (!open) {
      closeSheetBindDialog();
      form.reset();
    }
  };

  return (
    <Dialog open={sheetBindDialog.open} onOpenChange={handleClose}>
      <DialogContent className='max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Bind Channel')}</DialogTitle>
          <DialogDescription>
            {sheet?.name
              ? `${t('Bind channel for sheet')}: ${sheet.name}`
              : t('Bind a channel to this pricing sheet.')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form className='space-y-4'>
            <CurrentBinding
              currentChannel={currentChannel}
              currentChannelId={currentChannelId}
            />

            <Separator />

            <BindNewChannel
              channelOptions={channelOptions}
              isLoadingChannels={isLoadingChannels}
              sheet={sheet}
              currentChannelId={currentChannelId}
            />

            <DialogFooter className='gap-2 sm:gap-0'>
              <Button variant='outline' type='button' onClick={() => handleClose(false)}>
                {t('Cancel')}
              </Button>
              <BindSubmitButton
                sheet={sheet}
                currentChannelId={currentChannelId}
                queryClient={queryClient}
                closeSheetBindDialog={closeSheetBindDialog}
              />
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

type ChannelInfo2 = {
  id: number;
  name: string;
  type: number;
  status: number;
};

type Sheet = {
  id: number;
  supplier_id: number;
  channel_id: number;
  name: string;
  status: number;
  start_time: number;
  end_time: number;
};

function CurrentBinding({
  currentChannel,
  currentChannelId,
}: {
  currentChannel: ChannelInfo2 | null | undefined;
  currentChannelId: number;
}) {
  const { t } = useTranslation();
  const { sheetBindDialog, closeSheetBindDialog } = useSupplierPricing();
  const [isUnbinding, setIsUnbinding] = useState(false);
  const queryClient = useQueryClient();
  const sheet = sheetBindDialog.sheet;
  const isUniversal = currentChannelId === 0;

  const handleUnbind = async () => {
    if (!sheet) return;
    setIsUnbinding(true);
    try {
      await updateSupplierPricingSheet(sheet.supplier_id, sheet.id, {
        name: sheet.name,
        status: sheet.status,
        channel_id: 0,
        start_time: sheet.start_time,
        end_time: sheet.end_time,
      });
      toast.success(t('Channel unbound successfully'));
      queryClient.invalidateQueries({ queryKey: ['supplier-pricing-sheets'] });
      queryClient.invalidateQueries({ queryKey: ['supplier-pricing-sheets-all'] });
      closeSheetBindDialog();
    } catch (err) {
      toast.error((err as Error).message || t('Failed to unbind channel'));
    } finally {
      setIsUnbinding(false);
    }
  };

  return (
    <div>
      <div className='mb-2 text-sm font-medium'>{t('Current Binding')}</div>
      <div className='flex items-center justify-between rounded-md border border-border bg-muted/50 px-3 py-2'>
        <div className='flex items-center gap-2'>
          <Link2 className='h-4 w-4 text-muted-foreground' />
          {currentChannel ? (
            <span className='text-sm'>
              <span className='font-medium'>{currentChannel.name}</span>
              <span className='ml-1 text-muted-foreground'>
                (ID: {currentChannel.id}, Type: {currentChannel.type})
              </span>
            </span>
          ) : (
            <span className='text-sm text-muted-foreground'>
              {t('Universal — no channel bound')}
            </span>
          )}
        </div>
        {!isUniversal && (
          <Button
            variant='ghost'
            size='sm'
            className='text-destructive hover:text-destructive'
            onClick={handleUnbind}
            disabled={isUnbinding}
            type='button'
          >
            <Unlink className='mr-1 h-3.5 w-3.5' />
            {isUnbinding ? t('Unbinding...') : t('Unbind')}
          </Button>
        )}
      </div>
    </div>
  );
}

function BindNewChannel({
  channelOptions,
  isLoadingChannels,
}: {
  channelOptions: Array<{ value: string; label: string }>;
  isLoadingChannels: boolean;
}) {
  const { t } = useTranslation();
  const { control } = useFormContext<BindFormValues>();

  return (
    <div>
      <div className='mb-2 text-sm font-medium'>{t('Bind New Channel')}</div>
      <FormField
        control={control}
        name='channel_id'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Channel')}</FormLabel>
            <FormControl>
              {isLoadingChannels ? (
                <Skeleton className='h-10 w-full' />
              ) : (
                <Combobox
                  options={channelOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                  placeholder={t('Select a channel...')}
                  searchPlaceholder={t('Search channels...')}
                  emptyText={t('No channel found')}
                  customDisplayValue={
                    channelOptions.find((o) => o.value === field.value)?.label ?? ''
                  }
                />
              )}
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}

type QueryClient = ReturnType<typeof useQueryClient>;

function BindSubmitButton({
  sheet,
  currentChannelId,
  queryClient,
  closeSheetBindDialog,
}: {
  sheet: Sheet | undefined;
  currentChannelId: number;
  queryClient: QueryClient;
  closeSheetBindDialog: () => void;
}) {
  const { t } = useTranslation();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { handleSubmit, watch, formState } = useFormContext<BindFormValues>();

  const channelId = watch('channel_id');

  const onSubmit = async (values: BindFormValues) => {
    if (!sheet) return;
    const newChannelId = parseInt(values.channel_id);
    if (newChannelId === currentChannelId) {
      closeSheetBindDialog();
      return;
    }

    setIsSubmitting(true);
    try {
      await updateSupplierPricingSheet(sheet.supplier_id, sheet.id, {
        name: sheet.name,
        status: sheet.status,
        channel_id: newChannelId,
        start_time: sheet.start_time,
        end_time: sheet.end_time,
      });
      toast.success(t('Channel bound successfully'));
      queryClient.invalidateQueries({ queryKey: ['supplier-pricing-sheets'] });
      queryClient.invalidateQueries({ queryKey: ['supplier-pricing-sheets-all'] });
      closeSheetBindDialog();
    } catch (err) {
      toast.error((err as Error).message || t('Failed to bind channel'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Button
      onClick={handleSubmit(onSubmit)}
      disabled={isSubmitting || !channelId}
      type='button'
    >
      {isSubmitting ? t('Binding...') : t('Bind')}
    </Button>
  );
}
