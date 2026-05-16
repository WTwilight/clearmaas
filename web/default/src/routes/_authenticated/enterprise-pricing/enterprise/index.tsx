import { createFileRoute } from '@tanstack/react-router'
import { EnterpriseSection } from '@/features/enterprise-pricing/components/enterprise-section'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/enterprise/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <EnterpriseSection />
}
