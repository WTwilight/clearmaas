import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { deleteSheet } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function SheetDeleteDialog() {
  const { t } = useTranslation()
  const { sheetOpen, setSheetOpen, currentSheet, selectedEnterpriseId, triggerSheetRefresh } =
    useEnterprisePricing()
  const [isDeleting, setIsDeleting] = useState(false)

  const handleDelete = async () => {
    if (!currentSheet || !selectedEnterpriseId) return
    setIsDeleting(true)
    try {
      const result = await deleteSheet(selectedEnterpriseId, currentSheet.id)
      if (result.success) {
        toast.success(t('Pricing sheet deleted successfully'))
        setSheetOpen(null)
        triggerSheetRefresh()
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
      open={sheetOpen === 'delete'}
      onOpenChange={(v) => !v && setSheetOpen(null)}
      title={t('Delete Pricing Sheet')}
      desc={t(
        'Are you sure you want to delete "{{name}}"? This action cannot be undone.',
        { name: currentSheet?.name || '' }
      )}
      confirmText={t('Delete')}
      destructive
      handleConfirm={handleDelete}
      isLoading={isDeleting}
    />
  )
}
