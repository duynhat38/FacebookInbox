import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'

// ImagePicker shows the selected image path and opens the native file dialog.
export function ImagePicker() {
  const { t } = useT()
  const imagePath = useAppStore((s) => s.settings.imagePath)
  const pickImage = useAppStore((s) => s.pickImage)
  const clearImage = useAppStore((s) => s.clearImage)

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs font-semibold uppercase tracking-wider text-slate-300">
        {t('image.label')}
      </span>
      <div className="flex items-center gap-2">
        <div
          className="flex-1 truncate rounded-lg border border-slate-700 bg-slate-900/70 px-3 py-2 text-sm text-slate-300"
          title={imagePath || undefined}
        >
          {imagePath || (
            <span className="text-slate-500">{t('image.none')}</span>
          )}
        </div>
        <button
          onClick={() => void pickImage()}
          className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition hover:bg-brand-dark"
        >
          {t('image.pick')}
        </button>
        {imagePath && (
          <button
            onClick={() => void clearImage()}
            className="rounded-lg border border-slate-700 px-3 py-2 text-sm text-slate-300 transition hover:bg-slate-700/40"
          >
            {t('image.clear')}
          </button>
        )}
      </div>
    </div>
  )
}
