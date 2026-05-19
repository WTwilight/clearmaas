import { createFileRoute } from '@tanstack/react-router';
import { SupplierSection } from '@/features/supplier-pricing/components/supplier-section';

export const Route = createFileRoute('/_authenticated/supplier-pricing/suppliers/')({
  component: SupplierSection,
});
