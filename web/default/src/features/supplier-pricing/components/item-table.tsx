import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
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
import { DataTablePage } from '@/components/data-table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Empty, EmptyTitle, EmptyDescription } from '@/components/ui/empty';
import { getSuppliers, getSupplierPricingSheetsBySupplier, getSupplierPricingItems } from '../api';
import { supplierItemQueryKeys } from '../lib/query-keys';
import { useItemColumns } from './item-columns';
import { useSupplierPricing } from './supplier-pricing-provider';
import { useState } from 'react';

type ItemTableProps = {
  onSheetIdChange?: (sheetId: number | null) => void;
};

export function ItemTable({ onSheetIdChange }: ItemTableProps) {
  const { t } = useTranslation();
  const columns = useItemColumns();
  const {
    itemRefreshTrigger,
    selectedSupplierId,
    setSelectedSupplierId,
    selectedSheetId,
    setSelectedSheetId,
  } = useSupplierPricing();
  const isMobile = useMediaQuery('(max-width: 640px)');
  const [rowSelection, setRowSelection] = useState({});
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: isMobile ? 10 : 20 });

  // Fetch all suppliers
  const { data: suppliersData, isLoading: isLoadingSuppliers } = useQuery({
    queryKey: ['suppliers', 'item-selector'],
    queryFn: async () => {
      const result = await getSuppliers({ p: 1, page_size: 1000 });
      return result.data?.items ?? [];
    },
    staleTime: 5 * 60 * 1000,
  });

  // Fetch sheets when supplier changes
  const { data: sheetsData, isLoading: isLoadingSheets } = useQuery({
    queryKey: ['supplier-sheets', 'selector', selectedSupplierId],
    queryFn: async () => {
      if (!selectedSupplierId) return [];
      const result = await getSupplierPricingSheetsBySupplier(selectedSupplierId);
      return result.items ?? [];
    },
    enabled: !!selectedSupplierId,
    staleTime: 5 * 60 * 1000,
  });

  // Set initial supplier
  useEffect(() => {
    if (!selectedSupplierId && suppliersData && suppliersData.length > 0) {
      setSelectedSupplierId(suppliersData[0].id);
    }
  }, [suppliersData, selectedSupplierId, setSelectedSupplierId]);

  // Set initial sheet when sheets load
  useEffect(() => {
    if (!selectedSheetId && sheetsData && sheetsData.length > 0) {
      setSelectedSheetId(sheetsData[0].id);
    }
  }, [sheetsData, selectedSheetId, setSelectedSheetId]);

  // Notify parent of sheet changes
  useEffect(() => {
    onSheetIdChange?.(selectedSheetId);
  }, [selectedSheetId, onSheetIdChange]);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      ...supplierItemQueryKeys.bySheet(selectedSupplierId ?? 0, selectedSheetId ?? 0),
      pagination.pageIndex + 1,
      pagination.pageSize,
      itemRefreshTrigger,
    ],
    queryFn: async () => {
      if (!selectedSupplierId || !selectedSheetId) return [];
      const result = await getSupplierPricingItems(selectedSupplierId, selectedSheetId);
      if (!result.success) {
        toast.error(result.message || t('Failed to load pricing items'));
        return [];
      }
      return result.data ?? [];
    },
    enabled: !!selectedSupplierId && !!selectedSheetId,
    placeholderData: (previousData) => previousData,
  });

  const items = data ?? [];

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
    globalFilterFn: (row, _columnId, filterValue) => {
      const searchValue = String(filterValue).toLowerCase();
      const fields = [row.original.models?.join(' '), row.original.remark];
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
    pageCount: Math.ceil(items.length / pagination.pageSize),
  });

  if (isLoadingSuppliers || isLoadingSheets) {
    return <Skeleton className='h-10 w-[300px]' />;
  }

  if (suppliersData && suppliersData.length === 0) {
    return (
      <Empty>
        <EmptyTitle>{t('No Suppliers')}</EmptyTitle>
        <EmptyDescription>{t('Please create a supplier and pricing sheet first')}</EmptyDescription>
      </Empty>
    );
  }

  const selectedSupplier = suppliersData?.find((s: { id: number }) => s.id === selectedSupplierId);
  const selectedSheet = sheetsData?.find((s: { id: number }) => s.id === selectedSheetId);

  return (
    <div className='space-y-2.5'>
      <div className='flex flex-wrap items-center gap-3'>
        <div className='flex items-center gap-2'>
          <Label>{t('Supplier')}:</Label>
          <Select
            value={String(selectedSupplierId ?? '')}
            onValueChange={(v) => {
              setSelectedSupplierId(parseInt(v));
              setSelectedSheetId(null);
            }}
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
            value={String(selectedSheetId ?? '')}
            onValueChange={(v) => setSelectedSheetId(parseInt(v))}
            disabled={!selectedSupplierId || !sheetsData?.length}
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

      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No Pricing Items Found')}
        emptyDescription={t('No pricing items found for this sheet.')}
        skeletonKeyPrefix='item-skeleton'
      />
    </div>
  );
}
