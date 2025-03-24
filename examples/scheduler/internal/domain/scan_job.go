package domain

type ScanJob struct {
	ID     string `json:"id"`
	ScanID string `json:"scan_id"`
	State  string `json:"state"`
}
