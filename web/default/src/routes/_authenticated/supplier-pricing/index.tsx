import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/supplier-pricing/')({
  beforeLoad: () => {
    throw redirect({
      to: '/supplier-pricing/suppliers',
    });
  },
  component: () => null,
});
