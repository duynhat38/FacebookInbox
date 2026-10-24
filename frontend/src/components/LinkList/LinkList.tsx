import { Panel } from '../common/Panel'
import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { countLines } from '../../utils/lines'

// LinkList is the editable, auto-saving list of links (one per line).
export function LinkList() {
  const { t } = useT()
  const linksText = useAppStore((s) => s.linksText)
  const setLinksText = useAppStore((s) => s.setLinksText)
  const count = countLines(linksText)

  return (
    <Panel
      title={t('link.title')}
      right={
        <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs font-medium text-slate-200">
          {t('common.count')}: {count}
        </span>
      }
      className="min-h-0 flex-1"
    >
      <textarea
        data-selectable="true"
        value={linksText}
        onChange={(e) => setLinksText(e.target.value)}
        placeholder={t('link.placeholder')}
        spellCheck={false}
        className="h-full min-h-0 w-full flex-1 resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-2 font-mono text-sm leading-snug text-slate-100 outline-none transition focus:border-brand"
      />
    </Panel>
  )
}
