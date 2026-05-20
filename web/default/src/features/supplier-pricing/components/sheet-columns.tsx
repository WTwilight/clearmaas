import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { SHEET_STATUS_LABELS } from '../constants'
import type { SupplierPricingSheet } from '../types'
import { SheetRowActions } from './sheet-row-actions'

export function useSheetColumns(showSupplier = false): ColumnDef<SupplierPricingSheet>[] {
  const { t } = useTranslation()

  const baseColumns: ColumnDef<SupplierPricingSheet>[] = [
    {
      accessorKey: 'id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title='ID' />
      ),
      cell: ({ row }) => <div className='w-[60px]'>{row.getValue('id')}</div>,
      meta: { label: t('ID'), mobileHidden: true },
    },
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Name')} />
      ),
      cell: ({ row }) => {
        const name = row.getValue('name') as string
        return <span className='font-medium'>{name}</span>
      },
      meta: { label: t('Name') },
    },
    {
      accessorKey: 'channel_id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Channels')} />
      ),
      cell: ({ row }) => {
        const channelNames = row.original.channel_names
        if (!channelNames || channelNames.length === 0) {
          return <span className='text-muted-foreground text-sm'>-</span>
        }
        return (
          <div className='flex flex-wrap gap-1'>
            {channelNames.map((name, i) => (
              <span key={i} className='inline-flex items-center rounded bg-muted px-1.5 py-0.5 text-xs font-medium'>
                {name}
              </span>
            ))}
          </div>
        )
      },
      meta: { label: t('Channels'), mobileHidden: true },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) => {
        const status = row.original.status
        const label = SHEET_STATUS_LABELS[status] ?? String(status)
        const variant = status === 1 ? 'success' : 'neutral'
        return (
          <StatusBadge
            label={label}
            variant={variant}
            showDot
            copyable={false}
          />
        )
      },
      meta: { label: t('Status'), mobileBadge: true },
    },
    {
      accessorKey: 'start_time',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Start Time')} />
      ),
      cell: ({ row }) => {
        const ts = row.original.start_time
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('Start Time'), mobileHidden: true },
    },
    {
      id: 'actions',
      cell: ({ row }) => <SheetRowActions row={row} />,
      meta: { label: t('Actions') },
    },
  ]

  if (showSupplier) {
    const supplierCol: ColumnDef<SupplierPricingSheet> = {
      accessorKey: 'supplier_name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Supplier')} />
      ),
      cell: ({ row }) => {
        const name = row.original.supplier_name
        return <span className='font-medium'>{name ?? '-'}</span>
      },
      meta: { label: t('Supplier'), mobileTitle: true },
    }
    baseColumns.splice(1, 0, supplierCol)
  }

  return baseColumns
}
