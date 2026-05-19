import { Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { useSupplierPricing } from './supplier-pricing-provider';

type ItemPrimaryButtonsProps = {
  sheetId: number | null;
};

export function ItemPrimaryButtons({ sheetId }: ItemPrimaryButtonsProps) {
  const { t } = useTranslation();
  const { openItemDialog } = useSupplierPricing();

  return (
    <Button
      size='sm'
      onClick={() => openItemDialog({ type: 'create' })}
      disabled={!sheetId}
    >
      <Plus className='h-4 w-4' />
      {t('Add Pricing Item')}
    </Button>
  );
}
