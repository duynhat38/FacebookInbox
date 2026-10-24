import { useEffect } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { domain } from '../../wailsjs/go/models'
import { useRunnerStore } from '../stores/runnerStore'
import { useAppStore } from '../stores/appStore'

// useRunnerEvents subscribes to the backend 'runner:stats' event and feeds each
// pushed snapshot into the runner store. This is the push channel from Go to
// React; the UI never polls for stats.
export function useRunnerEvents() {
  const setStats = useRunnerStore((s) => s.setStats)
  const refresh = useRunnerStore((s) => s.refresh)

  useEffect(() => {
    // Pick up any state that exists before the first event arrives.
    void refresh()

    const cancelStats = EventsOn('runner:stats', (payload: unknown) => {
      setStats(domain.Stats.createFrom(payload))
    })

    // Remaining pages after the runner consumes successful ones.
    const cancelPages = EventsOn('pages:updated', (payload: unknown) => {
      const pages = Array.isArray(payload) ? (payload as string[]) : []
      useAppStore.getState().applyPagesFromBackend(pages)
    })

    return () => {
      cancelStats()
      cancelPages()
    }
  }, [setStats, refresh])
}
