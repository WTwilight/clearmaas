import { useTranslation } from 'react-i18next';
import { Outlet } from '@tanstack/react-router';
import { SupplierPricingProvider } from './components/supplier-pricing-provider';
import { SupplierPricingNav } from './components/supplier-pricing-nav';
import { SheetBindDialog } from './components/sheet-bind-dialog';
import { Main, PageFooterPortal } from '@/components/layout';

function SupplierPricingContent() {
  const { t } = useTranslation();

  return (
    <SupplierPricingProvider>
      <SheetBindDialog />
      <Main>
        <div className="shrink-0 px-3 pt-3 pb-2.5 sm:px-4 sm:pt-5 sm:pb-3">
          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 sm:gap-x-4">
            <div className="min-w-0">
              <h2 className="truncate text-base font-bold tracking-tight sm:text-lg">
                {t('Supplier Pricing')}
              </h2>
            </div>
          </div>
          <div className="mt-3 flex items-center gap-1 overflow-x-auto">
            <SupplierPricingNav />
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-auto px-3 pt-1 pb-3 sm:px-4 sm:pt-1.5 sm:pb-4">
          <Outlet />
        </div>

        <PageFooterPortal>
          <div />
        </PageFooterPortal>
      </Main>
    </SupplierPricingProvider>
  );
}

export { SupplierPricingContent };
export function SupplierPricing() {
  return <SupplierPricingContent />;
}
