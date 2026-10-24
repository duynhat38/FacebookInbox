import { useState } from 'react'
import { Header } from './Header'
import { RunControls } from './RunControls'
import { Tabs } from '../../components/Tabs/Tabs'
import { PageList } from '../../components/PageList/PageList'
import { ProxyList } from '../../components/ProxyList/ProxyList'
import { ContentList } from '../../components/ContentList/ContentList'
import { ShareContentList } from '../../components/ShareContentList/ShareContentList'
import { LinkList } from '../../components/LinkList/LinkList'
import { AccountTable } from '../../components/AccountTable/AccountTable'
import { ImagePicker } from '../../components/ImagePicker/ImagePicker'
import { WorkerSettings } from '../../components/WorkerSettings/WorkerSettings'
import { ErrorBanner } from '../../components/common/ErrorBanner'
import { Panel } from '../../components/common/Panel'
import { useT } from '../../hooks/useT'

type TabId = 'content' | 'proxy'

// The app shell is a fixed-height row. The left column holds the tabs and the
// Account table (each scrolls internally); the config sits in a fixed-width column
// on the right. The window itself never scrolls. The Account table and config stay
// visible on both tabs.
export function Home() {
  const { t } = useT()
  const [tab, setTab] = useState<TabId>('content')

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-slate-900 text-slate-100">
      <Header />

      <main className="flex min-h-0 flex-1 gap-3 overflow-hidden p-3">
        {/* Main content column */}
        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
          <ErrorBanner />

          <Tabs
            tabs={[
              { id: 'content', label: t('tab.pageContentLink') },
              { id: 'proxy', label: t('tab.proxy') },
            ]}
            active={tab}
            onChange={(id) => setTab(id as TabId)}
          />

          {/* Tab panel */}
          <div className="flex min-h-0 flex-1 flex-col">
            {tab === 'content' ? (
              <div className="flex min-h-0 flex-1 flex-col gap-3">
                <div className="grid min-h-0 flex-1 grid-cols-1 gap-3 md:grid-cols-4">
                  <PageList />
                  <ContentList />
                  <ShareContentList />
                  <LinkList />
                </div>
                <ImagePicker />
              </div>
            ) : (
              <div className="flex min-h-0 flex-1 flex-col">
                <ProxyList />
              </div>
            )}
          </div>

          {/* Account table — always visible regardless of active tab */}
          <AccountTable />
        </div>

        {/* Configuration column (right side, always visible) */}
        <Panel title={t('settings.title')} className="w-80 shrink-0">
          <div className="flex flex-col gap-3">
            <WorkerSettings />
            <RunControls />
          </div>
        </Panel>
      </main>
    </div>
  )
}
