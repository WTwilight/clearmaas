import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { DataTableColumnHeader } from '@/components/data-table'
import { LongText } from '@/components/long-text'
import { Badge } from '@/components/ui/badge'
import { DISCOUNT_TYPES, getVendorLabel } from '../constants'
import type { PricingItem } from '../types'
import { ItemRowActions } from './item-row-actions'

export function useItemColumns(): ColumnDef<PricingItem>[] {
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
        <DataTableColumnHeader column={column} title={t('Vendor')} />
      ),
      cell: ({ row }) => {
        const vendorType = row.original.vendor_type
        const label = vendorType ? getVendorLabel(vendorType) : null
        if (!label) return <span className='text-muted-foreground text-sm'>-</span>
        return <span className='text-xs font-medium'>{label}</span>
      },
      meta: { label: t('Vendor'), mobileHidden: true },
    },
    {
      accessorKey: 'models',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Models')} />
      ),
      cell: ({ row }) => {
        const models = row.original.models
        if (!models || models.length === 0) {
          return <span className='text-muted-foreground text-sm'>-</span>
        }
        if (models.length === 1) {
          return (
            <LongText className='max-w-[200px] font-medium'>
              {models[0]}
            </LongText>
          )
        }
        return (
          <div className='flex flex-wrap gap-1'>
            {models.slice(0, 3).map((m) => (
              <Badge key={m} variant='secondary' className='text-xs'>
                {m}
              </Badge>
            ))}
            {models.length > 3 && (
              <Badge variant='outline' className='text-xs'>
                +{models.length - 3}
              </Badge>
            )}
          </div>
        )
      },
      meta: { label: t('Models'), mobileTitle: true },
    },
    {
      accessorKey: 'discount_type',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Discount Type')} />
      ),
      cell: ({ row }) => {
        const type = row.getValue('discount_type') as string
        const config = DISCOUNT_TYPES[type as keyof typeof DISCOUNT_TYPES]
        return (
          <span className='text-sm'>
            {config ? t(config.labelKey) : type}
          </span>
        )
      },
      meta: { label: t('Discount Type') },
    },
    {
      accessorKey: 'discount_value',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Discount Value')} />
      ),
      cell: ({ row }) => {
        const type = row.original.discount_type
        const value = row.getValue('discount_value') as number
        if (type === 'ratio') {
          return <span className='font-medium'>{value.toFixed(2)}x</span>
        }
        if (type === 'per_call') {
          return <span className='font-medium'>${value.toFixed(4)}/call</span>
        }
        return <span className='font-medium'>${value.toFixed(4)}</span>
      },
      meta: { label: t('Discount Value') },
    },
    {
      accessorKey: 'remark',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Remark')} />
      ),
      cell: ({ row }) => {
        const remark = row.getValue('remark') as string | undefined
        return (
          <LongText className='text-muted-foreground max-w-[200px] text-sm'>
            {remark || '-'}
          </LongText>
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
