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
import { getEnterpriseUsers, getEnterprises } from '../api'
import { useBindingColumns } from './binding-columns'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyTitle, EmptyDescription } from '@/components/ui/empty'

export function BindingTable() {
  const { t } = useTranslation()
  const columns = useBindingColumns()
  const { bindingRefreshTrigger, selectedEnterpriseId, setSelectedEnterpriseId } = useEnterprisePricing()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const [rowSelection, setRowSelection] = useState({})
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: isMobile ? 10 : 20 })

  // Fetch enterprises for selector
  const { data: enterprisesData, isLoading: isLoadingEnterprises } = useQuery({
    queryKey: ['enterprises', 'binding-selector'],
    queryFn: async () => {
      const result = await getEnterprises({ p: 1, page_size: 1000 })
      return result.data?.items || []
    },
    staleTime: 5 * 60 * 1000,
  })

  // Set initial selected enterprise
  useEffect(() => {
    if (!selectedEnterpriseId && enterprisesData && enterprisesData.length > 0) {
      setSelectedEnterpriseId(enterprisesData[0].id)
    }
  }, [enterprisesData, selectedEnterpriseId, setSelectedEnterpriseId])

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'enterprise-users',
      selectedEnterpriseId,
      bindingRefreshTrigger,
    ],
    queryFn: async () => {
      if (!selectedEnterpriseId) return []

      const result = await getEnterpriseUsers(selectedEnterpriseId)

      if (!result.success) {
        toast.error(result.message || t('Failed to load bindings'))
        return []
      }

      return result.data || []
    },
    enabled: !!selectedEnterpriseId,
    placeholderData: (previousData) => previousData,
  })

  const bindings = data || []

  const table = useReactTable({
    data: bindings,
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
    pageCount: Math.ceil(bindings.length / pagination.pageSize),
  })

  if (isLoadingEnterprises) {
    return <Skeleton className='h-10 w-[200px]' />
  }

  if (enterprisesData && enterprisesData.length === 0) {
    return (
      <Empty>
        <EmptyTitle>{t('No Enterprises')}</EmptyTitle>
        <EmptyDescription>{t('Please create an enterprise first')}</EmptyDescription>
      </Empty>
    )
  }

  const selectedEnterprise = enterprisesData?.find((e) => e.id === selectedEnterpriseId)

  return (
    <div className='space-y-2.5'>
      <div className='flex items-center gap-2'>
        <Label>{t('Enterprise')}:</Label>
        <Select
          value={String(selectedEnterpriseId ?? '')}
          onValueChange={(v) => setSelectedEnterpriseId(parseInt(v ?? '0'))}
        >
          <SelectTrigger className='w-[200px]'>
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

      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No Bindings Found')}
        emptyDescription={t(
          'No users bound to this enterprise yet.'
        )}
        skeletonKeyPrefix='binding-skeleton'
      />
    </div>
  )
}
