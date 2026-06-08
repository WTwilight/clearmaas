import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ItemSection } from '@/features/supplier-pricing/components/item-section'

const supplierPricingItemsSearchSchema = z.object({
  sheetId: z.number().optional(),
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(20),
})

export const Route = createFileRoute('/_authenticated/supplier-pricing/items/')({
  validateSearch: supplierPricingItemsSearchSchema,
  component: ItemSection,
})
