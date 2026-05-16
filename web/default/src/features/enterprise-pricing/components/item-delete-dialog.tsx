import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { deletePricingItem } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'

type ItemDeleteDialogProps = {
  currentItemId: number | null
  sheetId: number | null
}

export function ItemDeleteDialog({ currentItemId, sheetId }: ItemDeleteDialogProps) {
  const { t } = useTranslation()
  const { itemOpen, setItemOpen, triggerItemRefresh } = useEnterprisePricing()
  const [isDeleting, setIsDeleting] = useState(false)

  const handleDelete = async () => {
    if (!currentItemId || !sheetId) return
    setIsDeleting(true)
    try {
      const result = await deletePricingItem(sheetId, currentItemId)
      if (result.success) {
        toast.success(t('Pricing item deleted successfully'))
        setItemOpen(null)
        triggerItemRefresh()
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
      open={itemOpen === 'delete'}
      onOpenChange={(v) => !v && setItemOpen(null)}
      title={t('Delete Pricing Item')}
      desc={t('Are you sure you want to delete this pricing item? This action cannot be undone.')}
      confirmText={t('Delete')}
      destructive
      handleConfirm={handleDelete}
      isLoading={isDeleting}
    />
  )
}
