import { type Row } from '@tanstack/react-table'
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  FileText,
  Link2,
  Network,
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
import type { PricingSheet, PricingSheetWithEnterprise } from '../types'

type SheetRowData = PricingSheet | PricingSheetWithEnterprise

interface SheetRowActionsProps {
  row: Row<SheetRowData>
}

export function SheetRowActions({ row }: SheetRowActionsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const sheet = row.original
  const { setSheetOpen, setCurrentSheet, setSelectedEnterpriseId, setBindingOpen, setChannelBindingOpen } =
    useEnterprisePricing()

  const handleEdit = () => {
    setCurrentSheet(sheet)
    setSheetOpen('update')
  }

  const handleDelete = () => {
    setCurrentSheet(sheet)
    setSheetOpen('delete')
  }

  const handleManageItems = () => {
    setSelectedEnterpriseId(sheet.enterprise_id)
    navigate({
      to: '/enterprise-pricing/items',
      search: { sheetId: sheet.id },
    })
  }

  const handleBindUsers = () => {
    setSelectedEnterpriseId(sheet.enterprise_id)
    setCurrentSheet(sheet)
    setBindingOpen('bind')
  }

  const handleBindChannels = () => {
    setCurrentSheet(sheet)
    setChannelBindingOpen('bindChannels')
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
        <DropdownMenuItem onClick={handleManageItems}>
          <FileText size={16} className='mr-2' />
          {t('Manage Items')}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={handleBindUsers}>
          <Link2 size={16} className='mr-2' />
          {t('Bind Users')}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={handleBindChannels}>
          <Network size={16} className='mr-2' />
          {t('Bind Channels')}
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
