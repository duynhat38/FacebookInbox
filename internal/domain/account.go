package domain

// AccountItem represents a single unit of "Account" managed in the table. In
// this application an Account value stands for an authorized Facebook Page
// credential or token the operator is permitted to use.
//
// Only ID and Value are persisted to data/account.json. Runtime information
// (proxy, success, errors, activity log) is NOT stored on disk — it is produced
// by the runner and surfaced through AccountSnapshot, keeping credential
// persistence separate from runtime state.
type AccountItem struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// AccountFile is the on-disk shape of data/account.json.
type AccountFile struct {
	Items []AccountItem `json:"items"`
}

// AccountSnapshot is an immutable, runtime view of one Account row sent to the
// frontend. The "Log" field is the live activity message shown in the
// "Trạng thái" column (e.g. "Send Mess đến 100001"); it is NOT a status enum.
type AccountSnapshot struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Proxy   string `json:"proxy"`
	Success int    `json:"success"`
	Errors  int    `json:"errors"`
	Log     string `json:"log"`
	Running bool   `json:"running"`
	Started bool   `json:"started"`
}

// WorkerStatus is the internal lifecycle state of a worker. It is deliberately
// kept separate from the Account table's activity log (see AccountSnapshot.Log) and
// is never shown directly in the "Trạng thái" column.
type WorkerStatus string

const (
	WorkerIdle      WorkerStatus = "idle"
	WorkerRunning   WorkerStatus = "running"
	WorkerStopped   WorkerStatus = "stopped"
	WorkerCompleted WorkerStatus = "completed"
	WorkerError     WorkerStatus = "error"
)

// WorkerRuntime is the in-memory runtime state the runner keeps for a worker.
// It is never persisted.
type WorkerRuntime struct {
	ID         int
	AccountID  string
	Proxy      string
	Success    int
	Errors     int
	CurrentLog string
	Running    bool
}
