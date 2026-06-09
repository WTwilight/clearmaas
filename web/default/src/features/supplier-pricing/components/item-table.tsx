import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import {
  type SortingState,
  type VisibilityState,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useMediaQuery } from '@/hooks'
import { toast } from 'sonner'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { DataTablePage } from '@/components/data-table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { getSuppliers, getSupplierPricingSheetsBySupplier, getSupplierPricingItems } from '../api'
import { supplierItemQueryKeys } from '../lib/query-keys'
import { useItemColumns } from './item-columns'
import { useSupplierPricing } from './supplier-pricing-provider'

const route = getRouteApi('/_authenticated/supplier-pricing/items/')

export function ItemTable() {
  const { t } = useTranslation()
  const columns = useItemColumns()
  const { itemRefreshTrigger } = useSupplierPricing()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})

  const routeSearch = route.useSearch()
  const navigate = route.useNavigate()

  const currentSupplierId = routeSearch.supplierId
  const currentSheetId = routeSearch.sheetId ?? null

  const {
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: routeSearch,
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
  })

  const { data: suppliersData, isLoading: isLoadingSuppliers } = useQuery({
    queryKey: ['suppliers', 'item-selector'],
    queryFn: async () => {
      const result = await getSuppliers({ p: 1, page_size: 1000 })
      return result.data?.items ?? []
    },
    staleTime: 5 * 60 * 1000,
  })

  const { data: sheetsData, isLoading: isLoadingSheets } = useQuery({
    queryKey: ['supplier-sheets', 'selector', currentSupplierId],
    queryFn: async () => {
      if (!currentSupplierId) return []
      const result = await getSupplierPricingSheetsBySupplier(currentSupplierId)
      return result.items ?? []
    },
    enabled: !!currentSupplierId,
    staleTime: 5 * 60 * 1000,
  })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      ...supplierItemQueryKeys.bySheet(currentSupplierId ?? 0, currentSheetId ?? 0),
      pagination.pageIndex + 1,
      pagination.pageSize,
      itemRefreshTrigger,
    ],
    queryFn: async () => {
      if (!currentSupplierId || !currentSheetId) return { items: [], total: 0 }
      const result = await getSupplierPricingItems(currentSupplierId, currentSheetId, {
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to load pricing items'))
        return { items: [], total: 0 }
      }
      return {
        items: result.data?.items ?? [],
        total: result.data?.total ?? 0,
      }
    },
    enabled: !!currentSupplierId && !!currentSheetId,
    placeholderData: (previousData) => previousData,
  })

  const items = data?.items ?? []

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
    pageCount: Math.ceil((data?.total ?? 0) / pagination.pageSize),
  })

  const pageCount = table.getPageCount()
  useEffect(() => {
    ensurePageInRange(pageCount)
  }, [pageCount, ensurePageInRange])

  if (isLoadingSuppliers || isLoadingSheets) {
    return <Skeleton className='h-10 w-[300px]' />
  }

  if (suppliersData && suppliersData.length === 0) {
    return (
      <div className='rounded-lg border p-8 text-center text-sm text-muted-foreground'>
        {t('Please create a supplier and pricing sheet first')}
      </div>
    )
  }

  const selectedSupplier = suppliersData?.find((s: { id: number }) => s.id === currentSupplierId)
  const selectedSheet = sheetsData?.find((s: { id: number }) => s.id === currentSheetId)

  const handleSupplierChange = (v: string | null) => {
    const supplierId = parseInt(v ?? '0') || undefined
    navigate({ search: { supplierId, sheetId: undefined, page: 1, pageSize: pagination.pageSize } })
  }

  const handleSheetChange = (v: string | null) => {
    const sheetId = parseInt(v ?? '0') || undefined
    navigate({ search: { supplierId: currentSupplierId, sheetId, page: 1, pageSize: pagination.pageSize } })
  }

  const toolbar = (
    <div className='flex flex-wrap items-center gap-3'>
      <div className='flex items-center gap-2'>
        <Label>{t('Supplier')}:</Label>
        <Select
          value={String(currentSupplierId ?? '')}
          onValueChange={handleSupplierChange}
        >
          <SelectTrigger className='w-[160px]'>
            <SelectValue>{selectedSupplier?.name ?? ''}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {suppliersData?.map((s: { id: number; name: string }) => (
              <SelectItem key={s.id} value={String(s.id)}>
                {s.name}
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
          disabled={!currentSupplierId || !sheetsData?.length}
        >
          <SelectTrigger className='w-[160px]'>
            <SelectValue>{selectedSheet?.name ?? ''}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {sheetsData?.map((s: { id: number; name: string }) => (
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
