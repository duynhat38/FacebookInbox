import { FacebookIcon } from '../../components/common/FacebookIcon'
import { LanguageSwitcher } from '../../components/common/LanguageSwitcher'
import { useT } from '../../hooks/useT'

export function Header() {
  const { t } = useT()
  return (
    <header className="flex items-center justify-between border-b border-slate-700/60 bg-slate-800/50 px-5 py-3">
      <div className="flex items-center gap-3">
        <FacebookIcon size={32} />
        <div>
          <h1 className="text-lg font-bold leading-tight text-slate-100">
            {t('app.title')}
          </h1>
          <p className="text-xs text-slate-400">{t('app.subtitle')}</p>
        </div>
      </div>
      <LanguageSwitcher />
    </header>
  )
}
