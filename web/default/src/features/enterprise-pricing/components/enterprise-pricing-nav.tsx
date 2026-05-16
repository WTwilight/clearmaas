import { useTranslation } from 'react-i18next'
import { Briefcase, FileText, ListChecks } from 'lucide-react'
import { Link, useLocation } from '@tanstack/react-router'
import { cn } from '@/lib/utils'

const TABS = [
  { key: 'enterprise', labelKey: 'Enterprise Management', icon: Briefcase, to: '/enterprise-pricing/enterprise' },
  { key: 'sheets', labelKey: 'Pricing Sheet Management', icon: FileText, to: '/enterprise-pricing/sheets' },
  { key: 'items', labelKey: 'Pricing Items', icon: ListChecks, to: '/enterprise-pricing/items' },
] as const

export function EnterprisePricingNav() {
  const { t } = useTranslation()
  const location = useLocation()

  return (
    <div className='flex items-center gap-1 overflow-x-auto'>
      {TABS.map((tab) => {
        const isActive = location.pathname.startsWith(tab.to)
        return (
          <Link
            key={tab.key}
            to={tab.to}
            className={cn(
              'text-muted-foreground hover:text-foreground flex items-center gap-1.5 whitespace-nowrap border-b-2 border-transparent px-2 py-2 text-sm transition-colors',
              isActive && 'text-foreground border-primary'
            )}
          >
            <tab.icon size={14} />
            {t(tab.labelKey)}
          </Link>
        )
      })}
    </div>
  )
}
