import { domain } from '../../wailsjs/go/models'

// Re-export the Go-generated domain models under friendly names so components
// import from a single place.
export type Settings = domain.Settings
export type AccountItem = domain.AccountItem
export type AccountSnapshot = domain.AccountSnapshot
export type Stats = domain.Stats

export type Language = 'vi' | 'en'

// RunnerState mirrors the Go domain.RunnerState constants.
export type RunnerState = 'stopped' | 'running' | 'done'
