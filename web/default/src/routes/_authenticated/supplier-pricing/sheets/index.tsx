import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { SheetSection } from '@/features/supplier-pricing/components/sheet-section';

const supplierSheetsSearchSchema = z.object({
  supplierId: z.number().optional(),
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(20),
  filter: z.string().optional().catch(''),
});

export const Route = createFileRoute('/_authenticated/supplier-pricing/sheets/')({
  validateSearch: supplierSheetsSearchSchema,
  component: SheetSection,
});
