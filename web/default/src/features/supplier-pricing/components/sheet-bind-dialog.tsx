import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link2, Unlink } from 'lucide-react';
import { api as axiosApi } from '@/lib/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Skeleton } from '@/components/ui/skeleton';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  getSupplierPricingSheetChannels,
  bindSupplierPricingSheetChannels,
  unbindSupplierPricingSheetChannel,
} from '../api';
import { useSupplierPricing } from './supplier-pricing-provider';

interface ChannelInfo {
  id: number;
  name: string;
  type: number;
  status: number;
}

export function SheetBindDialog() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { sheetBindDialog, closeSheetBindDialog, triggerSheetRefresh } = useSupplierPricing();
  const sheet = sheetBindDialog.sheet;

  const [selectedChannelIds, setSelectedChannelIds] = useState<Set<number>>(new Set());
  const [submittingChannelIds, setSubmittingChannelIds] = useState<Set<number>>(new Set());
  const [isBinding, setIsBinding] = useState(false);
  const [initialized, setInitialized] = useState(false);

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

  const { data: boundChannelIds, isLoading: isLoadingBound } = useQuery({
    queryKey: ['supplier-pricing-sheet-channels', sheet?.id],
    queryFn: async () => {
      if (!sheet?.supplier_id || !sheet?.id) return [];
      return await getSupplierPricingSheetChannels(sheet.supplier_id, sheet.id);
    },
    enabled: sheetBindDialog.open && !!sheet?.supplier_id && !!sheet?.id,
    staleTime: 30 * 1000,
  });

  useEffect(() => {
    if (!initialized && boundChannelIds !== undefined) {
      setSelectedChannelIds(new Set(boundChannelIds));
      setInitialized(true);
    }
  }, [boundChannelIds, initialized]);

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setChannelBindingOpen();
      setSelectedChannelIds(new Set());
      setInitialized(false);
    }
  };

  const setChannelBindingOpen = () => {
    closeSheetBindDialog();
  };

  const toggleChannel = (channelId: number) => {
    const next = new Set(selectedChannelIds);
    if (next.has(channelId)) {
      next.delete(channelId);
    } else {
      next.add(channelId);
    }
    setSelectedChannelIds(next);
  };

  const handleUnbind = async (channelId: number) => {
    if (!sheet?.supplier_id || !sheet?.id) return;
    setSubmittingChannelIds((s) => new Set([...s, channelId]));
    try {
      await unbindSupplierPricingSheetChannel(sheet.supplier_id, sheet.id, channelId);
      toast.success(t('Channel unbound successfully'));
      const next = new Set(selectedChannelIds);
      next.delete(channelId);
      setSelectedChannelIds(next);
      await queryClient.invalidateQueries({
        queryKey: ['supplier-pricing-sheet-channels', sheet.id],
      });
      triggerSheetRefresh();
    } catch (err) {
      toast.error((err as Error).message || t('Failed to unbind channel'));
    } finally {
      setSubmittingChannelIds((s) => {
        const next = new Set(s);
        next.delete(channelId);
        return next;
      });
    }
  };

  const handleSubmit = async () => {
    if (!sheet?.supplier_id || !sheet?.id) return;
    setIsBinding(true);
    try {
      await bindSupplierPricingSheetChannels(
        sheet.supplier_id,
        sheet.id,
        Array.from(selectedChannelIds)
      );
      toast.success(t('Channels bound successfully'));
      await queryClient.invalidateQueries({
        queryKey: ['supplier-pricing-sheet-channels', sheet.id],
      });
      triggerSheetRefresh();
      handleOpenChange(false);
    } catch (err) {
      toast.error((err as Error).message || t('Failed to bind channels'));
    } finally {
      setIsBinding(false);
    }
  };

  const isLoading = isLoadingChannels || isLoadingBound;

  const noChanges =
    selectedChannelIds.size === (boundChannelIds?.length ?? 0) &&
    [...selectedChannelIds].every((id) => (boundChannelIds ?? []).includes(id));

  return (
    <Dialog open={sheetBindDialog.open} onOpenChange={handleOpenChange}>
      <DialogContent className='max-w-lg flex flex-col max-h-[85vh]'>
        <DialogHeader>
          <DialogTitle>{t('Bind Channels')}</DialogTitle>
          <DialogDescription>
            {sheet?.name
              ? `${t('Bind channels for sheet')}: ${sheet.name}`
              : t('Manage channel bindings for this pricing sheet.')}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4 flex-1 overflow-y-auto'>
          <div className='space-y-2'>
            <div className='flex items-center gap-2'>
              <h4 className='text-sm font-medium'>{t('Bound Channels')}</h4>
              {isLoadingBound ? (
                <Skeleton className='h-4 w-16' />
              ) : (
                <span className='text-muted-foreground text-xs'>
                  ({boundChannelIds?.length || 0})
                </span>
              )}
            </div>

            {isLoadingBound ? (
              <div className='space-y-2'>
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
              </div>
            ) : boundChannelIds && boundChannelIds.length > 0 ? (
              <ScrollArea className='h-[200px] rounded-md border'>
                <div className='space-y-1 p-2'>
                  {boundChannelIds.map((channelId) => {
                    const ch = allChannels?.find((c) => c.id === channelId);
                    const isUnbinding = submittingChannelIds.has(channelId);
                    return (
                      <div
                        key={channelId}
                        className='flex items-center justify-between gap-2 rounded-md bg-muted/50 px-3 py-2'
                      >
                        <div className='flex items-center gap-2'>
                          <Link2 className='h-4 w-4 text-muted-foreground' />
                          <span className='text-sm'>
                            {ch ? (
                              <>
                                <span className='font-medium'>{ch.name}</span>
                                <span className='ml-1 text-muted-foreground'>
                                  (ID: {ch.id}, Type: {ch.type})
                                </span>
                              </>
                            ) : (
                              <span className='text-muted-foreground'>
                                Channel #{channelId}
                              </span>
                            )}
                          </span>
                        </div>
                        <Button
                          size='sm'
                          variant='ghost'
                          className='text-destructive hover:text-destructive'
                          onClick={() => handleUnbind(channelId)}
                          disabled={isUnbinding}
                          type='button'
                        >
                          <Unlink className='mr-1 h-3.5 w-3.5' />
                          {isUnbinding ? t('Unbinding...') : t('Unbind')}
                        </Button>
                      </div>
                    );
                  })}
                </div>
              </ScrollArea>
            ) : (
              <div className='rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground'>
                {t('No channels bound to this pricing sheet yet.')}
              </div>
            )}
          </div>

          <div className='relative'>
            <div className='absolute inset-0 flex items-center'>
              <span className='w-full border-t' />
            </div>
            <div className='relative flex justify-center text-xs uppercase'>
              <span className='bg-background px-2 text-muted-foreground'>
                {t('Select Channels')}
              </span>
            </div>
          </div>

          <div className='space-y-2'>
            <div className='text-sm font-medium'>
              {t('Available Channels')}
              {isLoadingChannels && (
                <span className='ml-2 text-muted-foreground'>
                  ({t('Loading...')})
                </span>
              )}
            </div>

            {isLoadingChannels ? (
              <div className='space-y-2'>
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className='h-10 w-full' />
                ))}
              </div>
            ) : (
              <ScrollArea className='max-h-[300px] rounded-md border'>
                <div className='space-y-1 p-2'>
                  {(allChannels ?? []).map((ch) => (
                    <label
                      key={ch.id}
                      className='flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 hover:bg-muted/50'
                    >
                      <Checkbox
                        checked={selectedChannelIds.has(ch.id)}
                        onCheckedChange={() => toggleChannel(ch.id)}
                      />
                      <span className='text-sm'>
                        <span className='font-medium'>{ch.name}</span>
                        <span className='ml-1 text-muted-foreground'>
                          (ID: {ch.id}, Type: {ch.type})
                        </span>
                      </span>
                    </label>
                  ))}
                  {allChannels?.length === 0 && (
                    <div className='p-4 text-center text-sm text-muted-foreground'>
                      {t('No channels available.')}
                    </div>
                  )}
                </div>
              </ScrollArea>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => handleOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isBinding || isLoading || noChanges}
          >
            {isBinding ? t('Binding...') : t('Bind Channels')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
