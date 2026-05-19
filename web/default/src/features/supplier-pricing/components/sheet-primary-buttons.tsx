import { Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { useSupplierPricing } from './supplier-pricing-provider';

export function SheetPrimaryButtons() {
  const { t } = useTranslation();
  const { openSheetDialog } = useSupplierPricing();

  return (
    <Button size='sm' onClick={() => openSheetDialog({ type: 'create' })}>
      <Plus className='h-4 w-4' />
      {t('Add Pricing Sheet')}
    </Button>
  );
}
