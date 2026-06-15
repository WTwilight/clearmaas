/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  IconDiscord,
  IconGithub,
  IconLinuxDo,
  IconWeChat,
} from '@/assets/brand-icons'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { useOAuthLogin } from '../hooks/use-oauth-login'
import type { SystemStatus } from '../types'

type OAuthProvidersProps = {
  status: SystemStatus | null
  disabled?: boolean
  className?: string
  beforeLogin?: () => boolean
  onWeChatLogin?: () => void
  isWeChatLoading?: boolean
  variant?: 'default' | 'close'
  allowedProviders?: string[]
  labelOverrides?: Record<string, string>
}

type ProviderButton = {
  key: string
  label: string
  onClick: () => void
  icon?: ReactNode
  disabled?: boolean
}

export function OAuthProviders({
  status,
  disabled = false,
  className,
  beforeLogin,
  onWeChatLogin,
  isWeChatLoading = false,
  variant = 'default',
  allowedProviders,
  labelOverrides,
}: OAuthProvidersProps) {
  const { t } = useTranslation()
  const {
    isLoading,
    githubButtonText,
    githubButtonDisabled,
    handleGitHubLogin,
    handleDiscordLogin,
    handleOIDCLogin,
    handleLinuxDOLogin,
    handleCustomOAuthLogin,
  } = useOAuthLogin(status)
  const statusData = status?.data ?? status

  const providerButtons: ProviderButton[] = []
  const runBeforeLogin = () => beforeLogin?.() ?? true
  const withBeforeLogin = (onClick: () => void) => () => {
    if (!runBeforeLogin()) return
    onClick()
  }

  if (statusData?.wechat_login && onWeChatLogin) {
    providerButtons.push({
      key: 'wechat',
      label: t('Continue with WeChat'),
      onClick: withBeforeLogin(onWeChatLogin),
      icon: <IconWeChat className='h-4 w-4' />,
      disabled: isWeChatLoading,
    })
  }

  if (statusData?.github_oauth && statusData?.github_client_id) {
    providerButtons.push({
      key: 'github',
      label: githubButtonText || t('Continue with GitHub'),
      onClick: withBeforeLogin(handleGitHubLogin),
      icon: <IconGithub className='h-4 w-4' />,
      disabled: githubButtonDisabled,
    })
  }

  if (statusData?.discord_oauth && statusData?.discord_client_id) {
    providerButtons.push({
      key: 'discord',
      label: t('Continue with Discord'),
      onClick: withBeforeLogin(handleDiscordLogin),
      icon: <IconDiscord className='h-4 w-4' />,
    })
  }

  if (
    statusData?.oidc_enabled &&
    statusData?.oidc_client_id &&
    statusData?.oidc_authorization_endpoint
  ) {
    providerButtons.push({
      key: 'oidc',
      label: t('Continue with OIDC'),
      onClick: withBeforeLogin(handleOIDCLogin),
    })
  }

  if (statusData?.linuxdo_oauth && statusData?.linuxdo_client_id) {
    providerButtons.push({
      key: 'linuxdo',
      label: t('Continue with LinuxDO'),
      onClick: withBeforeLogin(handleLinuxDOLogin),
      icon: <IconLinuxDo className='h-4 w-4' />,
    })
  }

  // Custom OAuth providers
  const customProviders = statusData?.custom_oauth_providers
  if (customProviders && customProviders.length > 0) {
    for (const provider of customProviders) {
      if (!provider.client_id || !provider.authorization_endpoint) continue
      providerButtons.push({
        key: `custom-${provider.slug}`,
        label: t('Continue with {{name}}', { name: provider.name }),
        onClick: withBeforeLogin(() => handleCustomOAuthLogin(provider)),
      })
    }
  }

  if (providerButtons.length === 0) return null

  const visibleProviderButtons = allowedProviders
    ? providerButtons.filter((providerButton) =>
        allowedProviders.includes(providerButton.key)
      )
    : providerButtons

  if (visibleProviderButtons.length === 0) return null

  if (variant === 'close') {
    return (
      <div className={cn('grid gap-2', className)}>
        {visibleProviderButtons.map(
          ({ key, label, onClick, icon, disabled: extraDisabled }) => {
            const displayLabel = labelOverrides?.[key] ?? label
            return (
              <button
                key={key}
                type='button'
                disabled={disabled || isLoading || extraDisabled}
                onClick={onClick}
                className='google-register'
              >
                <span className='google-g'>
                  {icon || displayLabel.slice(0, 1)}
                </span>
                <span>{displayLabel}</span>
              </button>
            )
          }
        )}
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', className)}>
      <div className='relative'>
        <div className='absolute inset-0 flex items-center'>
          <span className='w-full border-t' />
        </div>
        <div className='relative flex justify-center text-xs uppercase'>
          <span className='bg-background text-muted-foreground px-2'>
            {t('Or continue with')}
          </span>
        </div>
      </div>

      <div className='flex flex-col gap-2'>
        {visibleProviderButtons.map(
          ({ key, label, onClick, icon, disabled: extraDisabled }) => (
            <Button
              key={key}
              variant='outline'
              type='button'
              disabled={disabled || isLoading || extraDisabled}
              onClick={onClick}
              className='h-11 w-full justify-center gap-2 rounded-lg'
            >
              {icon}
              {labelOverrides?.[key] ?? label}
            </Button>
          )
        )}
      </div>
    </div>
  )
}
