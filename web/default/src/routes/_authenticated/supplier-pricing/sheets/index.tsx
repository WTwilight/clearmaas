import { createFileRoute } from '@tanstack/react-router';
import { SheetSection } from '@/features/supplier-pricing/components/sheet-section';

export const Route = createFileRoute('/_authenticated/supplier-pricing/sheets/')({
  component: SheetSection,
});
