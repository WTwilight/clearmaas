import { createFileRoute } from '@tanstack/react-router'
import { ItemSection } from '@/features/enterprise-pricing/components/item-section'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/items/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <ItemSection />
}
