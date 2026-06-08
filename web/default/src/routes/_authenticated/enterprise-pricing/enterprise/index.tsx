import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { EnterpriseSection } from '@/features/enterprise-pricing/components/enterprise-section'

const enterpriseSearchSchema = z.object({
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(20),
  filter: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/enterprise-pricing/enterprise/')({
  validateSearch: enterpriseSearchSchema,
  component: RouteComponent,
})

function RouteComponent() {
  return <EnterpriseSection />
}
