import { type Row } from '@tanstack/react-table'
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  ListChecks,
  Link2,
  Eye,
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
import type { SupplierPricingSheet } from '../types'

interface SheetRowActionsProps {
  row: Row<SupplierPricingSheet>
}

export function SheetRowActions({ row }: SheetRowActionsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const sheet = row.original
  const { openSheetDialog, openSheetBindDialog, setSelectedSupplierId, setSelectedSheetId } =
    useSupplierPricing()

  const handleManageItems = () => {
    setSelectedSupplierId(sheet.supplier_id)
    setSelectedSheetId(sheet.id)
    navigate({ to: '/supplier-pricing/items' })
  }

  const handleBindChannel = () => {
    openSheetBindDialog({
      id: sheet.id,
      supplier_id: sheet.supplier_id,
      name: sheet.name,
      channel_id: sheet.channel_id,
    })
  }

  const handleEdit = () => {
    openSheetDialog({ type: 'update', sheet, supplierId: sheet.supplier_id })
  }

  const handleView = () => {
    openSheetDialog({ type: 'view', sheet, supplierId: sheet.supplier_id })
  }

  const handleDelete = () => {
    openSheetDialog({ type: 'delete', sheet, supplierId: sheet.supplier_id })
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
        <DropdownMenuItem onClick={handleView}>
          <Eye size={16} className='mr-2' />
          {t('View')}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={handleManageItems}>
          <ListChecks size={16} className='mr-2' />
          {t('Manage Items')}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={handleBindChannel}>
          <Link2 size={16} className='mr-2' />
          {t('Bind Channel')}
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
