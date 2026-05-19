import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link2, Unlink } from 'lucide-react'
import { api as axiosApi } from '@/lib/api'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  getSheetChannels,
  bindSheetChannels,
  unbindSheetChannel,
} from '../api'
import { useEnterprisePricing } from './enterprise-pricing-provider'

interface ChannelInfo {
  id: number
  name: string
  type: number
  status: number
}

export function SheetBindChannelDialog() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { channelBindingOpen, setChannelBindingOpen, currentSheet } =
    useEnterprisePricing()

  const [selectedChannelIds, setSelectedChannelIds] = useState<Set<number>>(
    new Set()
  )
  const [submittingChannelIds, setSubmittingChannelIds] = useState<Set<number>>(
    new Set()
  )
  const [isBinding, setIsBinding] = useState(false)

  // Fetch all available channels
  const { data: allChannels, isLoading: isLoadingChannels } = useQuery({
    queryKey: ['channels', 'all'],
    queryFn: async () => {
      const res = await axiosApi.get('/api/channel', {
        params: { p: 1, page_size: 500, status: 1 },
      })
      return (res.data.data?.items ?? []) as ChannelInfo[]
    },
    enabled: channelBindingOpen === 'bindChannels',
    staleTime: 5 * 60 * 1000,
  })

  // Fetch currently bound channel IDs
  const { data: boundChannelIds, isLoading: isLoadingBound } = useQuery({
    queryKey: ['enterprise-pricing-sheet-channels', currentSheet?.id],
    queryFn: async () => {
      if (!currentSheet?.id) return []
      return await getSheetChannels(currentSheet.id)
    },
    enabled: channelBindingOpen === 'bindChannels' && !!currentSheet?.id,
    staleTime: 30 * 1000,
  })

  // Sync selectedChannelIds when boundChannelIds loads
  const [initialized, setInitialized] = useState(false)
  useEffect(() => {
    if (!initialized && boundChannelIds !== undefined) {
      setSelectedChannelIds(new Set(boundChannelIds))
      setInitialized(true)
    }
  }, [boundChannelIds, initialized])

  // Reset state when dialog opens
  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setChannelBindingOpen(null)
      setSelectedChannelIds(new Set())
      setInitialized(false)
    }
  }

  const toggleChannel = (channelId: number) => {
    const next = new Set(selectedChannelIds)
    if (next.has(channelId)) {
      next.delete(channelId)
    } else {
      next.add(channelId)
    }
    setSelectedChannelIds(next)
  }

  const handleUnbind = async (channelId: number) => {
    if (!currentSheet?.id) return
    setSubmittingChannelIds((s) => new Set([...s, channelId]))
    try {
      await unbindSheetChannel(currentSheet.id, channelId)
      toast.success(t('Channel unbound successfully'))
      const next = new Set(selectedChannelIds)
      next.delete(channelId)
      setSelectedChannelIds(next)
      await queryClient.invalidateQueries({
        queryKey: ['enterprise-pricing-sheet-channels', currentSheet.id],
      })
    } catch (err) {
      toast.error((err as Error).message || t('Failed to unbind channel'))
    } finally {
      setSubmittingChannelIds((s) => {
        const next = new Set(s)
        next.delete(channelId)
        return next
      })
    }
  }

  const handleSubmit = async () => {
    if (!currentSheet?.id) return
    setIsBinding(true)
    try {
      await bindSheetChannels(currentSheet.id, Array.from(selectedChannelIds))
      toast.success(t('Channels bound successfully'))
      await queryClient.invalidateQueries({
        queryKey: ['enterprise-pricing-sheet-channels', currentSheet.id],
      })
      handleOpenChange(false)
    } catch (err) {
      toast.error((err as Error).message || t('Failed to bind channels'))
    } finally {
      setIsBinding(false)
    }
  }

  const isLoading = isLoadingChannels || isLoadingBound

  const noChanges =
    selectedChannelIds.size === (boundChannelIds?.length ?? 0) &&
    [...selectedChannelIds].every((id) => (boundChannelIds ?? []).includes(id))

  return (
    <Dialog
      open={channelBindingOpen === 'bindChannels'}
      onOpenChange={handleOpenChange}
    >
      <DialogContent className='max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Bind Channels')}</DialogTitle>
          <DialogDescription>
            {currentSheet?.name
              ? `${t('Bind channels for sheet')}: ${currentSheet.name}`
              : t('Manage channel bindings for this pricing sheet.')}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4'>
          {/* Current bindings section */}
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
              <ScrollArea className='max-h-[200px] rounded-md border'>
                <div className='space-y-1 p-2'>
                  {boundChannelIds.map((channelId) => {
                    const ch = allChannels?.find((c) => c.id === channelId)
                    const isUnbinding = submittingChannelIds.has(channelId)
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
                    )
                  })}
                </div>
              </ScrollArea>
            ) : (
              <div className='rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground'>
                {t('No channels bound to this pricing sheet yet.')}
              </div>
            )}
          </div>

          {/* Divider */}
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

          {/* Channel selection */}
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
  )
}
