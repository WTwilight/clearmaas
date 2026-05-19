import { Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { useSupplierPricing } from './supplier-pricing-provider';

export function SupplierPrimaryButtons() {
  const { t } = useTranslation();
  const { openSupplierDialog } = useSupplierPricing();

  return (
    <Button size='sm' onClick={() => openSupplierDialog({ type: 'create' })}>
      <Plus className='h-4 w-4' />
      {t('Add Supplier')}
    </Button>
  );
}
