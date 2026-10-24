import { Panel } from '../common/Panel'
import { useAppStore } from '../../stores/appStore'
import { useT } from '../../hooks/useT'
import { countLines } from '../../utils/lines'

// ProxyList is the editable, auto-saving list of proxies.
export function ProxyList() {
  const { t } = useT()
  const proxiesText = useAppStore((s) => s.proxiesText)
  const setProxiesText = useAppStore((s) => s.setProxiesText)
  const count = countLines(proxiesText)

  return (
    <Panel
      title={t('proxyList.title')}
      right={
        <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs font-medium text-slate-200">
          {t('common.count')}: {count}
        </span>
      }
      className="min-h-0 flex-1"
    >
      <textarea
        data-selectable="true"
        value={proxiesText}
        onChange={(e) => setProxiesText(e.target.value)}
        placeholder={t('proxyList.placeholder')}
        spellCheck={false}
        className="h-full min-h-0 w-full flex-1 resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-2 font-mono text-sm leading-snug text-slate-100 outline-none transition focus:border-brand"
      />
    </Panel>
  )
}
