import { type Row } from '@tanstack/react-table'
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  ArrowRight,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import type { Enterprise } from '../types'

interface EnterpriseRowActionsProps {
  row: Row<Enterprise>
}

export function EnterpriseRowActions({ row }: EnterpriseRowActionsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const enterprise = row.original
  const { setEnterpriseOpen, setCurrentEnterprise, setSelectedEnterpriseId } =
    useEnterprisePricing()

  const handleEdit = () => {
    setCurrentEnterprise(enterprise)
    setEnterpriseOpen('update')
  }

  const handleDelete = () => {
    setCurrentEnterprise(enterprise)
    setEnterpriseOpen('delete')
  }

  const handleManagePricingSheets = () => {
    setSelectedEnterpriseId(enterprise.id)
    navigate({
      to: '/enterprise-pricing/sheets',
      search: { enterpriseId: enterprise.id },
    })
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
      <DropdownMenuContent align='end' className='w-[200px]'>
        <DropdownMenuItem onClick={handleEdit}>
          {t('Edit')}
          <DropdownMenuShortcut>
            <Pencil size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={handleManagePricingSheets}>
          {t('Manage Pricing Sheets')}
          <DropdownMenuShortcut>
            <ArrowRight size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={handleDelete}
          className='text-destructive focus:text-destructive'
        >
          {t('Delete')}
          <DropdownMenuShortcut>
            <Trash2 size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
