import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SheetSection } from '@/features/enterprise-pricing/components/sheet-section'

const enterpriseSheetsSearchSchema = z.object({
  enterpriseId: z.number().optional(),
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(20),
  filter: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/enterprise-pricing/sheets/')({
  validateSearch: enterpriseSheetsSearchSchema,
  component: RouteComponent,
})

function RouteComponent() {
  const { enterpriseId } = Route.useSearch()
  return <SheetSection enterpriseId={enterpriseId} />
}
