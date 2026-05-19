import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearch } from '@tanstack/react-router'
import { SheetTable } from './sheet-table'
import { SheetPrimaryButtons } from './sheet-primary-buttons'
import { SheetDrawer } from './sheet-drawer'
import { SheetDeleteDialog } from './sheet-delete-dialog'
import { BindUserDialog } from './bind-user-dialog'
import { useEnterprisePricing } from './enterprise-pricing-provider'
import { Skeleton } from '@/components/ui/skeleton'
import { getEnterprises } from '../api'

type SheetSectionProps = {
  enterpriseId?: number
}

export function SheetSection({ enterpriseId: urlEnterpriseId }: SheetSectionProps) {
  const { t } = useTranslation()
  const { sheetOpen, setSheetOpen, currentSheet, setCurrentSheet, setSelectedEnterpriseId } =
    useEnterprisePricing()
  const routeSearch = useSearch({ from: '/_authenticated/enterprise-pricing/sheets/' })
  const [filterEnterpriseId, setFilterEnterpriseId] = useState<string | null>(null)

  // Sync enterpriseId from URL into context (e.g. navigating from enterprise management)
  useEffect(() => {
    const id = urlEnterpriseId ?? routeSearch.enterpriseId
    if (id) {
      setSelectedEnterpriseId(id)
    }
  }, [urlEnterpriseId, routeSearch.enterpriseId, setSelectedEnterpriseId])

  // Fetch enterprises for filter dropdown
  const { data: enterprisesData } = useEnterprisesForFilter()

  // When creating a new sheet (no URL enterpriseId), use the filter-selected enterprise
  const selectedForCreate = urlEnterpriseId ?? routeSearch.enterpriseId
  const effectiveEnterpriseId = selectedForCreate ?? (filterEnterpriseId ? parseInt(filterEnterpriseId) : null)

  const handleCreate = () => {
    setCurrentSheet(null)
    // Set the enterpriseId into context before opening the drawer
    if (effectiveEnterpriseId) {
      setSelectedEnterpriseId(effectiveEnterpriseId)
    }
    setSheetOpen('create')
  }

  const handleSheetTableEnterpriseChange = (value: string | null) => {
    setFilterEnterpriseId(value)
    if (value) {
      setSelectedEnterpriseId(parseInt(value))
    }
  }

  return (
    <>
      <div className='mb-2.5 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{t('Pricing Sheet Management')}</h3>
        <SheetPrimaryButtons onCreate={handleCreate} />
      </div>
      <SheetTable onEnterpriseFilterChange={handleSheetTableEnterpriseChange} />
      <SheetDrawer
        open={sheetOpen === 'create' || sheetOpen === 'update'}
        onOpenChange={(isOpen) => !isOpen && setSheetOpen(null)}
        currentRow={sheetOpen === 'update' ? currentSheet || undefined : undefined}
      />
      <SheetDeleteDialog />
      <BindUserDialog />
    </>
  )
}

function useEnterprisesForFilter() {
  const [data, setData] = useState<Array<{ id: number; name: string }>>([])
  useEffect(() => {
    getEnterprises({ p: 1, page_size: 1000 }).then((result) => {
      if (result.data?.items) {
        setData(result.data.items)
      }
    })
  }, [])
  return { data }
}
