import { useEffect, useMemo, useState, type MouseEvent } from 'react'
import { Panel } from '../common/Panel'
import { useAccountStore } from '../../stores/accountStore'
import { useRunnerStore } from '../../stores/runnerStore'
import { useT } from '../../hooks/useT'
import type { AccountSnapshot } from '../../types'
import { AddAccountDialog } from './AddAccountDialog'
import { ContextMenu, type MenuPosition } from './ContextMenu'

interface Row {
  id: string
  index: number
  value: string
  proxy: string
  success: number
  errors: number
  log: string
  started: boolean
  running: boolean
}

// AccountTable is the single table of the app. Each row is a Account item; while a
// run is in progress the runner overlays the worker's current proxy, success,
// errors and live activity log for that row. The "Trạng thái" column shows the
// runtime activity log, not a status enum. Account values are shown in plaintext
// (never masked) but are never logged.
export function AccountTable() {
  const { t } = useT()
  const items = useAccountStore((s) => s.items)
  const selected = useAccountStore((s) => s.selected)
  const selectSingle = useAccountStore((s) => s.selectSingle)
  const toggleSelect = useAccountStore((s) => s.toggleSelect)
  const selectRange = useAccountStore((s) => s.selectRange)
  const add = useAccountStore((s) => s.add)
  const deleteSelected = useAccountStore((s) => s.deleteSelected)
  const stats = useRunnerStore((s) => s.stats)

  const [menu, setMenu] = useState<MenuPosition | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  // Merge the persisted Account list with any live stats keyed by id.
  const rows: Row[] = useMemo(() => {
    const statMap = new Map<string, AccountSnapshot>()
    stats?.accounts?.forEach((d) => statMap.set(d.id, d))
    return items.map((item, index) => {
      const s = statMap.get(item.id)
      return {
        id: item.id,
        index: index + 1,
        value: item.value,
        proxy: s?.proxy ?? '',
        success: s?.success ?? 0,
        errors: s?.errors ?? 0,
        log: s?.log ?? '',
        started: s?.started ?? false,
        running: s?.running ?? false,
      }
    })
  }, [items, stats])

  // Row tint by worker state: amber while running, green when finished with no
  // outstanding errors, red when stopped with errors. Idle/not-started rows keep
  // the default striping. Selection still takes visual priority over all of these.
  const stateClass = (row: Row) => {
    if (!row.started) return ''
    if (row.running) return 'border-l-2 border-l-amber-400 bg-amber-400/5'
    if (row.errors > 0) return 'border-l-2 border-l-red-400 bg-red-400/5'
    if (row.success > 0) return 'border-l-2 border-l-emerald-400 bg-emerald-400/5'
    return ''
  }

  // Proxy cell: "—" before a row has run, "Direct" when running without a proxy,
  // otherwise the proxy endpoint currently in use.
  const proxyLabel = (row: Row) => {
    if (!row.started) return '—'
    return row.proxy || t('table.directShort')
  }

  // Close the menu on an outside mousedown, Escape, resize or scroll.
  useEffect(() => {
    if (!menu) return
    const close = () => setMenu(null)
    const onDown = (ev: globalThis.MouseEvent) => {
      const el = ev.target as HTMLElement | null
      if (el?.closest('[role="menu"]')) return
      if (ev.button === 2 && el?.closest('[data-datatable]')) return
      setMenu(null)
    }
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key === 'Escape') setMenu(null)
    }
    const id = window.setTimeout(() => {
      window.addEventListener('mousedown', onDown)
      window.addEventListener('keydown', onKey)
      window.addEventListener('resize', close)
      window.addEventListener('scroll', close, true)
    }, 0)
    return () => {
      window.clearTimeout(id)
      window.removeEventListener('mousedown', onDown)
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('resize', close)
      window.removeEventListener('scroll', close, true)
    }
  }, [menu])

  const handleRowClick = (e: MouseEvent, id: string, index: number) => {
    if (e.shiftKey) selectRange(index)
    else if (e.ctrlKey || e.metaKey) toggleSelect(id, index)
    else selectSingle(id, index)
  }

  const openMenu = (
    clientX: number,
    clientY: number,
    id?: string,
    index?: number,
  ) => {
    if (id && index !== undefined && !selected.has(id)) {
      selectSingle(id, index)
    }
    setMenu({ x: clientX, y: clientY })
  }

  const onRowMouseDown = (e: MouseEvent, id: string, index: number) => {
    if (e.button === 2) {
      e.stopPropagation()
      openMenu(e.clientX, e.clientY, id, index)
    }
  }
  const onRowContextMenu = (e: MouseEvent, id: string, index: number) => {
    e.preventDefault()
    e.stopPropagation()
    openMenu(e.clientX, e.clientY, id, index)
  }

  const confirmDelete = () => {
    setMenu(null)
    if (selected.size > 2) {
      if (
        !window.confirm(t('account.confirmDelete').replace('{n}', String(selected.size)))
      ) {
        return
      }
    }
    void deleteSelected()
  }

  return (
    <Panel
      title={t('account.title')}
      right={
        <div className="flex items-center gap-2 text-xs">
          <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 font-medium text-slate-200">
            {t('common.count')}: {items.length}
          </span>
          {selected.size > 0 && (
            <span className="rounded-full bg-brand/20 px-2.5 py-0.5 font-medium text-blue-300">
              {t('account.selected')}: {selected.size}
            </span>
          )}
        </div>
      }
      className="min-h-0 flex-1"
    >
      <div
        data-datatable
        className="min-h-0 flex-1 overflow-auto rounded-lg border border-slate-700/60"
        onMouseDown={(e) => {
          if (e.button === 2) openMenu(e.clientX, e.clientY)
        }}
        onContextMenu={(e) => {
          e.preventDefault()
          openMenu(e.clientX, e.clientY)
        }}
      >
        {rows.length === 0 ? (
          <div className="flex h-full min-h-[80px] items-center justify-center p-6 text-center text-sm text-slate-500">
            {t('account.empty')}
          </div>
        ) : (
          <table className="w-full table-fixed border-collapse text-sm">
            <thead className="sticky top-0 z-10 bg-slate-800 text-left text-xs uppercase tracking-wide text-slate-400">
              <tr>
                <th className="w-[50px] px-3 py-2 font-semibold">{t('table.index')}</th>
                <th className="w-1/4 px-3 py-2 font-semibold">{t('table.account')}</th>
                <th className="w-1/5 px-3 py-2 font-semibold">{t('table.proxy')}</th>
                <th className="w-[90px] px-3 py-2 text-right font-semibold">
                  {t('table.success')}
                </th>
                <th className="w-[70px] px-3 py-2 text-right font-semibold">
                  {t('table.error')}
                </th>
                <th className="px-3 py-2 font-semibold">{t('table.status')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const isSel = selected.has(row.id)
                return (
                  <tr
                    key={row.id}
                    onClick={(e) => handleRowClick(e, row.id, row.index - 1)}
                    onMouseDown={(e) => onRowMouseDown(e, row.id, row.index - 1)}
                    onContextMenu={(e) => onRowContextMenu(e, row.id, row.index - 1)}
                    className={`cursor-pointer border-t border-slate-700/40 transition ${
                      isSel
                        ? 'bg-brand/20 text-blue-100'
                        : `${stateClass(row) || 'odd:bg-slate-900/30'} hover:bg-slate-700/30`
                    }`}
                  >
                    <td className="px-3 py-1.5 tabular-nums text-slate-500">
                      {row.index}
                    </td>
                    <td
                      data-selectable="true"
                      className="truncate px-3 py-1.5 font-mono text-slate-200"
                      title={row.value}
                    >
                      {row.value}
                    </td>
                    <td
                      className="truncate px-3 py-1.5 font-mono text-xs text-slate-400"
                      title={proxyLabel(row)}
                    >
                      {proxyLabel(row)}
                    </td>
                    <td className="px-3 py-1.5 text-right tabular-nums text-emerald-300">
                      {row.success}
                    </td>
                    <td className="px-3 py-1.5 text-right tabular-nums text-red-300">
                      {row.errors}
                    </td>
                    <td
                      className="truncate px-3 py-1.5 text-slate-300"
                      title={row.log || undefined}
                    >
                      {row.log || '—'}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      <ContextMenu
        position={menu}
        canDelete={selected.size > 0}
        onAdd={() => {
          setMenu(null)
          setDialogOpen(true)
        }}
        onDelete={confirmDelete}
      />

      <AddAccountDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onAdd={(raw) => void add(raw)}
      />
    </Panel>
  )
}
