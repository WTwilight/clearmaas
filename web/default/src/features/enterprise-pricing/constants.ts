// ============================================================================
// Enterprise Status
// ============================================================================

export const ENTERPRISE_STATUS = {
  ENABLED: 1,
  DISABLED: 0,
} as const

export const ENTERPRISE_STATUSES = {
  [ENTERPRISE_STATUS.ENABLED]: {
    labelKey: 'Enabled',
    variant: 'success' as const,
    value: ENTERPRISE_STATUS.ENABLED,
    showDot: true,
  },
  [ENTERPRISE_STATUS.DISABLED]: {
    labelKey: 'Disabled',
    variant: 'neutral' as const,
    value: ENTERPRISE_STATUS.DISABLED,
    showDot: true,
  },
} as const

export const getEnterpriseStatusOptions = (t: (key: string) => string) => [
  { label: t('Enabled'), value: String(ENTERPRISE_STATUS.ENABLED) },
  { label: t('Disabled'), value: String(ENTERPRISE_STATUS.DISABLED) },
]

// ============================================================================
// Sheet Status
// ============================================================================

export const SHEET_STATUS = {
  ACTIVE: 1,
  INACTIVE: 0,
} as const

export const SHEET_STATUSES = {
  [SHEET_STATUS.ACTIVE]: {
    labelKey: 'Active',
    variant: 'success' as const,
    value: SHEET_STATUS.ACTIVE,
    showDot: true,
  },
  [SHEET_STATUS.INACTIVE]: {
    labelKey: 'Inactive',
    variant: 'neutral' as const,
    value: SHEET_STATUS.INACTIVE,
    showDot: true,
  },
} as const

export const getSheetStatusOptions = (t: (key: string) => string) => [
  { label: t('Active'), value: String(SHEET_STATUS.ACTIVE) },
  { label: t('Inactive'), value: String(SHEET_STATUS.INACTIVE) },
]

// ============================================================================
// Discount Type
// ============================================================================

export const DISCOUNT_TYPE = {
  RATIO: 'ratio',
  FIXED_PRICE: 'fixed_price',
} as const

export const DISCOUNT_TYPES = {
  [DISCOUNT_TYPE.RATIO]: {
    labelKey: 'Ratio Discount',
    value: DISCOUNT_TYPE.RATIO,
  },
  [DISCOUNT_TYPE.FIXED_PRICE]: {
    labelKey: 'Fixed Price',
    value: DISCOUNT_TYPE.FIXED_PRICE,
  },
} as const

export const getDiscountTypeOptions = (t: (key: string) => string) => [
  { label: t('Ratio Discount'), value: DISCOUNT_TYPE.RATIO },
  { label: t('Fixed Price'), value: DISCOUNT_TYPE.FIXED_PRICE },
]

// ============================================================================
// Error Messages
// ============================================================================

export const ERROR_MESSAGES = {
  UNEXPECTED: 'An unexpected error occurred',
  LOAD_FAILED: 'Failed to load data',
  CREATE_FAILED: 'Failed to create',
  UPDATE_FAILED: 'Failed to update',
  DELETE_FAILED: 'Failed to delete',
  BIND_FAILED: 'Failed to bind user',
  UNBIND_FAILED: 'Failed to unbind user',
} as const

// ============================================================================
// Success Messages
// ============================================================================

export const SUCCESS_MESSAGES = {
  CREATED: 'Created successfully',
  UPDATED: 'Updated successfully',
  DELETED: 'Deleted successfully',
  USER_BOUND: 'User bound successfully',
  USER_UNBOUND: 'User unbound successfully',
} as const
