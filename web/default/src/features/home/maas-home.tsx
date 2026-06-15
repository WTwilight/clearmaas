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
import { Link } from '@tanstack/react-router'
import '@/styles/maas-page-design.css'
import { ArrowUpRight, Bell, BookOpen, Sun, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useTheme } from '@/context/theme-provider'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const publicLanguages = [
  { code: 'en', shortLabel: 'EN', label: 'English' },
  { code: 'zh', shortLabel: 'ZH', label: '中文' },
  { code: 'fr', shortLabel: 'FR', label: 'Français' },
  { code: 'ru', shortLabel: 'RU', label: 'Русский' },
  { code: 'ja', shortLabel: 'JA', label: '日本語' },
  { code: 'vi', shortLabel: 'VI', label: 'Tiếng Việt' },
]
export const CLEARMAAS_DOCS_URL = 'https://docs.clearmaas.com/introduction'

type MaasHomeProps = {
  isAuthenticated: boolean
}

export function MaasHome(props: MaasHomeProps) {
  const { t } = useTranslation()
  const { status } = useStatus()

  return (
    <div className='maas-page home-page'>
      <MaasPublicHeader
        active='home'
        docsHref={CLEARMAAS_DOCS_URL}
      />

      <main>
        <section className='hero-section'>
          <div className='hero-bg-grid' aria-hidden='true' />
          <div className='hero-content'>
            <div className='hero-copy'>
              <h1>{t('AI Reshape the World. We Aggregate AI Models.')}</h1>
              <p>
                {t(
                  'Power AI applications, manage digital assets, connect the Future. ClearMaas aggregates upstream AI providers behind one unified API, with routing, billing, keys, logs, and service health built in.'
                )}
              </p>
              <p className='hero-slogan'>
                {'—— '}
                {t('One API. Every model. Transparent usage.')}
              </p>
              <div className='hero-metrics' aria-label={t('Platform metrics')}>
                <span>
                  <strong>40+</strong>
                  {t('upstream services')}
                </span>
                <span>
                  <strong>500+</strong>
                  {t('model billing support')}
                </span>
                <span>
                  <strong>OpenAI</strong>
                  {t('compatible API routes')}
                </span>
                <span>
                  <strong>SSE</strong>
                  {t('stream scheduling')}
                </span>
              </div>
              <div className='hero-buttons'>
                <Link
                  className='primary-gradient-btn'
                  to={props.isAuthenticated ? '/dashboard' : '/sign-up'}
                >
                  {props.isAuthenticated ? t('Open console') : t('Get started')}
                </Link>
                <a
                  className='ghost-pill-btn'
                  href={CLEARMAAS_DOCS_URL}
                  rel='noopener noreferrer'
                  target='_blank'
                >
                  <BookOpen size={16} aria-hidden='true' />
                  {t('Help Center')}
                </a>
              </div>

              <div className='app-support'>
                <div className='app-support-head'>
                  <h2>{t('Common app support')}</h2>
                  <span>{t('OpenAI-compatible clients')}</span>
                </div>
                <div className='app-icons' aria-label={t('Common app support')}>
                  {appIcons.map((app) => (
                    <Link
                      key={app.name}
                      className='app-icon'
                      to={app.href}
                      aria-label={app.name}
                    >
                      {getLobeIcon(app.icon, 30)}
                    </Link>
                  ))}
                  <Link
                    className='more-icon'
                    to='/model-square'
                    aria-label={t('More apps')}
                  >
                    <svg viewBox='0 0 24 24' aria-hidden='true'>
                      <circle cx='5' cy='12' r='1.8' />
                      <circle cx='12' cy='12' r='1.8' />
                      <circle cx='19' cy='12' r='1.8' />
                    </svg>
                  </Link>
                </div>
                <p className='guide-link'>
                  <ArrowUpRight size={15} aria-hidden='true' />
                  {t('New to relay APIs? Please read')}{' '}
                  <a
                    href={CLEARMAAS_DOCS_URL}
                    rel='noopener noreferrer'
                    target='_blank'
                  >
                    {t('Getting Started')}
                  </a>
                </p>
              </div>
            </div>

            <aside
              className='operation-card api-preview'
              aria-label={t('API request example')}
            >
              <div className='api-preview-head'>
                <span>200 OK</span>
                <strong>POST</strong>
              </div>
              <h2>/v1/chat/completions</h2>
              <div className='api-tabs'>
                <span className='active'>{t('Chat')}</span>
                <span>{t('Responses')}</span>
                <span>{t('Claude')}</span>
                <span>{t('Gemini')}</span>
              </div>
              <div className='code-sample'>
                <span>{t('REQUEST')}</span>
                <pre>{`curl -X POST "/v1/chat/completions" \\
  -H "Authorization: Bearer sk-****" \\
  -d '{
    "model": "auto",
    "messages": [{ "role": "user", "content": "..." }]
  }'`}</pre>
              </div>
              <div className='api-result-grid'>
                <div>
                  <strong>93</strong>
                  <span>{t('MS')}</span>
                </div>
                <div>
                  <strong>25</strong>
                  <span>{t('TOKENS')}</span>
                </div>
                <div>
                  <strong>$0.00075</strong>
                  <span>{t('COST')}</span>
                </div>
                <div>
                  <strong>SSE</strong>
                  <span>{t('STREAM')}</span>
                </div>
              </div>
            </aside>
          </div>
        </section>

        <FeatureSection />
        <WorkflowSection />

        <section className='home-cta-section'>
          <div>
            <span className='section-kicker'>{t('Ready')}</span>
            <h2>{t('Ready to simplify your AI integration?')}</h2>
            <p>
              {t(
                'Deploy your own gateway and start routing requests through your configured upstream services.'
              )}
            </p>
          </div>
          <div className='hero-buttons'>
            <Link
              className='primary-gradient-btn'
              to={props.isAuthenticated ? '/dashboard' : '/sign-up'}
            >
              {t('Get started')}
            </Link>
            <Link className='ghost-pill-btn' to='/model-square'>
              {t('View Pricing')}
            </Link>
          </div>
        </section>
      </main>
    </div>
  )
}

