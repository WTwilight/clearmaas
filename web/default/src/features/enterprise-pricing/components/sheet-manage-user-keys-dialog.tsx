import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Key, Unlink, Search } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBadge } from '@/components/status-badge'
import { LongText } from '@/components/long-text'
import { formatTimestamp } from '@/lib/format'
import { getSheetTokenBindings, unbindTokenPricingBinding, type SheetTokenBinding } from '../api'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function SheetManageUserKeysDialog() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { userKeysOpen, setUserKeysOpen, currentSheet } = useEnterprisePricing()
  const [unbindingTokenId, setUnbindingTokenId] = useState<number | null>(null)
  const [usernameFilter, setUsernameFilter] = useState('')
  const [nameFilter, setNameFilter] = useState('')

  const { data: bindings, isLoading } = useQuery<SheetTokenBinding[]>({
    queryKey: ['sheet-token-bindings', currentSheet?.id],
    queryFn: async () => {
      if (!currentSheet?.id) return []
      return await getSheetTokenBindings(currentSheet.id)
    },
    enabled: userKeysOpen === 'manageUserKeys' && !!currentSheet?.id,
    staleTime: 30 * 1000,
  })

  const filteredBindings = useMemo(() => {
    if (!bindings) return []
    const q = usernameFilter.toLowerCase().trim()
    const n = nameFilter.toLowerCase().trim()
    return bindings.filter((b) => {
      const matchUsername = !q || b.username.toLowerCase().includes(q)
      const matchName = !n || (b.name ?? '').toLowerCase().includes(n)
      return matchUsername && matchName
    })
  }, [bindings, usernameFilter, nameFilter])

  const handleUnbind = async (tokenId: number) => {
    setUnbindingTokenId(tokenId)
    try {
      await unbindTokenPricingBinding(tokenId)
      toast.success(t('Token unbind successful'))
      await queryClient.invalidateQueries({ queryKey: ['sheet-token-bindings', currentSheet?.id] })
    } catch (err) {
      toast.error((err as Error).message || t('Failed to unbind token'))
    } finally {
      setUnbindingTokenId(null)
    }
  }

  const handleClose = (open: boolean) => {
    if (!open) {
      setUserKeysOpen(null)
      setUsernameFilter('')
      setNameFilter('')
    }
  }

  return (
    <Dialog open={userKeysOpen === 'manageUserKeys'} onOpenChange={handleClose}>
      <DialogContent className='!max-w-[900px] sm:!max-w-[900px] max-h-[85vh] flex flex-col'>
        <DialogHeader>
          <DialogTitle>{t('Manage User Keys')}</DialogTitle>
          <DialogDescription>
            {currentSheet?.name
              ? `${t('View tokens bound to this pricing sheet')}: ${currentSheet.name}`
              : t('View and unbind tokens bound to this pricing sheet.')}
          </DialogDescription>
        </DialogHeader>

        {/* Filters */}
        <div className='flex items-center gap-3 flex-wrap'>
          <div className='flex items-center gap-2 flex-1 min-w-[200px]'>
            <Search className='h-4 w-4 shrink-0 text-muted-foreground' />
            <Input
              placeholder={t('Filter by username...')}
              value={usernameFilter}
              onChange={(e) => setUsernameFilter(e.target.value)}
              className='h-8'
            />
          </div>
          <div className='flex items-center gap-2 flex-1 min-w-[200px]'>
            <Search className='h-4 w-4 shrink-0 text-muted-foreground' />
            <Input
              placeholder={t('Filter by key name...')}
              value={nameFilter}
              onChange={(e) => setNameFilter(e.target.value)}
              className='h-8'
            />
          </div>
          {!isLoading && (
            <span className='text-muted-foreground text-sm shrink-0'>
              {filteredBindings.length} / {bindings?.length ?? 0}
            </span>
          )}
        </div>

        <div className='flex-1 overflow-hidden'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className='h-16 w-full' />
              ))}
            </div>
          ) : filteredBindings.length > 0 ? (
            <ScrollArea className='h-[420px] rounded-md border'>
              <div className='space-y-1.5 p-2'>
                {filteredBindings.map((binding) => {
                  const isUnbinding = unbindingTokenId === binding.id
                  return (
                    <div
                      key={binding.id}
                      className='flex items-center justify-between gap-4 rounded-md bg-muted/50 px-3 py-2.5'
                    >
                      <div className='flex items-start gap-3 min-w-0 flex-1'>
                        <Key className='mt-0.5 h-4 w-4 shrink-0 text-muted-foreground' />
                        <div className='min-w-0 flex-1'>
                          <div className='flex items-center gap-2 flex-wrap'>
                            <LongText className='text-sm font-medium'>
                              {binding.name || t('Unnamed Token')}
                            </LongText>
                            <StatusBadge
                              label={binding.status === 1 ? t('Enabled') : t('Disabled')}
                              variant={binding.status === 1 ? 'success' : 'neutral'}
                              showDot={true}
                              copyable={false}
                            />
                          </div>
                          <div className='mt-0.5 flex items-center gap-4 text-xs text-muted-foreground flex-wrap'>
                            <span className='shrink-0'>ID: {binding.id}</span>
                            <span className='shrink-0'>User: {binding.username}</span>
                            {binding.token_group && (
                              <span className='shrink-0'>Group: {binding.token_group}</span>
                            )}
                            <span className='shrink-0'>
                              {t('Bound at')} {formatTimestamp(binding.binding_created_at)}
                            </span>
                            <span className='shrink-0'>
                              <LongText className='max-w-[200px]'>
                                Key: {binding.key ? `•••• ${binding.key.slice(-8)}` : '-'}
                              </LongText>
                            </span>
                          </div>
                        </div>
                      </div>
                      <Button
                        size='sm'
                        variant='ghost'
                        className='text-destructive hover:text-destructive shrink-0'
                        onClick={() => handleUnbind(binding.id)}
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
          ) : bindings && bindings.length > 0 ? (
            <div className='rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground'>
              {t('No results match the current filters.')}
            </div>
          ) : (
            <div className='rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground'>
              {t('No tokens bound to this pricing sheet yet.')}
            </div>
          )}
        </div>

        <div className='flex justify-end'>
          <Button variant='outline' onClick={() => handleClose(false)}>
            {t('Close')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
