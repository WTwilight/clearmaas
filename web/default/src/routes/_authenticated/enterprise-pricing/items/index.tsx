import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ItemSection } from '@/features/enterprise-pricing/components/item-section'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/items/')({
  validateSearch: z.object({
    sheetId: z.number().optional(),
    page: z.number().optional().catch(1),
    pageSize: z.number().optional().catch(20),
  }),
  component: RouteComponent,
})

function RouteComponent() {
  return <ItemSection />
}
