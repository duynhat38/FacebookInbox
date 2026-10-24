import type { ReactNode } from 'react'
import { useT } from '../../hooks/useT'

export interface MenuPosition {
  x: number
  y: number
}

interface Props {
  position: MenuPosition | null
  canDelete: boolean
  onAdd: () => void
  onDelete: () => void
}

// ContextMenu is the presentational right-click menu for the Account table. Per
// spec it has exactly two actions: add Account and delete the selected Account.
// Opening and outside-close handling live in AccountTable.
export function ContextMenu({ position, canDelete, onAdd, onDelete }: Props) {
  const { t } = useT()

  if (!position) return null

  // Keep the menu within the viewport.
  const left = Math.min(position.x, window.innerWidth - 220)
  const top = Math.min(position.y, window.innerHeight - 110)

  return (
    <div
      role="menu"
      style={{ left, top }}
      className="fixed z-50 w-52 overflow-hidden rounded-lg border border-slate-600 bg-slate-800 py-1 shadow-xl"
      onContextMenu={(e) => e.preventDefault()}
    >
      <MenuItem onClick={onAdd}>{t('account.add')}</MenuItem>
      <MenuItem onClick={onDelete} disabled={!canDelete} danger>
        {t('account.deleteSelected')}
      </MenuItem>
    </div>
  )
}

function MenuItem({
  children,
  onClick,
  disabled,
  danger,
}: {
  children: ReactNode
  onClick: () => void
  disabled?: boolean
  danger?: boolean
}) {
  return (
    <button
      type="button"
      role="menuitem"
      disabled={disabled}
      onClick={onClick}
      className={`block w-full px-4 py-2 text-left text-sm transition disabled:cursor-not-allowed disabled:opacity-40 ${
        danger
          ? 'text-red-300 hover:bg-red-500/15'
          : 'text-slate-200 hover:bg-slate-700/60'
      }`}
    >
      {children}
    </button>
  )
}
