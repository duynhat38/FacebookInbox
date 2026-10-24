import { useEffect } from 'react'
import { Home } from './pages/Home/Home'
import { useAppStore } from './stores/appStore'
import { useAccountStore } from './stores/accountStore'
import { useRunnerEvents } from './hooks/useRunnerEvents'

export default function App() {
  const init = useAppStore((s) => s.init)
  const loadAccounts = useAccountStore((s) => s.load)

  // Subscribe to backend stats events for the lifetime of the app.
  useRunnerEvents()

  useEffect(() => {
    void init()
    void loadAccounts()
  }, [init, loadAccounts])

  return <Home />
}
