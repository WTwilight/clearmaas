/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { type Table } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

type DataTableViewOptionsProps<TData> = {
  table: Table<TData>
}

export function DataTableViewOptions<TData>({
  table,
}: DataTableViewOptionsProps<TData>) {
  const { t } = useTranslation()

  const hideableColumns = table
    .getAllColumns()
    .filter((column) => typeof column.accessorFn !== 'undefined' && column.getCanHide())

  const showAll = () => hideableColumns.forEach((col) => col.setVisibility(true))
  const hideAll = () => hideableColumns.forEach((col) => col.setVisibility(false))

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={
          <Button
            variant='outline'
            className='shrink-0'
            aria-label={t('View')}
          />
        }
      >
        {t('View')}
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-[180px]'>
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t('Toggle columns')}</DropdownMenuLabel>
          <div className='flex gap-1 px-2 py-1'>
            <Button
              variant='ghost'
              size='sm'
              className='h-7 flex-1 text-xs'
              onClick={showAll}
            >
              {t('Show all')}
            </Button>
            <Button
              variant='ghost'
              size='sm'
              className='h-7 flex-1 text-xs'
              onClick={hideAll}
            >
              {t('Hide all')}
            </Button>
          </div>
          <DropdownMenuSeparator />
          {hideableColumns.map((column) => {
            return (
              <DropdownMenuCheckboxItem
                key={column.id}
                className='capitalize'
                checked={column.getIsVisible()}
                onCheckedChange={(value) => column.toggleVisibility(!!value)}
              >
                {column.columnDef.meta?.label ?? column.id}
              </DropdownMenuCheckboxItem>
            )
          })}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
