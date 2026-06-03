import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ItemSection } from '@/features/enterprise-pricing/components/item-section'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/items/')({
  validateSearch: z.object({
    sheetId: z.number().optional(),
  }),
  component: RouteComponent,
})

function RouteComponent() {
  return <ItemSection />
}
