import { Panel } from '../common/Panel'
import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { countContent } from '../../utils/lines'

// ContentList is the editable, auto-saving content list. Items are separated by
// "|" (newlines also split); the count updates in realtime.
export function ContentList() {
  const { t } = useT()
  const contentText = useAppStore((s) => s.contentText)
  const setContentText = useAppStore((s) => s.setContentText)
  const count = countContent(contentText)

  return (
    <Panel
      title={t('content.title')}
      right={
        <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs font-medium text-slate-200">
          {t('common.count')}: {count}
        </span>
      }
      className="min-h-0"
    >
      <textarea
        data-selectable="true"
        value={contentText}
        onChange={(e) => setContentText(e.target.value)}
        placeholder={t('content.placeholder')}
        spellCheck={false}
        className="h-full min-h-0 w-full flex-1 resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-2 font-mono text-sm leading-snug text-slate-100 outline-none transition focus:border-brand"
      />
    </Panel>
  )
}
