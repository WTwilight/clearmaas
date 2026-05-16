import { type Row } from '@tanstack/react-table'
import { MoreHorizontal, Unlink } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import type { EnterpriseUserWithBinding } from '../types'

interface BindingRowActionsProps {
  row: Row<EnterpriseUserWithBinding>
}

export function BindingRowActions({ row }: BindingRowActionsProps) {
  const { t } = useTranslation()
  const binding = row.original
  const { setBindingOpen, setCurrentBindingId } = useEnterprisePricing()

  const handleUnbind = () => {
    setCurrentBindingId(binding.user_id)
    setBindingOpen('unbind')
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant='ghost'
            className='data-popup-open:bg-muted flex h-8 w-8 p-0'
          />
        }
      >
        <MoreHorizontal className='h-4 w-4' />
        <span className='sr-only'>{t('Open menu')}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-[160px]'>
        <DropdownMenuItem
          onClick={handleUnbind}
          className='text-destructive focus:text-destructive'
        >
          <Unlink size={16} className='mr-2' />
          {t('Unbind')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
