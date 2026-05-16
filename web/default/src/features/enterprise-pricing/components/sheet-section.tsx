import { useTranslation } from 'react-i18next'
import { SheetTable } from './sheet-table'
import { SheetPrimaryButtons } from './sheet-primary-buttons'
import { SheetDrawer } from './sheet-drawer'
import { SheetDeleteDialog } from './sheet-delete-dialog'
import { BindUserDialog } from './bind-user-dialog'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function SheetSection() {
  const { t } = useTranslation()
  const { sheetOpen, setSheetOpen, currentSheet } = useEnterprisePricing()

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Sheet Management')}</h3>
        <SheetPrimaryButtons />
      </div>
      <SheetTable />
      <SheetDrawer
        open={sheetOpen === 'create' || sheetOpen === 'update'}
        onOpenChange={(isOpen) => !isOpen && setSheetOpen(null)}
        currentRow={sheetOpen === 'update' ? currentSheet || undefined : undefined}
      />
      <SheetDeleteDialog />
      <BindUserDialog />
    </>
  )
}
