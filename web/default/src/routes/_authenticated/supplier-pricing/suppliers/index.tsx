import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { SupplierSection } from '@/features/supplier-pricing/components/supplier-section';

const supplierSearchSchema = z.object({
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(20),
  filter: z.string().optional().catch(''),
});

export const Route = createFileRoute('/_authenticated/supplier-pricing/suppliers/')({
  validateSearch: supplierSearchSchema,
  component: SupplierSection,
});
