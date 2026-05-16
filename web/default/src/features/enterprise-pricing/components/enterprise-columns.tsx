import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { LongText } from '@/components/long-text'
import { ENTERPRISE_STATUSES } from '../constants'
import type { Enterprise } from '../types'
import { EnterpriseRowActions } from './enterprise-row-actions'

export function useEnterpriseColumns(): ColumnDef<Enterprise>[] {
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
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Enterprise Name')} />
      ),
      cell: ({ row }) => {
        const name = row.getValue('name') as string
        const remark = row.original.remark
        return (
          <div className='flex min-w-[160px] flex-col gap-1'>
            <LongText className='max-w-[180px] font-medium'>{name}</LongText>
            {remark && (
              <LongText className='text-muted-foreground max-w-[200px] text-xs'>
                {remark}
              </LongText>
            )}
          </div>
        )
      },
      meta: { label: t('Enterprise Name'), mobileTitle: true },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) => {
        const status = row.getValue('status') as number
        const config = ENTERPRISE_STATUSES[status as keyof typeof ENTERPRISE_STATUSES]
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
      accessorKey: 'updated_at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Updated At')} />
      ),
      cell: ({ row }) => {
        const ts = row.getValue('updated_at') as number | undefined
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('Updated At'), mobileHidden: true },
    },
    {
      id: 'actions',
      cell: ({ row }) => <EnterpriseRowActions row={row} />,
      meta: { label: t('Actions') },
    },
  ]
}
