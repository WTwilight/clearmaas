import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { LongText } from '@/components/long-text'
import { SHEET_STATUSES } from '../constants'
import type { PricingSheet, PricingSheetWithEnterprise } from '../types'
import { SheetRowActions } from './sheet-row-actions'

type SheetRowData = PricingSheet | PricingSheetWithEnterprise

export function useSheetColumns(showEnterpriseColumn = false): ColumnDef<SheetRowData>[] {
  const { t } = useTranslation()
  const columns: ColumnDef<SheetRowData>[] = [
    {
      accessorKey: 'id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title='ID' />
      ),
      cell: ({ row }) => <div className='w-[60px]'>{row.getValue('id')}</div>,
      meta: { label: t('ID'), mobileHidden: true },
    },
  ]

  if (showEnterpriseColumn) {
    columns.push({
      accessorKey: 'enterprise_name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Enterprise')} />
      ),
      cell: ({ row }) => {
        const name = (row.original as PricingSheetWithEnterprise).enterprise_name || '-'
        return (
          <div className='min-w-[120px]'>
            <LongText className='max-w-[160px] font-medium'>{name}</LongText>
          </div>
        )
      },
      meta: { label: t('Enterprise'), mobileHidden: true },
    })
  }

  columns.push(
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Sheet Name')} />
      ),
      cell: ({ row }) => {
        const name = row.getValue('name') as string
        return (
          <div className='min-w-[140px]'>
            <LongText className='max-w-[180px] font-medium'>{name}</LongText>
          </div>
        )
      },
      meta: { label: t('Sheet Name'), mobileTitle: true },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) => {
        const status = row.getValue('status') as number
        const config = SHEET_STATUSES[status as keyof typeof SHEET_STATUSES]
        if (!config) return null
        return (
          <StatusBadge
            label={t(config.labelKey)}
            variant={config.variant}
            showDot={config.showDot}
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
        const ts = row.getValue('start_time') as number
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('Start Time'), mobileHidden: true },
    },
    {
      accessorKey: 'end_time',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('End Time')} />
      ),
      cell: ({ row }) => {
        const ts = row.getValue('end_time') as number
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('End Time'), mobileHidden: true },
    },
    {
      accessorKey: 'created_at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Created At')} />
      ),
      cell: ({ row }) => {
        const ts = row.getValue('created_at') as number | undefined
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
      cell: ({ row }) => <SheetRowActions row={row} />,
      meta: { label: t('Actions') },
    }
  )

  return columns
}
