import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { getRouteApi } from '@tanstack/react-router';
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
} from '@tanstack/react-table';
import { useMediaQuery } from '@/hooks';
import { toast } from 'sonner';
import { useTableUrlState } from '@/hooks/use-table-url-state';
import { DataTablePage } from '@/components/data-table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { getSuppliers, getSupplierPricingSheets } from '../api';
import { supplierSheetQueryKeys } from '../lib/query-keys';
import { useSheetColumns } from './sheet-columns';
import { useSupplierPricing } from './supplier-pricing-provider';

const route = getRouteApi('/_authenticated/supplier-pricing/sheets/');

const ALL_SUPPLIERS_VALUE = 'all';

export function SheetTable() {
  const { t } = useTranslation();
  const columns = useSheetColumns();
  const { sheetRefreshTrigger } = useSupplierPricing();
  const isMobile = useMediaQuery('(max-width: 640px)');
  const [rowSelection, setRowSelection] = useState({});
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [filterSupplierId, setFilterSupplierId] = useState<string>(ALL_SUPPLIERS_VALUE);

  const routeSearch = route.useSearch();
  useEffect(() => {
    if (routeSearch.supplierId) {
      setFilterSupplierId(String(routeSearch.supplierId));
    }
  }, [routeSearch.supplierId]);

  const {
    globalFilter,
    onGlobalFilterChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
    globalFilter: { enabled: true, key: 'filter' },
  });

  const { data: suppliersData, isLoading: isLoadingSuppliers } = useQuery({
    queryKey: ['suppliers', 'selector'],
    queryFn: async () => {
      const result = await getSuppliers({ p: 1, page_size: 1000 });
      return result.data?.items ?? [];
    },
    staleTime: 5 * 60 * 1000,
  });

  const effectiveSupplierId =
    filterSupplierId === ALL_SUPPLIERS_VALUE
      ? undefined
      : parseInt(filterSupplierId);

  const showSupplierColumn =
    filterSupplierId === ALL_SUPPLIERS_VALUE;

  const columnsWithCondition = useSheetColumns(showSupplierColumn);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      ...supplierSheetQueryKeys.lists(),
      effectiveSupplierId,
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      sheetRefreshTrigger,
    ],
    queryFn: async () => {
      const result = await getSupplierPricingSheets({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      });
      if (!result.success) {
        toast.error(result.message || t('Failed to load pricing sheets'));
        return { items: [], total: 0 };
      }
      return {
        items: result.data?.items ?? [],
        total: result.data?.total ?? 0,
      };
    },
    placeholderData: (previousData) => previousData,
  });

  const table = useReactTable({
    data: data?.items ?? [],
    columns: columnsWithCondition,
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
      const searchValue = String(filterValue).toLowerCase();
      const fields = [row.original.name, row.original.supplier_name];
      return fields.some((field) =>
        String(field || '').toLowerCase().includes(searchValue)
      );
    },
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    onPaginationChange,
    onGlobalFilterChange,
    pageCount: Math.ceil((data?.total ?? 0) / pagination.pageSize),
  });

  const pageCount = table.getPageCount();
  useEffect(() => {
    ensurePageInRange(pageCount);
  }, [pageCount, ensurePageInRange]);

  const handleSupplierFilterChange = (value: string | null) => {
    const v = value ?? ALL_SUPPLIERS_VALUE;
    setFilterSupplierId(v);
    onPaginationChange({ pageIndex: 0, pageSize: pagination.pageSize });
  };

  const selectedSupplierName =
    filterSupplierId === ALL_SUPPLIERS_VALUE
      ? t('All Suppliers')
      : suppliersData?.find((s) => String(s.id) === filterSupplierId)?.name ?? '';

  if (isLoadingSuppliers) {
    return <Skeleton className='h-10 w-[200px]' />;
  }

  const toolbar = (
    <div className='flex flex-wrap items-center gap-2'>
      <Select
        value={filterSupplierId}
        onValueChange={handleSupplierFilterChange}
      >
        <SelectTrigger className='w-[180px]'>
          <SelectValue>{selectedSupplierName}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL_SUPPLIERS_VALUE}>
            {t('All Suppliers')}
          </SelectItem>
          {suppliersData?.map((s) => (
            <SelectItem key={s.id} value={String(s.id)}>
              {s.name}
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
  );

  return (
    <DataTablePage
      table={table}
      columns={columnsWithCondition}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Pricing Sheets Found')}
      emptyDescription={t('No pricing sheets found. Try adjusting your search.')}
      skeletonKeyPrefix='sheet-skeleton'
      toolbar={toolbar}
    />
  );
}
