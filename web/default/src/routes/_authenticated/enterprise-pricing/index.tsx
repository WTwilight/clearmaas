import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/enterprise-pricing/')({
  beforeLoad: () => {
    throw redirect({
      to: '/enterprise-pricing/enterprise',
    })
  },
  component: () => null,
})
