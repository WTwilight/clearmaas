import { useTranslation } from 'react-i18next'
import { useNavigate } from '@tanstack/react-router'
import { ItemTable } from './item-table'
import { ItemPrimaryButtons } from './item-primary-buttons'
import { ItemDrawer } from './item-drawer'
import { ItemDeleteDialog } from './item-delete-dialog'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Route } from '@/routes/_authenticated/enterprise-pricing/items'

export function ItemSection() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const {
    itemOpen,
    setItemOpen,
    currentItemId,
    setCurrentItemId,
  } = useEnterprisePricing()

  const urlSheetId = Route.useSearch({ strict: false }).sheetId

  const handleSheetIdChange = (sheetId: number | null) => {
    if (sheetId) {
      navigate({
        search: (prev) => ({ ...prev, sheetId }),
      })
    }
  }

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
        <ItemPrimaryButtons sheetId={urlSheetId ?? null} />
      </div>
      <ItemTable
        urlSheetId={urlSheetId}
        onSheetIdChange={handleSheetIdChange}
      />
      <ItemDrawer
        open={itemOpen === 'create' || itemOpen === 'update'}
        onOpenChange={handleOpenChange}
        currentItemId={currentItemId}
        sheetId={urlSheetId ?? null}
      />
      <ItemDeleteDialog
        currentItemId={currentItemId}
        sheetId={urlSheetId ?? null}
      />
    </>
  )
}
