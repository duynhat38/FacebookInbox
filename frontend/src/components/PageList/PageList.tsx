import { Panel } from '../common/Panel'
import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { countLines } from '../../utils/lines'

// PageList is the editable, auto-saving list of Facebook Page IDs.
export function PageList() {
  const { t } = useT()
  const pagesText = useAppStore((s) => s.pagesText)
  const setPagesText = useAppStore((s) => s.setPagesText)
  const count = countLines(pagesText)

  return (
    <Panel
      title={t('pageList.title')}
      right={<Count value={count} label={t('common.count')} />}
      className="min-h-0"
    >
      <textarea
        data-selectable="true"
        value={pagesText}
        onChange={(e) => setPagesText(e.target.value)}
        placeholder={t('pageList.placeholder')}
        spellCheck={false}
        className="h-full min-h-0 w-full flex-1 resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-2 font-mono text-sm leading-snug text-slate-100 outline-none transition focus:border-brand"
      />
    </Panel>
  )
}

function Count({ value, label }: { value: number; label: string }) {
  return (
    <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs font-medium text-slate-200">
      {label}: {value}
    </span>
  )
}
