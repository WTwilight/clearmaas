import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  type SortingState,
  type VisibilityState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useMediaQuery } from '@/hooks'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import type { NavigateFn } from '@/hooks/use-table-url-state'
import { DataTablePage } from '@/components/data-table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getAllSheets, getEnterprises } from '../api'
import { useSheetColumns } from './sheet-columns'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { getRouteApi } from '@tanstack/react-router'

const route = getRouteApi('/_authenticated/enterprise-pricing/sheets/')

// Selector option for "all enterprises"
const ALL_ENTERPRISES_VALUE = 'all'

type SheetTableProps = {
  onEnterpriseFilterChange?: (value: string | null) => void
}

export function SheetTable({ onEnterpriseFilterChange }: SheetTableProps) {
  const { t } = useTranslation()
  const { sheetRefreshTrigger } = useEnterprisePricing()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})

  const routeSearch = route.useSearch()
  const [filterEnterpriseId, setFilterEnterpriseId] = useState<string | null>(
    routeSearch.enterpriseId ? String(routeSearch.enterpriseId) : ALL_ENTERPRISES_VALUE
  )

  const {
    globalFilter,
    onGlobalFilterChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate() as NavigateFn,
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
    globalFilter: { enabled: true, key: 'filter' },
  })

  // Sync filterEnterpriseId when route search changes (e.g. navigating from enterprise management)
  useEffect(() => {
    if (routeSearch.enterpriseId) {
      setFilterEnterpriseId(String(routeSearch.enterpriseId))
    }
  }, [routeSearch.enterpriseId])

  // Determine if we should show enterprise column
  const showEnterpriseColumn = filterEnterpriseId === null || filterEnterpriseId === ALL_ENTERPRISES_VALUE
  const columns = useSheetColumns(showEnterpriseColumn)

  // Fetch enterprises for selector
  const { data: enterprisesData, isLoading: isLoadingEnterprises } = useQuery({
    queryKey: ['enterprises', 'selector'],
    queryFn: async () => {
      const result = await getEnterprises({ p: 1, page_size: 1000 })
      return result.data?.items || []
    },
    staleTime: 5 * 60 * 1000,
  })

  // Determine which enterprise to filter by
  const effectiveEnterpriseId = (filterEnterpriseId === null || filterEnterpriseId === ALL_ENTERPRISES_VALUE)
    ? undefined
    : parseInt(filterEnterpriseId)

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'sheets',
      'all',
      effectiveEnterpriseId,
      pagination.pageIndex + 1,
      pagination.pageSize,
      sheetRefreshTrigger,
    ],
    queryFn: async () => {
      const result = await getAllSheets({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        enterprise_id: effectiveEnterpriseId,
      })

      if (!result.success) {
        toast.error(t('Failed to load pricing sheets'))
        return { items: [], total: 0 }
      }

      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const sheets = data?.items || []

  const table = useReactTable({
    data: sheets,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      globalFilter,
      pagination,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    globalFilterFn: (row, _columnId, filterValue) => {
      const searchValue = String(filterValue).toLowerCase()
      const sheet = row.original
      const name = sheet.name || ''
      const enterpriseName = 'enterprise_name' in sheet ? sheet.enterprise_name : ''
      return [name, enterpriseName].some((field) =>
        String(field || '').toLowerCase().includes(searchValue)
      )
    },
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    onPaginationChange,
    onGlobalFilterChange,
    pageCount: Math.ceil((data?.total || 0) / pagination.pageSize),
  })

  const pageCount = table.getPageCount()
  useEffect(() => {
    ensurePageInRange(pageCount)
  }, [pageCount, ensurePageInRange])

  if (isLoadingEnterprises) {
    return <Skeleton className='h-10 w-[200px]' />
  }

  const handleEnterpriseFilterChange = (value: string | null) => {
    setFilterEnterpriseId(value ?? ALL_ENTERPRISES_VALUE)
    onPaginationChange({ pageIndex: 0, pageSize: pagination.pageSize })
    onEnterpriseFilterChange?.(value ?? ALL_ENTERPRISES_VALUE)
  }

  const selectedEnterpriseName = filterEnterpriseId === ALL_ENTERPRISES_VALUE
    ? t('All Enterprises')
    : enterprisesData?.find((e) => String(e.id) === filterEnterpriseId)?.name ?? ''

  const toolbar = (
    <div className='flex flex-wrap items-center gap-2'>
      <Select
        value={filterEnterpriseId ?? ALL_ENTERPRISES_VALUE}
        onValueChange={handleEnterpriseFilterChange}
      >
        <SelectTrigger className='w-[200px]'>
          <SelectValue>{selectedEnterpriseName}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL_ENTERPRISES_VALUE}>
            {t('All Enterprises')}
          </SelectItem>
          {enterprisesData?.map((e) => (
            <SelectItem key={e.id} value={String(e.id)}>
              {e.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <div className='ms-auto'>
        <input
          placeholder={t('Filter by name...')}
          value={globalFilter}
          onChange={(e) => onGlobalFilterChange?.(e.target.value)}
          className='h-9 w-full rounded-md border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sm:w-[200px] lg:w-[240px]'
        />
      </div>
    </div>
  )

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Pricing Sheets Found')}
      emptyDescription={t(
        'No pricing sheets found.'
      )}
      skeletonKeyPrefix='sheet-skeleton'
      toolbar={toolbar}
      paginationInFooter={false}
    />
  )
}
