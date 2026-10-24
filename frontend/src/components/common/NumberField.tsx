interface Props {
  label: string
  value: number
  min?: number
  onChange: (value: number) => void
  disabled?: boolean
}

// NumberField is a labelled integer input used for the runner settings.
export function NumberField({ label, value, min = 0, onChange, disabled }: Props) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[11px] font-medium uppercase tracking-wide text-slate-400">
        {label}
      </span>
      <input
        type="number"
        min={min}
        value={Number.isFinite(value) ? value : ''}
        disabled={disabled}
        onChange={(e) => {
          const n = parseInt(e.target.value, 10)
          onChange(Number.isNaN(n) ? 0 : n)
        }}
        className="w-full rounded-lg border border-slate-700 bg-slate-900/70 px-3 py-2 text-sm text-slate-100 outline-none transition focus:border-brand disabled:cursor-not-allowed disabled:opacity-50"
      />
    </label>
  )
}
