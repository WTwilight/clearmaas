export const enterpriseQueryKeys = {
  all: ['enterprise'] as const,
  lists: () => [...enterpriseQueryKeys.all, 'list'] as const,
  list: (params?: { p?: number; page_size?: number }) => [
    ...enterpriseQueryKeys.lists(),
    params,
  ],
  details: () => [...enterpriseQueryKeys.all, 'detail'] as const,
  detail: (id: number) => [...enterpriseQueryKeys.details(), id],
}

export const sheetsQueryKeys = {
  all: ['sheets'] as const,
  lists: () => [...sheetsQueryKeys.all, 'list'] as const,
  list: (enterpriseId: number, params?: { p?: number; page_size?: number }) => [
    ...sheetsQueryKeys.lists(),
    enterpriseId,
    params,
  ],
  details: () => [...sheetsQueryKeys.all, 'detail'] as const,
  detail: (enterpriseId: number, sheetId: number) => [
    ...sheetsQueryKeys.details(),
    enterpriseId,
    sheetId,
  ],
}

export const bindingQueryKeys = {
  all: ['enterprise-users'] as const,
  list: (enterpriseId: number) => [...bindingQueryKeys.all, enterpriseId],
}

export const pricingItemQueryKeys = {
  all: ['pricing-items'] as const,
  list: (sheetId: number) => [...pricingItemQueryKeys.all, sheetId],
}
