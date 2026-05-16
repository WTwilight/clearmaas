import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useEnterprisePricing } from './enterprise-pricing-provider'

export function BindingPrimaryButtons() {
  const { t } = useTranslation()
  const { setBindingOpen, selectedEnterpriseId } = useEnterprisePricing()

  const handleBind = () => {
    setBindingOpen('bind')
  }

  const isDisabled = !selectedEnterpriseId

  return (
    <div className='flex gap-2'>
      <Button size='sm' onClick={handleBind} disabled={isDisabled}>
        <Plus className='h-4 w-4' />
        {t('Bind User')}
      </Button>
    </div>
  )
}
