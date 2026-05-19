import { createFileRoute, redirect } from '@tanstack/react-router';
import { useAuthStore } from '@/stores/auth-store';
import { ROLE } from '@/lib/roles';
import { SupplierPricing } from '@/features/supplier-pricing';

export const Route = createFileRoute('/_authenticated/supplier-pricing')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState();

    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({
        to: '/403',
      });
    }
  },
  component: SupplierPricing,
});