function FeatureSection() {
  const { t } = useTranslation()
  const features = [
    [
      '01',
      'Lightning Fast',
      'Optimized routing for millisecond response times.',
      'OpenAI, Claude, Gemini, DeepSeek, Qwen, Llama and other upstream services are unified behind one gateway.',
    ],
    [
      '02',
      'Secure & Reliable',
      'Enterprise-grade access control and auditability.',
      'API keys, user groups, quotas, rate limits, logs, and permissions are managed in one place.',
    ],
    [
      '03',
      'Global Coverage',
      'Stable global access through multi-channel routing.',
      'Load balancing, retry strategies, and scheduling keep model calls stable across scenarios.',
    ],
    [
      '04',
      'Developer Friendly',
      'Compatible API routes for common AI workflows.',
      'Use familiar OpenAI-compatible routes for Chat, Responses, Claude, Gemini, and more.',
    ],
    [
      '05',
      'Transparent Billing',
      'Pay-as-you-go with real-time usage monitoring.',
      'Track costs by model, group, ratio, token usage, and request logs.',
    ],
    [
      '06',
      'Team Collaboration',
      'Multi-user management with flexible permissions.',
      'Teams, enterprise customers, price sheets, and access policies can be configured independently.',
    ],
  ] as const

  return (
    <section className='service-section'>
      <div className='section-head'>
        <span className='section-kicker'>{t('Core Features')}</span>
        <h2>{t('Built for developers, designed for scale')}</h2>
        <p>
          {t(
            'Product capabilities are organized into scannable modules: speed, security, global access, protocol compatibility, billing transparency, and team collaboration.'
          )}
        </p>
      </div>
      <div className='service-grid feature-grid'>
        {features.map(([index, title, desc, detail]) => (
          <article key={title} className='service-card'>
            <span className='service-icon'>{index}</span>
            <h3>{t(title)}</h3>
            <p>{t(desc)}</p>
            <small>{t(detail)}</small>
          </article>
        ))}
      </div>
    </section>
  )
}

function WorkflowSection() {
  const { t } = useTranslation()
  const steps = [
    [
      '1',
      'Configure',
      'Add your API keys, set up channels and configure access permissions.',
    ],
    [
      '2',
      'Connect',
      'Connect through OpenAI, Claude, Gemini, and compatible API routes.',
    ],
    [
      '3',
      'Monitor',
      'Track usage, costs and performance with real-time analytics.',
    ],
  ] as const

  return (
    <section className='workflow-section'>
      <div className='section-head'>
        <span className='section-kicker'>{t('How It Works')}</span>
        <h2>{t('Three steps to get started')}</h2>
        <p>
          {t(
            'Keep the three-step onboarding flow and present it with the model-square card rhythm.'
          )}
        </p>
      </div>
      <div className='workflow-grid'>
        {steps.map(([index, title, desc]) => (
          <article key={title}>
            <span>{index}</span>
            <h3>{t(title)}</h3>
            <p>{t(desc)}</p>
          </article>
        ))}
      </div>
    </section>
  )
}

