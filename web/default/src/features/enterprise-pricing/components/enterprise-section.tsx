import { useTranslation } from 'react-i18next'
import { EnterpriseTable } from './enterprise-table'
import { EnterprisePrimaryButtons } from './enterprise-primary-buttons'
import { EnterpriseDrawer } from './enterprise-drawer'
import { EnterpriseDeleteDialog } from './enterprise-delete-dialog'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function EnterpriseSection() {
  const { t } = useTranslation()
  const { enterpriseOpen, setEnterpriseOpen, currentEnterprise } = useEnterprisePricing()

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Enterprise Management')}</h3>
        <EnterprisePrimaryButtons />
      </div>
      <EnterpriseTable />
      <EnterpriseDrawer
        open={enterpriseOpen === 'create' || enterpriseOpen === 'update'}
        onOpenChange={(isOpen) => !isOpen && setEnterpriseOpen(null)}
        currentRow={enterpriseOpen === 'update' ? currentEnterprise || undefined : undefined}
      />
      <EnterpriseDeleteDialog />
    </>
  )
}
