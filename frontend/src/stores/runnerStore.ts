import { create } from 'zustand'
import * as App from '../../wailsjs/go/app/App'
import type { Stats } from '../types'
import { useAppStore } from './appStore'

interface RunnerState {
  stats: Stats | null
  isRunning: boolean

  start: () => Promise<void>
  stop: () => Promise<void>
  setStats: (stats: Stats) => void
  refresh: () => Promise<void>
}

export const useRunnerStore = create<RunnerState>((set) => ({
  stats: null,
  isRunning: false,

  start: async () => {
    useAppStore.getState().setError(null)
    try {
      await App.Start()
      set({ isRunning: true })
    } catch (e) {
      // Surface the raw backend key; components localize it for display.
      useAppStore.getState().setError(String(e))
    }
  },

  stop: async () => {
    try {
      await App.Stop()
    } catch (e) {
      useAppStore.getState().setError(String(e))
    }
  },

  setStats: (stats) => {
    set({ stats, isRunning: stats.state === 'running' })
  },

  refresh: async () => {
    try {
      const [stats, running] = await Promise.all([App.GetStats(), App.IsRunning()])
      set({ stats, isRunning: running })
    } catch {
      // Non-fatal: stats will arrive via events.
    }
  },
}))
