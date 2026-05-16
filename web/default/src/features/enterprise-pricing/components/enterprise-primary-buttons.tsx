import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function EnterprisePrimaryButtons() {
  const { t } = useTranslation()
  const { setEnterpriseOpen, setCurrentEnterprise } = useEnterprisePricing()

  const handleCreate = () => {
    setCurrentEnterprise(null)
    setEnterpriseOpen('create')
  }

  return (
    <div className='flex gap-2'>
      <Button size='sm' onClick={handleCreate}>
        <Plus className='h-4 w-4' />
        {t('Add Enterprise')}
      </Button>
    </div>
  )
}
