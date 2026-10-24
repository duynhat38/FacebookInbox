import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { languages } from '../../i18n'

// LanguageSwitcher toggles between the available UI languages. The choice is
// persisted via settings.
export function LanguageSwitcher() {
  const { lang, t } = useT()
  const setLanguage = useAppStore((s) => s.setLanguage)

  return (
    <div className="flex items-center gap-1 rounded-lg border border-slate-700 bg-slate-900/60 p-0.5">
      {languages.map((code) => (
        <button
          key={code}
          onClick={() => void setLanguage(code)}
          className={`rounded-md px-2.5 py-1 text-xs font-medium transition ${
            lang === code
              ? 'bg-brand text-white'
              : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          {t(`lang.${code}`)}
        </button>
      ))}
    </div>
  )
}
