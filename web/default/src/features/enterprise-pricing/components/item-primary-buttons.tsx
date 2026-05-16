import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useEnterprisePricing } from './enterprise-pricing-provider'

type ItemPrimaryButtonsProps = {
  sheetId: number | null
}

export function ItemPrimaryButtons({ sheetId }: ItemPrimaryButtonsProps) {
  const { t } = useTranslation()
  const { setItemOpen } = useEnterprisePricing()

  const handleCreate = () => {
    setItemOpen('create')
  }

  const isDisabled = !sheetId

  return (
    <div className='flex gap-2'>
      <Button size='sm' onClick={handleCreate} disabled={isDisabled}>
        <Plus className='h-4 w-4' />
        {t('Add Pricing Item')}
      </Button>
    </div>
  )
}
