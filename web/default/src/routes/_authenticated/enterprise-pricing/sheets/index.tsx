import { createFileRoute } from '@tanstack/react-router'
import { SheetSection } from '@/features/enterprise-pricing/components/sheet-section'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/sheets/')({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      enterpriseId: search.enterpriseId as number | undefined,
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { enterpriseId } = Route.useSearch()
  return <SheetSection enterpriseId={enterpriseId} />
}
