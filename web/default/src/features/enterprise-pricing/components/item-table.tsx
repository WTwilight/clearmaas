import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  type SortingState,
  type VisibilityState,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { getRouteApi } from '@tanstack/react-router'
import { useMediaQuery } from '@/hooks'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import type { NavigateFn } from '@/hooks/use-table-url-state'
import { DataTablePage } from '@/components/data-table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { getPricingItems, getEnterprises, getSheets, getAllSheets } from '../api'
import { useItemColumns } from './item-columns'
import { useEnterprisePricing } from './enterprise-pricing-provider'

const route = getRouteApi('/_authenticated/enterprise-pricing/items/')

type ItemTableProps = {
  urlSheetId?: number
  onSheetIdChange: (sheetId: number | null) => void
}

export function ItemTable({ urlSheetId, onSheetIdChange }: ItemTableProps) {
  const { t } = useTranslation()
  const columns = useItemColumns()
  const { itemRefreshTrigger, selectedEnterpriseId, setSelectedEnterpriseId } = useEnterprisePricing()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})

  const routeSearch = route.useSearch()
  const [currentSheetId, setCurrentSheetId] = useState<number | null>(null)

  const {
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: routeSearch,
    navigate: route.useNavigate() as NavigateFn,
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
  })

  useEffect(() => {
    onSheetIdChange(currentSheetId)
  }, [currentSheetId, onSheetIdChange])

  const { data: enterprisesData, isLoading: isLoadingEnterprises } = useQuery({
    queryKey: ['enterprises', 'item-selector'],
    queryFn: async () => {
      const [entResult, sheetsResult] = await Promise.all([
        getEnterprises({ p: 1, page_size: 1000 }),
        getAllSheets({ p: 1, page_size: 10000 }),
      ])
      const enterprises = entResult.data?.items || []
      const sheets = sheetsResult.data?.items || []

      const enterpriseIdsWithSheets = new Set(sheets.map((s) => s.enterprise_id))

      return enterprises.filter((e) => enterpriseIdsWithSheets.has(e.id))
    },
    staleTime: 0,
  })

  const { data: sheetsData, isLoading: isLoadingSheets } = useQuery({
    queryKey: ['sheets', 'selector', selectedEnterpriseId],
    queryFn: async () => {
      if (!selectedEnterpriseId) return []
      const result = await getSheets(selectedEnterpriseId, { p: 1, page_size: 1000 })
      return result.data?.items || []
    },
    enabled: !!selectedEnterpriseId,
    staleTime: 0,
  })

  useEffect(() => {
    if (urlSheetId !== undefined) {
      setCurrentSheetId(urlSheetId)
    }
  }, [urlSheetId])

  useEffect(() => {
    if (!selectedEnterpriseId && enterprisesData && enterprisesData.length > 0 && !urlSheetId) {
      setSelectedEnterpriseId(enterprisesData[0].id)
    }
  }, [enterprisesData, selectedEnterpriseId, setSelectedEnterpriseId, urlSheetId])

  useEffect(() => {
    if (!currentSheetId && sheetsData && sheetsData.length > 0) {
      const hasUrlSheet = urlSheetId && sheetsData.some((s) => s.id === urlSheetId)
      if (hasUrlSheet) {
        setCurrentSheetId(urlSheetId)
      } else {
        setCurrentSheetId(sheetsData[0].id)
      }
    }
  }, [sheetsData, currentSheetId, urlSheetId])

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'pricing-items',
      currentSheetId,
      pagination.pageIndex + 1,
      pagination.pageSize,
      itemRefreshTrigger,
    ],
    queryFn: async () => {
      if (!currentSheetId) return { items: [], total: 0 }

      const result = await getPricingItems(currentSheetId, {
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      })

      if (!result.success) {
        toast.error(result.message || t('Failed to load pricing items'))
        return { items: [], total: 0 }
      }

      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
      }
    },
    enabled: !!currentSheetId,
    placeholderData: (previousData) => previousData,
  })

  const items = data?.items || []

  const table = useReactTable({
    data: items,
    columns,
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      pagination,
    },
    enableRowSelection: false,
    manualPagination: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    pageCount: Math.ceil((data?.total || 0) / pagination.pageSize),
  })

  const pageCount = table.getPageCount()
  useEffect(() => {
    ensurePageInRange(pageCount)
  }, [pageCount, ensurePageInRange])

  if (isLoadingEnterprises || isLoadingSheets) {
    return <Skeleton className='h-10 w-[300px]' />
  }

  if (enterprisesData && enterprisesData.length === 0) {
    return (
      <div className='rounded-lg border p-8 text-center text-sm text-muted-foreground'>
        {t('Please create an enterprise and pricing sheet first')}
      </div>
    )
  }

  const selectedEnterprise = enterprisesData?.find((e) => e.id === selectedEnterpriseId)
  const selectedSheet = sheetsData?.find((s) => s.id === currentSheetId)

  const handleEnterpriseChange = (v: string | null) => {
    const val = v ?? ''
    setSelectedEnterpriseId(parseInt(val) || 0)
    setCurrentSheetId(null)
    onPaginationChange({ pageIndex: 0, pageSize: pagination.pageSize })
  }

  const handleSheetChange = (v: string | null) => {
    const val = v ?? ''
    const sheetId = parseInt(val) || null
    setCurrentSheetId(sheetId)
    onPaginationChange({ pageIndex: 0, pageSize: pagination.pageSize })
  }

  const toolbar = (
    <div className='flex flex-wrap items-center gap-3'>
      <div className='flex items-center gap-2'>
        <Label>{t('Enterprise')}:</Label>
        <Select
          value={String(selectedEnterpriseId ?? '')}
          onValueChange={handleEnterpriseChange}
        >
          <SelectTrigger className='w-[180px]'>
            <SelectValue>{selectedEnterprise?.name ?? ''}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {enterprisesData?.map((e) => (
              <SelectItem key={e.id} value={String(e.id)}>
                {e.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className='flex items-center gap-2'>
        <Label>{t('Pricing Sheet')}:</Label>
        <Select
          value={String(currentSheetId ?? '')}
          onValueChange={handleSheetChange}
          disabled={!selectedEnterpriseId || !sheetsData?.length}
        >
          <SelectTrigger className='w-[180px]'>
            <SelectValue>{selectedSheet?.name ?? ''}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {sheetsData?.map((s) => (
              <SelectItem key={s.id} value={String(s.id)}>
                {s.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Pricing Items Found')}
      emptyDescription={t('No pricing items found for this sheet.')}
      skeletonKeyPrefix='item-skeleton'
      toolbar={toolbar}
      paginationInFooter={false}
    />
  )
}
