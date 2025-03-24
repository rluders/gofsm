package domain

import "time"

type Scan struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	Jobs      []*ScanJob `json:"jobs"`
}
