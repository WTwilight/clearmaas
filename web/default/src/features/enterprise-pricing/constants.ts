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
  PER_CALL: 'per_call',
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
  [DISCOUNT_TYPE.PER_CALL]: {
    labelKey: 'Per Call',
    value: DISCOUNT_TYPE.PER_CALL,
  },
} as const

export const getDiscountTypeOptions = (t: (key: string) => string) => [
  { label: t('Ratio Discount'), value: DISCOUNT_TYPE.RATIO },
  { label: t('Fixed Price'), value: DISCOUNT_TYPE.FIXED_PRICE },
  { label: t('Per Call'), value: DISCOUNT_TYPE.PER_CALL },
]

// Mirrors the pattern-matching rules from resolveModelProvider in model-badge.tsx
export const VENDOR_TYPES = {
  openai: {
    labelKey: 'OpenAI',
    patterns: ['gpt-', 'chatgpt-', 'dall-', 'whisper-', 'tts-', 'o1', 'o3', 'o4'],
  },
  anthropic: { labelKey: 'Anthropic', patterns: ['claude-', 'anthropic-'] },
  google: { labelKey: 'Google', patterns: ['gemini-', 'learnlm-'] },
  xai: { labelKey: 'xAI', patterns: ['grok-', 'xai-'] },
  deepseek: { labelKey: 'DeepSeek', patterns: ['deepseek-'] },
  qwen: { labelKey: 'Qwen', patterns: ['qwen', 'qwq-'] },
  doubao: { labelKey: 'Doubao', patterns: ['doubao-', 'volcengine'] },
  moonshot: { labelKey: 'Moonshot', patterns: ['moonshot-', 'kimi-'] },
  mistral: { labelKey: 'Mistral', patterns: ['mistral-', 'mixtral-'] },
  meta: { labelKey: 'Meta', patterns: ['llama-', 'meta-'] },
  cohere: { labelKey: 'Cohere', patterns: ['command-', 'cohere-'] },
} as const

export type VendorType = keyof typeof VENDOR_TYPES

const hasAny = (model: string, keywords: string[]) =>
  keywords.some((kw) => model.includes(kw))

function doResolveVendorType(modelName: string): VendorType | null {
  const model = modelName.toLowerCase()
  if (hasAny(model, VENDOR_TYPES.openai.patterns)) return 'openai'
  if (hasAny(model, VENDOR_TYPES.anthropic.patterns)) return 'anthropic'
  if (hasAny(model, VENDOR_TYPES.google.patterns)) return 'google'
  if (hasAny(model, VENDOR_TYPES.xai.patterns)) return 'xai'
  if (hasAny(model, VENDOR_TYPES.deepseek.patterns)) return 'deepseek'
  if (hasAny(model, VENDOR_TYPES.qwen.patterns)) return 'qwen'
  if (hasAny(model, VENDOR_TYPES.doubao.patterns)) return 'doubao'
  if (hasAny(model, VENDOR_TYPES.moonshot.patterns)) return 'moonshot'
  if (hasAny(model, VENDOR_TYPES.mistral.patterns)) return 'mistral'
  if (hasAny(model, VENDOR_TYPES.meta.patterns)) return 'meta'
  if (hasAny(model, VENDOR_TYPES.cohere.patterns)) return 'cohere'
  return null
}

export function resolveVendorType(modelName: string): VendorType | null {
  return doResolveVendorType(modelName)
}

export function getVendorTypeOptions(t: (key: string) => string) {
  return (Object.keys(VENDOR_TYPES) as VendorType[]).map((key) => ({
    value: key,
    label: t(VENDOR_TYPES[key].labelKey),
  }))
}

export function getModelsByVendor(vendor: VendorType, allModels: string[]): string[] {
  const patterns = VENDOR_TYPES[vendor].patterns
  return allModels.filter((model) => hasAny(model.toLowerCase(), patterns))
}

export function getVendorLabel(vendor: VendorType | string): string {
  return VENDOR_TYPES[vendor as VendorType]?.labelKey ?? vendor
}

// ============================================================================
// Error Messages
// ============================================================================

export const ERROR_MESSAGES = {
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
