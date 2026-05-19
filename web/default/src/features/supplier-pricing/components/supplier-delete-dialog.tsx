import { Modal, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteSupplier } from '../api';
import { supplierQueryKeys } from '../lib/query-keys';
import { useSupplierPricing } from './supplier-pricing-provider';
import type { Supplier } from '../types';

interface Props {
  open: boolean;
  supplier?: Supplier;
  onClose: () => void;
}

export function SupplierDeleteDialog({ open, supplier, onClose }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { triggerSupplierRefresh } = useSupplierPricing();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!supplier) throw new Error('No supplier selected');
      return deleteSupplier(supplier.id);
    },
    onSuccess: () => {
      message.success(t('Supplier deleted'));
      queryClient.invalidateQueries({ queryKey: supplierQueryKeys.all });
      triggerSupplierRefresh();
      onClose();
    },
    onError: (err: Error) => {
      message.error(err.message || t('Operation failed'));
    },
  });

  return (
    <Modal
      title={t('Delete Supplier')}
      open={open}
      onOk={() => mutation.mutate()}
      onCancel={onClose}
      okText={t('Delete')}
      okButtonProps={{ danger: true, loading: mutation.isPending }}
    >
      <p>
        {t('Are you sure you want to delete supplier "{{name}}"?', {
          name: supplier?.name,
        })}
      </p>
    </Modal>
  );
}
