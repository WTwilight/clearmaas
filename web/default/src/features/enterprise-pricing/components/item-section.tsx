import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ItemTable } from './item-table'
import { ItemPrimaryButtons } from './item-primary-buttons'
import { ItemDrawer } from './item-drawer'
import { ItemDeleteDialog } from './item-delete-dialog'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function ItemSection() {
  const { t } = useTranslation()
  const {
    itemOpen,
    setItemOpen,
    currentItemId,
    setCurrentItemId,
  } = useEnterprisePricing()

  const [currentSheetId, setCurrentSheetId] = useState<number | null>(null)

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setItemOpen(null)
      setCurrentItemId(null)
    }
  }

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Items')}</h3>
        <ItemPrimaryButtons sheetId={currentSheetId} />
      </div>
      <ItemTable onSheetIdChange={setCurrentSheetId} />
      <ItemDrawer
        open={itemOpen === 'create' || itemOpen === 'update'}
        onOpenChange={handleOpenChange}
        currentItemId={currentItemId}
        sheetId={currentSheetId}
      />
      <ItemDeleteDialog
        currentItemId={currentItemId}
        sheetId={currentSheetId}
      />
    </>
  )
}
