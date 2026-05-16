import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
import { DataTableColumnHeader } from '@/components/data-table'
import { LongText } from '@/components/long-text'
import type { EnterpriseUserWithBinding } from '../types'
import { BindingRowActions } from './binding-row-actions'

export function useBindingColumns(): ColumnDef<EnterpriseUserWithBinding>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'user_id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title='User ID' />
      ),
      cell: ({ row }) => <div>{row.original.user_id}</div>,
      meta: { label: t('User ID'), mobileHidden: true },
    },
    {
      accessorKey: 'username',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Username')} />
      ),
      cell: ({ row }) => {
        const username = row.original.username
        const displayName = row.original.display_name
        return (
          <div className='flex min-w-[140px] flex-col gap-0.5'>
            <LongText className='max-w-[160px] font-medium'>{username}</LongText>
            {displayName && displayName !== username && (
              <LongText className='text-muted-foreground max-w-[180px] text-xs'>
                {displayName}
              </LongText>
            )}
          </div>
        )
      },
      meta: { label: t('Username'), mobileTitle: true },
    },
    {
      accessorKey: 'created_at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Bound At')} />
      ),
      cell: ({ row }) => {
        const ts = row.original.created_at
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      meta: { label: t('Bound At'), mobileHidden: true },
    },
    {
      id: 'actions',
      cell: ({ row }) => <BindingRowActions row={row} />,
      meta: { label: t('Actions') },
    },
  ]
}
