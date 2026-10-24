import { Panel } from '../common/Panel'
import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { countContent } from '../../utils/lines'

// ShareContentList is the editable, auto-saving share content list. Items are
// separated by "|" (newlines also split); each item is prepended to the share
// URL when sending (e.g. "Check this out!\nhttps://fb.com/share/...").
export function ShareContentList() {
  const { t } = useT()
  const shareContentText = useAppStore((s) => s.shareContentText)
  const setShareContentText = useAppStore((s) => s.setShareContentText)
  const count = countContent(shareContentText)

  return (
    <Panel
      title={t('shareContent.title')}
      right={
        <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs font-medium text-slate-200">
          {t('common.count')}: {count}
        </span>
      }
      className="min-h-0"
    >
      <textarea
        data-selectable="true"
        value={shareContentText}
        onChange={(e) => setShareContentText(e.target.value)}
        placeholder={t('shareContent.placeholder')}
        spellCheck={false}
        className="h-full min-h-0 w-full flex-1 resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-2 font-mono text-sm leading-snug text-slate-100 outline-none transition focus:border-brand"
      />
    </Panel>
  )
}
