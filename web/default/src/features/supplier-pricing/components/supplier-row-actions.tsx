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
import { useSupplierPricing } from './supplier-pricing-provider'
import type { Supplier } from '../types'

interface SupplierRowActionsProps {
  row: Row<Supplier>
}

export function SupplierRowActions({ row }: SupplierRowActionsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const supplier = row.original
  const { openSupplierDialog, setSelectedSupplierId } = useSupplierPricing()

  const handleEdit = () => {
    openSupplierDialog({ type: 'update', supplier })
  }

  const handleDelete = () => {
    openSupplierDialog({ type: 'delete', supplier })
  }

  const handleManagePricingSheets = () => {
    setSelectedSupplierId(supplier.id)
    navigate({
      to: '/supplier-pricing/sheets',
      search: { supplierId: supplier.id },
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
        <DropdownMenuItem onClick={handleManagePricingSheets}>
          {t('Manage Pricing Sheets')}
          <DropdownMenuShortcut>
            <ArrowRight size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={handleEdit}>
          {t('Edit')}
          <DropdownMenuShortcut>
            <Pencil size={16} />
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
