package domain

// RunnerState is the overall lifecycle of the runner.
type RunnerState string

const (
	RunnerStopped RunnerState = "stopped"
	RunnerRunning RunnerState = "running"
	RunnerDone    RunnerState = "done"
)

// Stats is the aggregate snapshot pushed to the UI. It bundles global counters
// together with per-Accounts snapshots so the frontend can render the single Accounts
// table from one event payload.
type Stats struct {
	State        RunnerState       `json:"state"`
	TotalWorkers int               `json:"totalWorkers"`
	Running      int               `json:"running"`
	Success      int               `json:"success"`
	Errors       int               `json:"errors"`
	Remaining    int               `json:"remaining"`
	TotalPages   int               `json:"totalPages"`
	Processed    int               `json:"processed"`
	Accounts     []AccountSnapshot `json:"accounts"`
}
