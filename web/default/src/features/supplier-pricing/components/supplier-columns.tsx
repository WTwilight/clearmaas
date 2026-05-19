import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { SUPPLIER_STATUS_OPTIONS } from '../constants'
import type { Supplier } from '../types'
import { SupplierRowActions } from './supplier-row-actions'

export function useSupplierColumns(): ColumnDef<Supplier>[] {
  const { t } = useTranslation()

  const statusMap = Object.fromEntries(
    SUPPLIER_STATUS_OPTIONS.map((o) => [o.value, o.label])
  )

  return [
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
        const remark = row.original.remark
        return (
          <div className='flex min-w-[160px] flex-col gap-1'>
            <span className='font-medium'>{name}</span>
            {remark && (
              <span className='text-muted-foreground truncate text-xs max-w-[200px]'>{remark}</span>
            )}
          </div>
        )
      },
      meta: { label: t('Name'), mobileTitle: true },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) => {
        const status = row.original.status
        const label = statusMap[status] ?? String(status)
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
      accessorKey: 'created_at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Created At')} />
      ),
      cell: ({ row }) => {
        const ts = row.original.created_at
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('Created At'), mobileHidden: true },
    },
    {
      id: 'actions',
      cell: ({ row }) => <SupplierRowActions row={row} />,
      meta: { label: t('Actions') },
    },
  ]
}
