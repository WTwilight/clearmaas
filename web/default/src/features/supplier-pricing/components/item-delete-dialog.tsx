import { Modal, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useMutation } from '@tanstack/react-query';
import { deleteSupplierPricingItem } from '../api';
import { useSupplierPricing } from './supplier-pricing-provider';
import type { SupplierPricingItem } from '../types';

interface Props {
  open: boolean;
  item?: SupplierPricingItem;
  onClose: () => void;
}

export function ItemDeleteDialog({ open, item, onClose }: Props) {
  const { t } = useTranslation();
  const { selectedSupplierId, selectedSheetId, triggerItemRefresh } = useSupplierPricing();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!item || !selectedSupplierId || !selectedSheetId) throw new Error('Invalid state');
      return deleteSupplierPricingItem(selectedSupplierId, selectedSheetId, item.id);
    },
    onSuccess: () => {
      message.success(t('Pricing item deleted'));
      triggerItemRefresh();
      onClose();
    },
    onError: (err: Error) => {
      message.error(err.message || t('Operation failed'));
    },
  });

  return (
    <Modal
      title={t('Delete Pricing Item')}
      open={open}
      onOk={() => mutation.mutate()}
      onCancel={onClose}
      okText={t('Delete')}
      okButtonProps={{ danger: true, loading: mutation.isPending }}
    >
      <p>
        {t('Are you sure you want to delete this pricing item?', {
          models: item?.models?.join(', '),
        })}
      </p>
    </Modal>
  );
}
