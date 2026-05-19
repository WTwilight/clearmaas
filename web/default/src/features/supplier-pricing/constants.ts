import { t } from 'i18next';
import type { VendorType } from '@/features/enterprise-pricing/constants';
import {
  VENDOR_TYPES,
  getVendorTypeOptions,
  getModelsByVendor,
  getVendorLabel,
  DISCOUNT_TYPE,
} from '@/features/enterprise-pricing/constants';

export { VENDOR_TYPES, getVendorTypeOptions, getModelsByVendor, getVendorLabel, DISCOUNT_TYPE };
export type { VendorType };

export function resolveVendorType(modelName: string): VendorType | null {
  const model = modelName.toLowerCase();
  const hasAny = (keywords: string[]) => keywords.some((kw) => model.includes(kw));

  if (hasAny(VENDOR_TYPES.openai.patterns)) return 'openai';
  if (hasAny(VENDOR_TYPES.anthropic.patterns)) return 'anthropic';
  if (hasAny(VENDOR_TYPES.google.patterns)) return 'google';
  if (hasAny(VENDOR_TYPES.xai.patterns)) return 'xai';
  if (hasAny(VENDOR_TYPES.deepseek.patterns)) return 'deepseek';
  if (hasAny(VENDOR_TYPES.qwen.patterns)) return 'qwen';
  if (hasAny(VENDOR_TYPES.doubao.patterns)) return 'doubao';
  if (hasAny(VENDOR_TYPES.moonshot.patterns)) return 'moonshot';
  if (hasAny(VENDOR_TYPES.mistral.patterns)) return 'mistral';
  if (hasAny(VENDOR_TYPES.meta.patterns)) return 'meta';
  if (hasAny(VENDOR_TYPES.cohere.patterns)) return 'cohere';
  return null;
}

// ---------------------------------------------------------------------------
// Supplier-specific constants
// ---------------------------------------------------------------------------

export const SUPPLIER_STATUS_OPTIONS = [
  { value: 1, label: t('Enabled', { ns: 'common' }) },
  { value: 0, label: t('Disabled', { ns: 'common' }) },
];

export const DISCOUNT_TYPE_OPTIONS = [
  { value: 'ratio', label: t('Ratio', { ns: 'common' }) },
  { value: 'fixed_price', label: t('Fixed Price', { ns: 'common' }) },
  { value: 'per_call', label: t('Per Call', { ns: 'common' }) },
];

export const DISCOUNT_TYPE_LABELS: Record<string, string> = {
  [DISCOUNT_TYPE.RATIO]: t('Ratio', { ns: 'common' }),
  [DISCOUNT_TYPE.FIXED_PRICE]: t('Fixed Price', { ns: 'common' }),
  [DISCOUNT_TYPE.PER_CALL]: t('Per Call', { ns: 'common' }),
};

export const SHEET_STATUS_OPTIONS = [
  { value: 1, label: t('Active', { ns: 'common' }) },
  { value: 0, label: t('Inactive', { ns: 'common' }) },
];

export const SHEET_STATUS = {
  ACTIVE: 1,
  INACTIVE: 0,
} as const;

export const SHEET_STATUS_LABELS: Record<number, string> = {
  [SHEET_STATUS.ACTIVE]: t('Active', { ns: 'common' }),
  [SHEET_STATUS.INACTIVE]: t('Inactive', { ns: 'common' }),
};

export const SUPPLIER_STATUS = {
  ENABLED: 1,
  DISABLED: 0,
} as const;

export function getSupplierStatusOptions(t: (key: string, opts?: object) => string) {
  return [
    { value: String(SUPPLIER_STATUS.ENABLED), label: t('Enabled', { ns: 'common' }) },
    { value: String(SUPPLIER_STATUS.DISABLED), label: t('Disabled', { ns: 'common' }) },
  ];
}

export function getSheetStatusOptions(t: (key: string, opts?: object) => string) {
  return [
    { value: String(SHEET_STATUS.ACTIVE), label: t('Active', { ns: 'common' }) },
    { value: String(SHEET_STATUS.INACTIVE), label: t('Inactive', { ns: 'common' }) },
  ];
}

// ============================================================================
// Error Messages
// ============================================================================

export const ERROR_MESSAGES = {
  CREATE_FAILED: 'Failed to create',
  UPDATE_FAILED: 'Failed to update',
  DELETE_FAILED: 'Failed to delete',
  UNEXPECTED: 'An unexpected error occurred',
} as const;

// ============================================================================
// Success Messages
// ============================================================================

export const SUCCESS_MESSAGES = {
  CREATED: 'Created successfully',
  UPDATED: 'Updated successfully',
  DELETED: 'Deleted successfully',
} as const;
