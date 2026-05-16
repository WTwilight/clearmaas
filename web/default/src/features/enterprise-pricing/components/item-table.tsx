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
import { getPricingItems, getEnterprises, getSheets } from '../api'
import { useItemColumns } from './item-columns'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyTitle, EmptyDescription } from '@/components/ui/empty'

type ItemTableProps = {
  onSheetIdChange?: (sheetId: number | null) => void
}

export function ItemTable({ onSheetIdChange }: ItemTableProps) {
  const { t } = useTranslation()
  const columns = useItemColumns()
  const { itemRefreshTrigger, selectedEnterpriseId, setSelectedEnterpriseId } = useEnterprisePricing()
  const [currentSheetId, setCurrentSheetId] = useState<number | null>(null)
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: isMobile ? 10 : 20 })

  // Notify parent of sheet changes
  useEffect(() => {
    onSheetIdChange?.(currentSheetId)
  }, [currentSheetId, onSheetIdChange])

  // Fetch enterprises
  const { data: enterprisesData, isLoading: isLoadingEnterprises } = useQuery({
    queryKey: ['enterprises', 'item-selector'],
    queryFn: async () => {
      const result = await getEnterprises({ p: 1, page_size: 1000 })
      return result.data?.items || []
    },
    staleTime: 5 * 60 * 1000,
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
    staleTime: 5 * 60 * 1000,
  })

  // Set initial enterprise
  useEffect(() => {
    if (!selectedEnterpriseId && enterprisesData && enterprisesData.length > 0) {
      setSelectedEnterpriseId(enterprisesData[0].id)
    }
  }, [enterprisesData, selectedEnterpriseId, setSelectedEnterpriseId])

  // Set initial sheet
  useEffect(() => {
    if (!currentSheetId && sheetsData && sheetsData.length > 0) {
      setCurrentSheetId(sheetsData[0].id)
    }
  }, [sheetsData, currentSheetId])

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
