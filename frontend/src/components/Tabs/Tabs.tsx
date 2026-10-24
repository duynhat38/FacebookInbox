export interface TabDef {
  id: string
  label: string
}

interface Props {
  tabs: TabDef[]
  active: string
  onChange: (id: string) => void
}

// Tabs renders a horizontal tab bar with a clear active-state distinction.
export function Tabs({ tabs, active, onChange }: Props) {
  return (
    <div className="flex items-center gap-1 rounded-lg border border-slate-700/60 bg-slate-800/40 p-1">
      {tabs.map((tab) => {
        const isActive = tab.id === active
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            aria-selected={isActive}
            role="tab"
            className={`rounded-md px-4 py-1.5 text-sm font-semibold transition ${
              isActive
                ? 'bg-brand text-white shadow'
                : 'text-slate-400 hover:bg-slate-700/40 hover:text-slate-200'
            }`}
          >
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}
