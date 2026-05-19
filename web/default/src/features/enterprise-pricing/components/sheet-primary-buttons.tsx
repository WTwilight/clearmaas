import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'

type SheetPrimaryButtonsProps = {
  onCreate?: () => void
}

export function SheetPrimaryButtons({ onCreate }: SheetPrimaryButtonsProps) {
  const { t } = useTranslation()

  return (
    <div className='flex gap-2'>
      <Button size='sm' onClick={onCreate}>
        <Plus className='h-4 w-4' />
        {t('Add Pricing Sheet')}
      </Button>
    </div>
  )
}
