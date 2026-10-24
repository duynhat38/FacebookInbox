import type { ReactNode } from 'react'

interface Props {
  title?: string
  right?: ReactNode
  children: ReactNode
  className?: string
}

// Panel is the standard titled card used across the layout.
export function Panel({ title, right, children, className }: Props) {
  return (
    <section
      className={`flex flex-col rounded-xl border border-slate-700/60 bg-slate-800/40 ${className ?? ''}`}
    >
      {(title || right) && (
        <header className="flex items-center justify-between border-b border-slate-700/60 px-4 py-2.5">
          {title && (
            <h2 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
              {title}
            </h2>
          )}
          {right}
        </header>
      )}
      <div className="flex min-h-0 flex-1 flex-col p-3">{children}</div>
    </section>
  )
}
