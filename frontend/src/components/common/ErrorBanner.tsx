import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'

// ErrorBanner shows the latest localized error from the app store, dismissible.
export function ErrorBanner() {
  const error = useAppStore((s) => s.error)
  const setError = useAppStore((s) => s.setError)
  const { t, te } = useT()

  if (!error) return null

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-200">
      <span>
        <strong className="mr-1">{t('error.title')}:</strong>
        {te(error)}
      </span>
      <button
        onClick={() => setError(null)}
        className="rounded px-2 py-0.5 text-red-200/80 transition hover:bg-red-500/20 hover:text-white"
      >
        ✕
      </button>
    </div>
  )
}
