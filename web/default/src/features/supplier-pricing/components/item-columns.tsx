import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { DataTableColumnHeader } from '@/components/data-table'
import { DISCOUNT_TYPE_LABELS, getVendorLabel } from '../constants'
import type { SupplierPricingItem } from '../types'
import { ItemRowActions } from './item-row-actions'

export function useItemColumns(): ColumnDef<SupplierPricingItem>[] {
  const { t } = useTranslation()

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
      accessorKey: 'vendor_type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Vendor Type')} />
      ),
      cell: ({ row }) => {
        const vendor = row.original.vendor_type
        return (
          <span className='text-sm'>
            {vendor ? getVendorLabel(vendor) : '-'}
          </span>
        )
      },
      meta: { label: t('Vendor Type'), mobileHidden: true },
    },
    {
      accessorKey: 'models',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Models')} />
      ),
      cell: ({ row }) => {
        const models = row.original.models as string[]
        return (
          <div className='flex flex-wrap gap-1'>
            {models?.map((m) => (
              <span key={m} className='inline-block rounded bg-muted px-1.5 py-0.5 text-xs font-medium'>
                {m}
              </span>
            ))}
          </div>
        )
      },
      meta: { label: t('Models'), mobileTitle: true },
    },
    {
      accessorKey: 'discount_type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Cost Type')} />
      ),
      cell: ({ row }) => {
        const type = row.original.discount_type
        return (
          <span className='text-sm'>
            {DISCOUNT_TYPE_LABELS[type] ?? type}
          </span>
        )
      },
      meta: { label: t('Cost Type'), mobileHidden: true },
    },
    {
      accessorKey: 'discount_value',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Cost Value')} />
      ),
      cell: ({ row }) => {
        const type = row.original.discount_type
        const value = row.original.discount_value
        let display = `$${value.toFixed(6)}`
        if (type === 'ratio') display = `${(value * 100).toFixed(0)}%`
        if (type === 'per_call') display = `$${value.toFixed(4)}/call`
        return <span className='text-sm font-medium'>{display}</span>
      },
      meta: { label: t('Cost Value'), mobileHidden: true },
    },
    {
      accessorKey: 'remark',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Remark')} />
      ),
      cell: ({ row }) => {
        const remark = row.original.remark
        return remark ? (
          <span className='text-muted-foreground truncate text-sm max-w-[200px] block'>{remark}</span>
        ) : (
          <span className='text-muted-foreground text-sm'>-</span>
        )
      },
      meta: { label: t('Remark'), mobileHidden: true },
    },
    {
      id: 'actions',
      cell: ({ row }) => <ItemRowActions row={row} />,
      meta: { label: t('Actions') },
    },
  ]
}
