import { useEffect, useRef, useState } from 'react'
import { useT } from '../../hooks/useT'

interface Props {
  open: boolean
  onClose: () => void
  onAdd: (raw: string) => void
}

// AddAccountDialog is a modal that lets the user paste multiple Account values, one
// per line.
export function AddAccountDialog({ open, onClose, onAdd }: Props) {
  const { t } = useT()
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (open) {
      setText('')
      // Focus after the dialog mounts.
      requestAnimationFrame(() => textareaRef.current?.focus())
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const submit = () => {
    if (text.trim() !== '') onAdd(text)
    onClose()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onMouseDown={onClose}
    >
      <div
        className="w-full max-w-lg rounded-xl border border-slate-700 bg-slate-800 shadow-2xl"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <header className="border-b border-slate-700 px-5 py-3">
          <h3 className="text-sm font-semibold text-slate-100">
            {t('account.addDialogTitle')}
          </h3>
        </header>
        <div className="p-5">
          <textarea
            ref={textareaRef}
            data-selectable="true"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={t('account.addPlaceholder')}
            spellCheck={false}
            className="h-44 w-full resize-none rounded-lg border border-slate-700 bg-slate-900/70 p-3 font-mono text-sm text-slate-100 outline-none focus:border-brand"
          />
        </div>
        <footer className="flex justify-end gap-2 border-t border-slate-700 px-5 py-3">
          <button
            onClick={onClose}
            className="rounded-lg border border-slate-600 px-4 py-2 text-sm text-slate-300 transition hover:bg-slate-700/50"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={submit}
            className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition hover:bg-brand-dark"
          >
            {t('account.confirmAdd')}
          </button>
        </footer>
      </div>
    </div>
  )
}
