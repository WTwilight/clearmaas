import { createFileRoute } from '@tanstack/react-router';
import { ItemSection } from '@/features/supplier-pricing/components/item-section';

export const Route = createFileRoute('/_authenticated/supplier-pricing/items/')({
  component: ItemSection,
});
