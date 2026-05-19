import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Unlink } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Combobox } from '@/components/ui/combobox'
import { getEnterpriseUsers, bindUser, unbindUser, searchAllUsers, getAllUserBindings } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { z } from 'zod'
import { Skeleton } from '@/components/ui/skeleton'
import { LongText } from '@/components/long-text'
import { formatTimestamp } from '@/lib/format'
import type { EnterpriseUserBinding, EnterpriseUserWithBinding } from '../types'

const bindFormSchema = z.object({
  user_id: z.string().min(1, 'Please select a user'),
})

type BindFormValues = z.infer<typeof bindFormSchema>

export function BindUserDialog() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const {
    bindingOpen,
    setBindingOpen,
    selectedEnterpriseId,
  } = useEnterprisePricing()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [unbindingUserId, setUnbindingUserId] = useState<number | null>(null)

  const form = useForm<BindFormValues>({
    resolver: zodResolver(bindFormSchema),
    defaultValues: {
      user_id: '',
    },
  })

  // Fetch all users for selection
  const { data: allUsersData, isLoading: isLoadingUsers } = useQuery({
    queryKey: ['users', 'all'],
    queryFn: async () => {
      const result = await searchAllUsers()
      return result || []
    },
    enabled: bindingOpen === 'bind',
    staleTime: 5 * 60 * 1000,
  })

  // Fetch all bindings across all enterprises to filter out already-bound users
  const { data: allBindingsData, isLoading: isLoadingAllBindings } = useQuery<EnterpriseUserBinding[]>({
    queryKey: ['enterprise-users', 'all-bindings'],
    queryFn: async () => {
      const result = await getAllUserBindings()
      if (!result.success) {
        return []
      }
      return result.data || []
    },
    enabled: bindingOpen === 'bind',
    staleTime: 30 * 1000,
  })

  // Get IDs of users already bound to any enterprise
  const boundUserIds = new Set((allBindingsData || []).map((b) => b.user_id))

  // Fetch bound users for the current enterprise (for display only)
  const { data: boundUsersData, isLoading: isLoadingBoundUsers } = useQuery<EnterpriseUserWithBinding[]>({
    queryKey: ['enterprise-users', selectedEnterpriseId, 'binding-dialog'],
    queryFn: async () => {
      if (!selectedEnterpriseId) return []
      const result = await getEnterpriseUsers(selectedEnterpriseId)
      if (!result.success) {
        return []
      }
      return result.data || []
    },
    enabled: bindingOpen === 'bind' && !!selectedEnterpriseId,
    staleTime: 30 * 1000,
  })

  // Filter out already bound users from the options
  const availableUsers = (allUsersData || []).filter(
    (user) => !boundUserIds.has(user.id)
  )

  const onSubmit = async (data: BindFormValues) => {
    if (!selectedEnterpriseId) return
    const userId = parseInt(data.user_id)
    if (isNaN(userId)) return

    setIsSubmitting(true)
    try {
      const result = await bindUser(selectedEnterpriseId, { user_id: userId })

      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.USER_BOUND))
        form.reset()
        await queryClient.invalidateQueries({ queryKey: ['enterprise-users', selectedEnterpriseId] })
        await queryClient.invalidateQueries({ queryKey: ['users', 'all'] })
        await queryClient.invalidateQueries({ queryKey: ['enterprise-users', 'all-bindings'] })
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.BIND_FAILED))
      }
    } catch (_error) {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleUnbind = async (userId: number) => {
    if (!selectedEnterpriseId) return
    setUnbindingUserId(userId)
    try {
      const result = await unbindUser(selectedEnterpriseId, userId)
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.USER_UNBOUND))
        await queryClient.invalidateQueries({ queryKey: ['enterprise-users', selectedEnterpriseId] })
        await queryClient.invalidateQueries({ queryKey: ['users', 'all'] })
        await queryClient.invalidateQueries({ queryKey: ['enterprise-users', 'all-bindings'] })
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.UNBIND_FAILED))
      }
    } catch (_error) {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setUnbindingUserId(null)
    }
  }

  const handleClose = (open: boolean) => {
    if (!open) {
      setBindingOpen(null)
      form.reset()
    }
  }

  return (
    <Dialog open={bindingOpen === 'bind'} onOpenChange={handleClose}>
      <DialogContent className='max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Bind User')}</DialogTitle>
          <DialogDescription>
            {t('Manage users for this enterprise.')}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4'>
          {/* Already bound users section */}
          <div className='space-y-2'>
            <div className='flex items-center gap-2'>
              <h4 className='text-sm font-medium'>{t('Bound Users')}</h4>
              {isLoadingBoundUsers ? (
                <Skeleton className='h-4 w-16' />
              ) : (
                <span className='text-muted-foreground text-xs'>
                  ({boundUsersData?.length || 0})
                </span>
              )}
            </div>

            {isLoadingBoundUsers ? (
              <div className='space-y-2'>
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
              </div>
            ) : boundUsersData && boundUsersData.length > 0 ? (
              <div className='max-h-[200px] space-y-1.5 overflow-y-auto rounded-md border p-2'>
                {boundUsersData.map((user) => (
                  <div
                    key={user.user_id}
                    className='flex items-center justify-between gap-2 rounded-md bg-muted/50 px-3 py-2'
                  >
                    <div className='min-w-0 flex-1'>
                      <div className='flex items-center gap-2'>
                        <LongText className='max-w-[180px] text-sm font-medium'>
                          {user.display_name
                            ? `${user.username} (${user.display_name})`
                            : user.username}
                        </LongText>
                      </div>
                      {user.created_at && (
                        <span className='text-muted-foreground text-xs'>
                          {t('Bound at')} {formatTimestamp(user.created_at)}
                        </span>
                      )}
                    </div>
                    <Button
                      size='sm'
                      variant='ghost'
                      className='text-destructive hover:text-destructive'
                      onClick={() => handleUnbind(user.user_id)}
                      disabled={unbindingUserId === user.user_id}
                    >
                      <Unlink className='mr-1 h-3.5 w-3.5' />
                      {t('Unbind')}
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <div className='rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground'>
                {t('No users bound to this enterprise yet.')}
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
                {t('Bind New User')}
              </span>
            </div>
          </div>

          {/* Bind new user form */}
          <Form {...form}>
            <form
              id='bind-form'
              onSubmit={form.handleSubmit(onSubmit)}
              className='space-y-4'
            >
              <FormField
                control={form.control}
                name='user_id'
                render={({ field }) => {
                  const selectedUser = availableUsers.find(
                    (u) => String(u.id) === field.value
                  )
                  const displayLabel = selectedUser
                    ? selectedUser.display_name
                      ? `${selectedUser.username} (${selectedUser.display_name})`
                      : selectedUser.username
                    : ''
                  return (
                    <FormItem>
                      <FormLabel>{t('Select User')}</FormLabel>
                      <FormControl>
                        <Combobox
                          options={
                            availableUsers.map((user) => ({
                              value: String(user.id),
                              label: user.display_name
                                ? `${user.username} (${user.display_name})`
                                : user.username,
                            })) || []
                          }
                          value={field.value}
                          onValueChange={field.onChange}
                          placeholder={t('Search users...')}
                          searchPlaceholder={t('Search users...')}
                          emptyText={t('No user found')}
                          customDisplayValue={displayLabel}
                        />
                      </FormControl>
                      <FormDescription>
                        {isLoadingUsers ? t('Loading users...') : ''}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )
                }}
              />
            </form>
          </Form>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => handleClose(false)}>
            {t('Close')}
          </Button>
          <Button
            form='bind-form'
            type='submit'
            disabled={isSubmitting || !form.watch('user_id')}
          >
            {isSubmitting ? t('Binding...') : t('Bind')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
