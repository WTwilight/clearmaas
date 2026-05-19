import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert } from '@/components/ui/alert';
import { ItemTable } from './item-table';
import { ItemPrimaryButtons } from './item-primary-buttons';
import { ItemDrawer } from './item-drawer';
import { ItemDeleteDialog } from './item-delete-dialog';
import { useSupplierPricing } from './supplier-pricing-provider';

export function ItemSection() {
  const { t } = useTranslation();
  const {
    itemDialog,
    openItemDialog,
    closeItemDialog,
    selectedSheetId,
  } = useSupplierPricing();

  const [currentSheetId, setCurrentSheetId] = useState<number | null>(null);

  const hasSelection = selectedSheetId && currentSheetId;

  const handleDrawerOpenChange = (isOpen: boolean) => {
    if (!isOpen) closeItemDialog();
  };

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Items', { ns: 'common' })}</h3>
        <ItemPrimaryButtons sheetId={currentSheetId} />
      </div>
      {!hasSelection && (
        <Alert variant='warning' className='mb-2'>
          {t('Please select a supplier and pricing sheet first', { ns: 'common' })}
        </Alert>
      )}
      <ItemTable onSheetIdChange={setCurrentSheetId} />
      <ItemDrawer
        open={itemDialog.open && itemDialog.type !== 'delete' && itemDialog.type !== 'view'}
        onOpenChange={handleDrawerOpenChange}
        item={itemDialog.item}
        mode={itemDialog.type === 'create' ? 'create' : 'update'}
      />
      {itemDialog.type === 'view' && (
        <ItemDrawer
          open={itemDialog.open}
          onOpenChange={handleDrawerOpenChange}
          item={itemDialog.item}
          mode='view'
        />
      )}
      <ItemDeleteDialog
        open={itemDialog.type === 'delete' && itemDialog.open}
        item={itemDialog.item}
        onClose={closeItemDialog}
      />
    </>
  );
}
