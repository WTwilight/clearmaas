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
import { getSuppliers } from '../api';
import { supplierQueryKeys } from '../lib/query-keys';
import { useSupplierPricing } from './supplier-pricing-provider';
import { useSupplierColumns } from './supplier-columns';

const route = getRouteApi('/_authenticated/supplier-pricing/suppliers/');

export function SupplierTable() {
  const { t } = useTranslation();
  const columns = useSupplierColumns();
  const { supplierRefreshTrigger } = useSupplierPricing();
  const isMobile = useMediaQuery('(max-width: 640px)');
  const [rowSelection, setRowSelection] = useState({});
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

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

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      ...supplierQueryKeys.lists(),
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      supplierRefreshTrigger,
    ],
    queryFn: async () => {
      const result = await getSuppliers({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      });
      if (!result.success) {
        toast.error(result.message || t('Failed to load suppliers'));
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
      const searchValue = String(filterValue).toLowerCase();
      const fields = [row.getValue('name'), row.original.remark];
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

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Suppliers Found')}
      emptyDescription={t('No suppliers found. Try adjusting your search.')}
      skeletonKeyPrefix='supplier-skeleton'
      toolbarProps={{
        searchPlaceholder: t('Filter by name...'),
      }}
    />
  );
}
