import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

interface SupplierDialogState {
  type: 'create' | 'update' | 'delete' | null;
  open: boolean;
  supplier?: {
    id: number;
    name: string;
    status: number;
    remark?: string;
  };
}

interface SheetDialogState {
  type: 'create' | 'update' | 'delete' | 'view' | null;
  open: boolean;
  supplierId?: number;
  sheet?: {
    id: number;
    supplier_id: number;
    name: string;
    status: number;
    channel_id: number;
    start_time: number;
    end_time: number;
  };
}

interface SheetBindDialogState {
  open: boolean;
  sheet?: {
    id: number;
    supplier_id: number;
    name: string;
    status: number;
    channel_id: number;
    start_time: number;
    end_time: number;
  };
}

interface ItemDialogState {
  type: 'create' | 'update' | 'delete' | 'view' | null;
  open: boolean;
  supplierId?: number;
  sheetId?: number;
  item?: {
    id: number;
    pricing_sheet_id: number;
    vendor_type: string;
    models: string[];
    discount_type: string;
    discount_value: number;
    remark?: string;
  };
}

interface SupplierPricingContextValue {
  // Supplier dialog
  supplierDialog: SupplierDialogState;
  openSupplierDialog: (state: Omit<SupplierDialogState, 'open'>) => void;
  closeSupplierDialog: () => void;
  // Sheet dialog
  sheetDialog: SheetDialogState;
  openSheetDialog: (state: Omit<SheetDialogState, 'open'>) => void;
  closeSheetDialog: () => void;
  // Sheet bind dialog
  sheetBindDialog: SheetBindDialogState;
  openSheetBindDialog: (sheet: SheetBindDialogState['sheet']) => void;
  closeSheetBindDialog: () => void;
  // Item dialog
  itemDialog: ItemDialogState;
  openItemDialog: (state: Omit<ItemDialogState, 'open'>) => void;
  closeItemDialog: () => void;
  // Selected state
  selectedSupplierId: number | null;
  setSelectedSupplierId: (id: number | null) => void;
  selectedSheetId: number | null;
  setSelectedSheetId: (id: number | null) => void;
  // Refresh triggers
  supplierRefreshTrigger: number;
  triggerSupplierRefresh: () => void;
  sheetRefreshTrigger: number;
  triggerSheetRefresh: () => void;
  itemRefreshTrigger: number;
  triggerItemRefresh: () => void;
}

const SupplierPricingContext = createContext<SupplierPricingContextValue | null>(null);

export function SupplierPricingProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const [selectedSupplierId, setSelectedSupplierId] = useState<number | null>(null);
  const [selectedSheetId, setSelectedSheetId] = useState<number | null>(null);
  const [supplierRefreshTrigger, setSupplierRefreshTrigger] = useState(0);
  const [sheetRefreshTrigger, setSheetRefreshTrigger] = useState(0);
  const [itemRefreshTrigger, setItemRefreshTrigger] = useState(0);
  const [supplierDialog, setSupplierDialog] = useState<SupplierDialogState>({
    type: null,
    open: false,
  });
  const [sheetDialog, setSheetDialog] = useState<SheetDialogState>({
    type: null,
    open: false,
  });
  const [sheetBindDialog, setSheetBindDialog] = useState<SheetBindDialogState>({
    open: false,
  });
  const [itemDialog, setItemDialog] = useState<ItemDialogState>({
    type: null,
    open: false,
  });

  const openSupplierDialog = useCallback((state: Omit<SupplierDialogState, 'open'>) => {
    setSupplierDialog({ ...state, open: true });
  }, []);

  const closeSupplierDialog = useCallback(() => {
    setSupplierDialog({ type: null, open: false });
  }, []);

  const openSheetDialog = useCallback((state: Omit<SheetDialogState, 'open'>) => {
    setSheetDialog({ ...state, open: true });
  }, []);

  const closeSheetDialog = useCallback(() => {
    setSheetDialog({ type: null, open: false });
  }, []);

  const openSheetBindDialog = useCallback((sheet: SheetBindDialogState['sheet']) => {
    setSheetBindDialog({ open: true, sheet });
  }, []);

  const closeSheetBindDialog = useCallback(() => {
    setSheetBindDialog({ open: false });
  }, []);

  const openItemDialog = useCallback((state: Omit<ItemDialogState, 'open'>) => {
    setItemDialog({ ...state, open: true });
  }, []);

  const closeItemDialog = useCallback(() => {
    setItemDialog({ type: null, open: false });
  }, []);

  const triggerSupplierRefresh = useCallback(() => {
    setSupplierRefreshTrigger((n) => n + 1);
  }, []);

  const triggerSheetRefresh = useCallback(() => {
    setSheetRefreshTrigger((n) => n + 1);
  }, []);

  const triggerItemRefresh = useCallback(() => {
    setItemRefreshTrigger((n) => n + 1);
  }, []);

  return (
    <SupplierPricingContext.Provider
      value={{
        supplierDialog,
        openSupplierDialog,
        closeSupplierDialog,
        sheetDialog,
        openSheetDialog,
        closeSheetDialog,
        sheetBindDialog,
        openSheetBindDialog,
        closeSheetBindDialog,
        itemDialog,
        openItemDialog,
        closeItemDialog,
        selectedSupplierId,
        setSelectedSupplierId,
        selectedSheetId,
        setSelectedSheetId,
        supplierRefreshTrigger,
        triggerSupplierRefresh,
        sheetRefreshTrigger,
        triggerSheetRefresh,
        itemRefreshTrigger,
        triggerItemRefresh,
      }}
    >
      {children}
    </SupplierPricingContext.Provider>
  );
}

export function useSupplierPricing(): SupplierPricingContextValue {
  const ctx = useContext(SupplierPricingContext);
  if (!ctx) {
    throw new Error('useSupplierPricing must be used within SupplierPricingProvider');
  }
  return ctx;
}
