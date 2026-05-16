import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { unbindUser } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function UnbindUserDialog() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { bindingOpen, setBindingOpen, currentBindingId, selectedEnterpriseId } =
    useEnterprisePricing()
  const [isUnbinding, setIsUnbinding] = useState(false)

  const handleUnbind = async () => {
    if (!currentBindingId || !selectedEnterpriseId) return
    setIsUnbinding(true)
    try {
      const result = await unbindUser(selectedEnterpriseId, currentBindingId)
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.USER_UNBOUND))
        setBindingOpen(null)
        await queryClient.invalidateQueries({ queryKey: ['enterprise-users', selectedEnterpriseId] })
        await queryClient.invalidateQueries({ queryKey: ['users', 'all'] })
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.UNBIND_FAILED))
      }
    } catch (_error) {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsUnbinding(false)
    }
  }

  return (
    <ConfirmDialog
      open={bindingOpen === 'unbind'}
      onOpenChange={(v) => !v && setBindingOpen(null)}
      title={t('Unbind User')}
      desc={t(
        'Are you sure you want to unbind this user from the enterprise? This action cannot be undone.'
      )}
      confirmText={t('Unbind')}
      destructive
      handleConfirm={handleUnbind}
      isLoading={isUnbinding}
    />
  )
}
