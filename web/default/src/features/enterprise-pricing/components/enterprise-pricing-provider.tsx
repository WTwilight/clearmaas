import React, { useState } from 'react'
import useDialogState from '@/hooks/use-dialog'
import { type Enterprise, type PricingSheet } from '../types'

type EnterpriseDialogType = 'create' | 'update' | 'delete'
type SheetDialogType = 'create' | 'update' | 'delete'
type ItemDialogType = 'create' | 'update' | 'delete'
type BindingDialogType = 'bind' | 'unbind'
type ChannelBindingDialogType = 'bindChannels'

type EnterprisePricingContextType = {
  // Enterprise dialog state
  enterpriseOpen: EnterpriseDialogType | null
  setEnterpriseOpen: (str: EnterpriseDialogType | null) => void
  currentEnterprise: Enterprise | null
  setCurrentEnterprise: React.Dispatch<React.SetStateAction<Enterprise | null>>
  // Sheet dialog state
  sheetOpen: SheetDialogType | null
  setSheetOpen: (str: SheetDialogType | null) => void
  currentSheet: PricingSheet | null
  setCurrentSheet: React.Dispatch<React.SetStateAction<PricingSheet | null>>
  // Pricing item dialog state
  itemOpen: ItemDialogType | null
  setItemOpen: (str: ItemDialogType | null) => void
  currentItemId: number | null
  setCurrentItemId: React.Dispatch<React.SetStateAction<number | null>>
  // Binding dialog state
  bindingOpen: BindingDialogType | null
  setBindingOpen: (str: BindingDialogType | null) => void
  currentBindingId: number | null
  setCurrentBindingId: React.Dispatch<React.SetStateAction<number | null>>
  // Channel binding dialog state
  channelBindingOpen: ChannelBindingDialogType | null
  setChannelBindingOpen: (str: ChannelBindingDialogType | null) => void
  // Shared enterprise context for sheets/items pages
  selectedEnterpriseId: number | null
  setSelectedEnterpriseId: React.Dispatch<React.SetStateAction<number | null>>
  // Refresh trigger
  refreshTrigger: number
  triggerRefresh: () => void
  sheetRefreshTrigger: number
  triggerSheetRefresh: () => void
  itemRefreshTrigger: number
  triggerItemRefresh: () => void
  bindingRefreshTrigger: number
  triggerBindingRefresh: () => void
}

const EnterprisePricingContext =
  React.createContext<EnterprisePricingContextType | null>(null)

export function EnterprisePricingProvider({
  children,
}: {
  children: React.ReactNode
}) {
  // Enterprise dialogs
  const [enterpriseOpen, setEnterpriseOpen] =
    useDialogState<EnterpriseDialogType>(null)
  const [currentEnterprise, setCurrentEnterprise] = useState<Enterprise | null>(
    null
  )
  // Sheet dialogs
  const [sheetOpen, setSheetOpen] = useDialogState<SheetDialogType>(null)
  const [currentSheet, setCurrentSheet] = useState<PricingSheet | null>(null)
  // Pricing item dialogs
  const [itemOpen, setItemOpen] = useDialogState<ItemDialogType>(null)
  const [currentItemId, setCurrentItemId] = useState<number | null>(null)
  // Binding dialogs
  const [bindingOpen, setBindingOpen] = useDialogState<BindingDialogType>(null)
  const [currentBindingId, setCurrentBindingId] = useState<number | null>(null)
  // Channel binding dialogs
  const [channelBindingOpen, setChannelBindingOpen] =
    useDialogState<ChannelBindingDialogType>(null)
  // Shared context
  const [selectedEnterpriseId, setSelectedEnterpriseId] = useState<number | null>(
    null
  )
  // Refresh triggers
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [sheetRefreshTrigger, setSheetRefreshTrigger] = useState(0)
  const [itemRefreshTrigger, setItemRefreshTrigger] = useState(0)
  const [bindingRefreshTrigger, setBindingRefreshTrigger] = useState(0)

  return (
    <EnterprisePricingContext.Provider
      value={{
        enterpriseOpen,
        setEnterpriseOpen,
        currentEnterprise,
        setCurrentEnterprise,
        sheetOpen,
        setSheetOpen,
        currentSheet,
        setCurrentSheet,
        itemOpen,
        setItemOpen,
        currentItemId,
        setCurrentItemId,
        bindingOpen,
        setBindingOpen,
        currentBindingId,
        setCurrentBindingId,
        channelBindingOpen,
        setChannelBindingOpen,
        selectedEnterpriseId,
        setSelectedEnterpriseId,
        refreshTrigger,
        triggerRefresh: () => setRefreshTrigger((p) => p + 1),
        sheetRefreshTrigger,
        triggerSheetRefresh: () => setSheetRefreshTrigger((p) => p + 1),
        itemRefreshTrigger,
        triggerItemRefresh: () => setItemRefreshTrigger((p) => p + 1),
        bindingRefreshTrigger,
        triggerBindingRefresh: () => setBindingRefreshTrigger((p) => p + 1),
      }}
    >
      {children}
    </EnterprisePricingContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useEnterprisePricing = () => {
  const ctx = React.useContext(EnterprisePricingContext)
  if (!ctx) {
    throw new Error(
      'useEnterprisePricing must be used within EnterprisePricingProvider'
    )
  }
  return ctx
}
