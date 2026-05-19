import { Modal, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteSupplierPricingSheet } from '../api';
import { supplierSheetQueryKeys } from '../lib/query-keys';
import { useSupplierPricing } from './supplier-pricing-provider';
import type { SupplierPricingSheet } from '../types';

interface Props {
  open: boolean;
  sheet?: SupplierPricingSheet;
  onClose: () => void;
}

export function SheetDeleteDialog({ open, sheet, onClose }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { triggerSheetRefresh } = useSupplierPricing();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!sheet) throw new Error('No sheet selected');
      return deleteSupplierPricingSheet(sheet.supplier_id, sheet.id);
    },
    onSuccess: () => {
      message.success(t('Pricing sheet deleted'));
      queryClient.invalidateQueries({ queryKey: supplierSheetQueryKeys.all });
      triggerSheetRefresh();
      onClose();
    },
    onError: (err: Error) => {
      message.error(err.message || t('Operation failed'));
    },
  });

  return (
    <Modal
      title={t('Delete Pricing Sheet')}
      open={open}
      onOk={() => mutation.mutate()}
      onCancel={onClose}
      okText={t('Delete')}
      okButtonProps={{ danger: true, loading: mutation.isPending }}
    >
      <p>
        {t('Are you sure you want to delete pricing sheet "{{name}}"?', {
          name: sheet?.name,
        })}
      </p>
    </Modal>
  );
}
