import { useAppStore } from '../../stores/appStore'
import { useAccountStore } from '../../stores/accountStore'
import { useRunnerStore } from '../../stores/runnerStore'
import { useT } from '../../hooks/useT'
import { countLines } from '../../utils/lines'

export function RunControls() {
  const { t } = useT()
  const pagesText = useAppStore((s) => s.pagesText)
  const accountCount = useAccountStore((s) => s.items.length)
  const isRunning = useRunnerStore((s) => s.isRunning)
  const start = useRunnerStore((s) => s.start)
  const stop = useRunnerStore((s) => s.stop)

  const canStart = countLines(pagesText) > 0 && accountCount > 0

  return (
    <div className="flex flex-col gap-3">
      <button
        onClick={() => void start()}
        disabled={isRunning || !canStart}
        className="w-full rounded-lg bg-emerald-600 px-6 py-2.5 text-sm font-bold tracking-wide text-white shadow transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:text-slate-400 disabled:shadow-none"
      >
        {t('actions.start')}
      </button>
      <button
        onClick={() => void stop()}
        disabled={!isRunning}
        className="w-full rounded-lg bg-red-600 px-6 py-2.5 text-sm font-bold tracking-wide text-white shadow transition hover:bg-red-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:text-slate-400 disabled:shadow-none"
      >
        {t('actions.stop')}
      </button>
    </div>
  )
}
