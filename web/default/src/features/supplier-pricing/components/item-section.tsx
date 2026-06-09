import { useTranslation } from 'react-i18next'
import { Alert } from '@/components/ui/alert'
import { ItemTable } from './item-table'
import { ItemPrimaryButtons } from './item-primary-buttons'
import { ItemDrawer } from './item-drawer'
import { ItemDeleteDialog } from './item-delete-dialog'
import { useSupplierPricing } from './supplier-pricing-provider'
import { Route } from '@/routes/_authenticated/supplier-pricing/items'
import type { SupplierPricingItem } from '../types'

export function ItemSection() {
  const { t } = useTranslation()
  const {
    itemDialog,
    closeItemDialog,
  } = useSupplierPricing()

  const routeSearch = Route.useSearch()
  const currentSheetId = routeSearch.sheetId
  const currentSupplierId = routeSearch.supplierId

  const hasSelection = currentSupplierId && currentSheetId

  const handleDrawerOpenChange = (isOpen: boolean) => {
    if (!isOpen) closeItemDialog()
  }

  const dialogItem: SupplierPricingItem | undefined = itemDialog.item as SupplierPricingItem | undefined

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Items', { ns: 'common' })}</h3>
        <ItemPrimaryButtons sheetId={currentSheetId ?? null} />
      </div>
      {!hasSelection && (
        <Alert variant='destructive' className='mb-2'>
          {t('Please select a supplier and pricing sheet first', { ns: 'common' })}
        </Alert>
      )}
      <ItemTable />
      <ItemDrawer
        open={itemDialog.open && itemDialog.type !== 'delete' && itemDialog.type !== 'view'}
        onOpenChange={handleDrawerOpenChange}
        item={dialogItem}
        mode={itemDialog.type === 'create' ? 'create' : 'update'}
      />
      {itemDialog.type === 'view' && (
        <ItemDrawer
          open={itemDialog.open}
          onOpenChange={handleDrawerOpenChange}
          item={dialogItem}
          mode='view'
        />
      )}
      <ItemDeleteDialog
        open={itemDialog.type === 'delete' && itemDialog.open}
        item={dialogItem}
        onClose={closeItemDialog}
      />
    </>
  )
}
