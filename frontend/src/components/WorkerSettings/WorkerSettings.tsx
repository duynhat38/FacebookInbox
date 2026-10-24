import { NumberField } from '../common/NumberField'
import { useAppStore } from '../../stores/appStore'
import { useRunnerStore } from '../../stores/runnerStore'
import { useT } from '../../hooks/useT'

// WorkerSettings renders the numeric runner settings and the share URL input.
// Each change auto-saves; inputs are disabled while a run is in progress.
export function WorkerSettings() {
  const { t } = useT()
  const settings = useAppStore((s) => s.settings)
  const updateSettings = useAppStore((s) => s.updateSettings)
  const isRunning = useRunnerStore((s) => s.isRunning)

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-1 gap-3">
        <NumberField
          label={t('settings.threads')}
          value={settings.threads}
          min={1}
          disabled={isRunning}
          onChange={(v) => void updateSettings({ threads: v })}
        />
        <NumberField
          label={t('settings.delay')}
          value={settings.delaySeconds}
          min={0}
          disabled={isRunning}
          onChange={(v) => void updateSettings({ delaySeconds: v })}
        />
        <NumberField
          label={t('settings.maxErrors')}
          value={settings.maxErrors}
          min={1}
          disabled={isRunning}
          onChange={(v) => void updateSettings({ maxErrors: v })}
        />
        <NumberField
          label={t('settings.pagesPerThread')}
          value={settings.pagesPerThread}
          min={1}
          disabled={isRunning}
          onChange={(v) => void updateSettings({ pagesPerThread: v })}
        />
        <NumberField
          label={t('settings.accountPerLink')}
          value={settings.accountPerLink}
          min={1}
          disabled={isRunning}
          onChange={(v) => void updateSettings({ accountPerLink: v })}
        />
        <label className="flex items-center gap-2 select-none cursor-pointer">
          <input
            type="checkbox"
            className="w-4 h-4 accent-primary cursor-pointer disabled:cursor-not-allowed"
            checked={settings.deleteThreadAfterSend ?? false}
            disabled={isRunning}
            onChange={(e) => void updateSettings({ deleteThreadAfterSend: e.target.checked })}
          />
          <span className="text-sm">{t('settings.deleteThread')}</span>
        </label>
        <label className="flex items-center gap-2 select-none cursor-pointer">
          <input
            type="checkbox"
            className="w-4 h-4 accent-primary cursor-pointer disabled:cursor-not-allowed"
            checked={settings.deleteMessageAfterSend ?? false}
            disabled={isRunning}
            onChange={(e) => void updateSettings({ deleteMessageAfterSend: e.target.checked })}
          />
          <span className="text-sm">{t('settings.deleteMessage')}</span>
        </label>
      </div>
    </div>
  )
}