export function MaasPublicHeader(props: {
  active: 'home' | 'model-square'
  docsHref?: string
  brandTitle?: string
}) {
  const { t, i18n } = useTranslation()
  const { theme, setTheme } = useTheme()
  const user = useAuthStore((state) => state.auth.user)
  const { systemName, logo, logoLoaded, loading } = useSystemConfig()
  const activeLanguage =
    publicLanguages.find((language) => i18n.language?.startsWith(language.code))
      ?.code || 'en'
  const currentLanguage =
    publicLanguages.find((language) => language.code === activeLanguage) ||
    publicLanguages[0]
  const userName =
    user?.display_name || user?.username || user?.email || t('Login')
  const avatar = userName.slice(0, 1).toUpperCase()
  const changeLanguage = async (code: string) => {
    await i18n.changeLanguage(code)
    if (user) {
      try {
        await api.put('/api/user/self', { language: code })
      } catch {
        // Keep language switching responsive even if preference persistence fails.
      }
    }
  }

  return (
    <div className='header-wrap'>
      <header className='top-nav'>
        <Link className='brand' to='/'>
          <span className='brand-mark' aria-hidden='true'>
            <img
              src={logo}
              alt=''
              className={logoLoaded && !loading ? 'is-loaded' : undefined}
            />
          </span>
          <span className='brand-copy'>
            <strong>{props.brandTitle || systemName || 'ClearMaas'}</strong>
            <span>{t('Enterprise AI model API proxy platform')}</span>
          </span>
        </Link>
        <nav className='nav-links' aria-label={t('Primary')}>
          <Link
            className={props.active === 'home' ? 'active' : undefined}
            to='/'
          >
            {t('Home')}
          </Link>
          <Link
            className={props.active === 'model-square' ? 'active' : undefined}
            to='/model-square'
          >
            {t('Model Square')}
          </Link>
          <a
            href={props.docsHref || CLEARMAAS_DOCS_URL}
            rel='noopener noreferrer'
            target='_blank'
          >
            {t('Docs')}
          </a>
          <Link to='/dashboard'>{t('Console')}</Link>
        </nav>
        <div className='nav-tools'>
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger
              className='language-toggle'
              aria-label={t('Change language')}
            >
              <span className='active'>{currentLanguage.shortLabel}</span>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align='end'
              sideOffset={24}
              positionerClassName='z-[120]'
              className='maas-language-menu'
            >
              {publicLanguages.map((language) => (
                <DropdownMenuItem
                  key={language.code}
                  className={cn(
                    'maas-language-item',
                    language.code === activeLanguage && 'active'
                  )}
                  onClick={() => changeLanguage(language.code)}
                >
                  <span>{language.label}</span>
                  <small>{language.shortLabel}</small>
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <span className='divider' aria-hidden='true' />
          <button
            className='icon-button'
            type='button'
            aria-label={t('Toggle theme')}
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          >
            <Sun size={15} aria-hidden='true' />
          </button>
          <Link
            className='icon-button notification-button'
            to={user ? '/dashboard' : '/sign-in'}
            aria-label={t('Notifications')}
          >
            <Bell size={15} aria-hidden='true' />
          </Link>
          <Link
            className='user-entry'
            to={user ? '/dashboard' : '/sign-in'}
            aria-label={t('Login or user information')}
          >
            <span className='user-avatar'>
              {user ? avatar : <UserRound size={13} aria-hidden='true' />}
            </span>
            <span>{user ? userName : t('Login')}</span>
          </Link>
        </div>
      </header>
    </div>
  )
}

type HomeAppIcon = {
  name: string
  href: '/model-square'
  icon: string
}

const appIcons = [
  {
    name: 'OpenAI',
    href: '/model-square',
    icon: 'OpenAI',
  },
  {
    name: 'Claude',
    href: '/model-square',
    icon: 'Claude.Color',
  },
  {
    name: 'Gemini',
    href: '/model-square',
    icon: 'Gemini.Color',
  },
  {
    name: 'Cherry Studio',
    href: '/model-square',
    icon: 'CherryStudio.Color',
  },
  {
    name: 'Cursor',
    href: '/model-square',
    icon: 'Cursor',
  },
  {
    name: 'Open WebUI',
    href: '/model-square',
    icon: 'OpenWebUI',
  },
  {
    name: 'Cline',
    href: '/model-square',
    icon: 'Cline',
  },
  {
    name: 'Dify',
    href: '/model-square',
    icon: 'Dify.Color',
  },
] satisfies HomeAppIcon[]
