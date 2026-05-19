import { useTranslation } from 'react-i18next';
import { SheetTable } from './sheet-table';
import { SheetPrimaryButtons } from './sheet-primary-buttons';
import { SheetDrawer } from './sheet-drawer';
import { SheetDeleteDialog } from './sheet-delete-dialog';
import { SheetBindDialog } from './sheet-bind-dialog';
import { useSupplierPricing } from './supplier-pricing-provider';

export function SheetSection() {
  const { t } = useTranslation();
  const {
    sheetDialog,
    openSheetDialog,
    closeSheetDialog,
  } = useSupplierPricing();

  const handleDrawerOpenChange = (isOpen: boolean) => {
    if (!isOpen) closeSheetDialog();
  };

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Sheets', { ns: 'common' })}</h3>
        <SheetPrimaryButtons />
      </div>
      <SheetTable />
      <SheetDrawer
        open={sheetDialog.open && sheetDialog.type !== 'delete' && sheetDialog.type !== 'view'}
        onOpenChange={handleDrawerOpenChange}
        supplierId={sheetDialog.supplierId}
        currentRow={sheetDialog.sheet}
        mode={sheetDialog.type === 'create' ? 'create' : 'update'}
      />
      {sheetDialog.type === 'view' && (
        <SheetDrawer
          open={sheetDialog.open}
          onOpenChange={handleDrawerOpenChange}
          supplierId={sheetDialog.supplierId}
          currentRow={sheetDialog.sheet}
          mode='view'
        />
      )}
      <SheetDeleteDialog
        open={sheetDialog.type === 'delete' && sheetDialog.open}
        sheet={sheetDialog.sheet}
        onClose={closeSheetDialog}
      />
      <SheetBindDialog />
    </>
  );
}
