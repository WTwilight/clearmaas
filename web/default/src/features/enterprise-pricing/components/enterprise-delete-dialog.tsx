import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { deleteEnterprise } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function EnterpriseDeleteDialog() {
  const { t } = useTranslation()
  const { enterpriseOpen, setEnterpriseOpen, currentEnterprise, triggerRefresh } =
    useEnterprisePricing()
  const [isDeleting, setIsDeleting] = useState(false)

  const handleDelete = async () => {
    if (!currentEnterprise) return
    setIsDeleting(true)
    try {
      const result = await deleteEnterprise(currentEnterprise.id)
      if (result.success) {
        toast.success(t('Enterprise deleted successfully'))
        setEnterpriseOpen(null)
        triggerRefresh()
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.DELETE_FAILED))
      }
    } catch (_error) {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <ConfirmDialog
      open={enterpriseOpen === 'delete'}
      onOpenChange={(v) => !v && setEnterpriseOpen(null)}
      title={t('Delete Enterprise')}
      desc={t(
        'Are you sure you want to delete "{{name}}"? This action cannot be undone.',
        { name: currentEnterprise?.name || '' }
      )}
      confirmText={t('Delete')}
      destructive
      handleConfirm={handleDelete}
      isLoading={isDeleting}
    />
  )
}
