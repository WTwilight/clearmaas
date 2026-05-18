import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { DataTableColumnHeader } from '@/components/data-table'
import { LongText } from '@/components/long-text'
import { DISCOUNT_TYPES } from '../constants'
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
      accessorKey: 'model',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Model')} />
      ),
      cell: ({ row }) => {
        const model = row.getValue('model') as string
        return (
          <div className='min-w-[140px]'>
            <LongText className='max-w-[180px] font-medium'>{model}</LongText>
          </div>
        )
      },
      meta: { label: t('Model'), mobileTitle: true },
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
