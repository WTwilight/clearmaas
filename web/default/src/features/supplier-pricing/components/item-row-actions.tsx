import { type Row } from '@tanstack/react-table'
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  Eye,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
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
import type { SupplierPricingItem } from '../types'

interface ItemRowActionsProps {
  row: Row<SupplierPricingItem>
}

export function ItemRowActions({ row }: ItemRowActionsProps) {
  const { t } = useTranslation()
  const item = row.original
  const { openItemDialog } = useSupplierPricing()

  const handleEdit = () => {
    openItemDialog({ type: 'update', item })
  }

  const handleView = () => {
    openItemDialog({ type: 'view', item })
  }

  const handleDelete = () => {
    openItemDialog({ type: 'delete', item })
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
        <DropdownMenuItem onClick={handleView}>
          <Eye size={16} className='mr-2' />
          {t('View')}
        </DropdownMenuItem>
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
