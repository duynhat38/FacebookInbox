import { create } from 'zustand'
import * as App from '../../wailsjs/go/app/App'
import type { AccountItem } from '../types'
import { useAppStore } from './appStore'

interface AccountState {
  items: AccountItem[]
  selected: Set<string>
  lastClickedIndex: number | null

  load: () => Promise<void>
  add: (raw: string) => Promise<void>
  deleteSelected: () => Promise<void>

  // Selection helpers supporting single / ctrl-multi / shift-range clicks.
  selectSingle: (id: string, index: number) => void
  toggleSelect: (id: string, index: number) => void
  selectRange: (index: number) => void
  clearSelection: () => void
  isSelected: (id: string) => boolean
}

function reportError(e: unknown) {
  useAppStore.getState().setError(String(e))
}

export const useAccountStore = create<AccountState>((set, get) => ({
  items: [],
  selected: new Set<string>(),
  lastClickedIndex: null,

  load: async () => {
    try {
      const items = await App.GetAccounts()
      set({ items: items ?? [] })
    } catch (e) {
      reportError(e)
    }
  },

  add: async (raw) => {
    try {
      const items = await App.AddAccounts(raw)
      set({ items: items ?? [] })
    } catch (e) {
      reportError(e)
    }
  },

  deleteSelected: async () => {
    const ids = Array.from(get().selected)
    if (ids.length === 0) return
    try {
      const items = await App.DeleteAccounts(ids)
      set({ items: items ?? [], selected: new Set(), lastClickedIndex: null })
    } catch (e) {
      reportError(e)
    }
  },

  selectSingle: (id, index) => {
    set({ selected: new Set([id]), lastClickedIndex: index })
  },

  toggleSelect: (id, index) => {
    const next = new Set(get().selected)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    set({ selected: next, lastClickedIndex: index })
  },

  selectRange: (index) => {
    const { items, lastClickedIndex } = get()
    const anchor = lastClickedIndex ?? index
    const start = Math.min(anchor, index)
    const end = Math.max(anchor, index)
    const next = new Set(get().selected)
    for (let i = start; i <= end; i++) {
      if (items[i]) next.add(items[i].id)
    }
    set({ selected: next })
  },

  clearSelection: () => set({ selected: new Set(), lastClickedIndex: null }),

  isSelected: (id) => get().selected.has(id),
}))
