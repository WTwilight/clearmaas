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
import { DataTablePage } from '@/components/data-table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { getPricingItems, getEnterprises, getSheets, getAllSheets } from '../api'
import { useItemColumns } from './item-columns'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyTitle, EmptyDescription } from '@/components/ui/empty'

type ItemTableProps = {
  urlSheetId?: number
  onSheetIdChange: (sheetId: number | null) => void
}

export function ItemTable({ urlSheetId, onSheetIdChange }: ItemTableProps) {
  const { t } = useTranslation()
  const columns = useItemColumns()
  const { itemRefreshTrigger, selectedEnterpriseId, setSelectedEnterpriseId } = useEnterprisePricing()
  const [currentSheetId, setCurrentSheetId] = useState<number | null>(null)
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: isMobile ? 10 : 20 })

  // Sync currentSheetId back to parent for URL synchronization
  useEffect(() => {
    onSheetIdChange(currentSheetId)
  }, [currentSheetId, onSheetIdChange])

  // Fetch all enterprises and all sheets, then filter to enterprises that have at least one sheet
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

  // Fetch sheets when enterprise changes
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

  // Sync URL sheetId → local state when URL changes
  useEffect(() => {
    if (urlSheetId !== undefined) {
      setCurrentSheetId(urlSheetId)
    }
  }, [urlSheetId])

  // Set initial enterprise (only if no URL sheetId and no enterprise selected)
  useEffect(() => {
    if (!selectedEnterpriseId && enterprisesData && enterprisesData.length > 0 && !urlSheetId) {
      setSelectedEnterpriseId(enterprisesData[0].id)
    }
  }, [enterprisesData, selectedEnterpriseId, setSelectedEnterpriseId, urlSheetId])

  // Set initial sheet: prefer URL sheetId if valid, otherwise auto-select first sheet
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
    queryKey: ['pricing-items', currentSheetId, itemRefreshTrigger],
    queryFn: async () => {
      if (!currentSheetId) return []

      const result = await getPricingItems(currentSheetId)

      if (!result.success) {
        toast.error(result.message || t('Failed to load pricing items'))
        return []
      }

      return result.data || []
    },
    enabled: !!currentSheetId,
    placeholderData: (previousData) => previousData,
  })

  const items = data || []

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
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    pageCount: Math.ceil(items.length / pagination.pageSize),
  })

  if (isLoadingEnterprises || isLoadingSheets) {
    return <Skeleton className='h-10 w-[300px]' />
  }

  if (enterprisesData && enterprisesData.length === 0) {
    return (
      <Empty>
        <EmptyTitle>{t('No Enterprises')}</EmptyTitle>
        <EmptyDescription>{t('Please create an enterprise and pricing sheet first')}</EmptyDescription>
      </Empty>
    )
  }

  const selectedEnterprise = enterprisesData?.find((e) => e.id === selectedEnterpriseId)
  const selectedSheet = sheetsData?.find((s) => s.id === currentSheetId)

  return (
    <div className='space-y-2.5'>
      <div className='flex items-center gap-3'>
        <div className='flex items-center gap-2'>
          <Label>{t('Enterprise')}:</Label>
          <Select
            value={String(selectedEnterpriseId ?? '')}
            onValueChange={(v) => {
              setSelectedEnterpriseId(parseInt(v ?? '0'))
              setCurrentSheetId(null)
            }}
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
            onValueChange={(v) => setCurrentSheetId(parseInt(v ?? '0'))}
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

      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No Pricing Items Found')}
        emptyDescription={t(
          'No pricing items found for this sheet.'
        )}
        skeletonKeyPrefix='item-skeleton'
      />
    </div>
  )
}
