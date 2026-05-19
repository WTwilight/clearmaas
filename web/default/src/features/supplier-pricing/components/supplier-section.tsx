import { useTranslation } from 'react-i18next';
import { SupplierTable } from './supplier-table';
import { SupplierPrimaryButtons } from './supplier-primary-buttons';
import { SupplierDrawer } from './supplier-drawer';
import { SupplierDeleteDialog } from './supplier-delete-dialog';
import { useSupplierPricing } from './supplier-pricing-provider';

export function SupplierSection() {
  const { t } = useTranslation();
  const {
    supplierDialog,
    openSupplierDialog,
    closeSupplierDialog,
  } = useSupplierPricing();

  const handleDrawerOpenChange = (isOpen: boolean) => {
    if (!isOpen) closeSupplierDialog();
  };

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Suppliers', { ns: 'common' })}</h3>
        <SupplierPrimaryButtons />
      </div>
      <SupplierTable />
      <SupplierDrawer
        open={supplierDialog.open && supplierDialog.type !== 'delete'}
        onOpenChange={handleDrawerOpenChange}
        currentRow={supplierDialog.supplier}
      />
      <SupplierDeleteDialog
        open={supplierDialog.type === 'delete' && supplierDialog.open}
        supplier={supplierDialog.supplier}
        onClose={closeSupplierDialog}
      />
    </>
  );
}
