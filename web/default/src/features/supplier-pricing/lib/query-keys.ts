export const supplierQueryKeys = {
  all: ['supplier'] as const,
  lists: () => [...supplierQueryKeys.all, 'list'] as const,
  list: (params?: Record<string, unknown>) =>
    [...supplierQueryKeys.lists(), params] as const,
  details: () => [...supplierQueryKeys.all, 'detail'] as const,
  detail: (id: number) => [...supplierQueryKeys.details(), id] as const,
};

export const supplierSheetQueryKeys = {
  all: ['supplier-sheet'] as const,
  lists: () => [...supplierSheetQueryKeys.all, 'list'] as const,
  list: (params?: Record<string, unknown>) =>
    [...supplierSheetQueryKeys.lists(), params] as const,
  bySupplier: (supplierId: number) =>
    [...supplierSheetQueryKeys.all, 'by-supplier', supplierId] as const,
  details: () => [...supplierSheetQueryKeys.all, 'detail'] as const,
  detail: (supplierId: number, sheetId: number) =>
    [...supplierSheetQueryKeys.details(), supplierId, sheetId] as const,
};

export const supplierItemQueryKeys = {
  all: ['supplier-item'] as const,
  bySheet: (supplierId: number, sheetId: number) =>
    [...supplierItemQueryKeys.all, 'by-sheet', supplierId, sheetId] as const,
};
