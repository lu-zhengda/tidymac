package scanner

import (
	"context"
	"time"
)

type RiskLevel int

const (
	Safe RiskLevel = iota
	Moderate
	Risky
)

func (r RiskLevel) String() string {
	switch r {
	case Safe:
		return "Safe"
	case Moderate:
		return "Moderate"
	case Risky:
		return "Risky"
	default:
		return "Unknown"
	}
}

type Target struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Risk        RiskLevel `json:"risk"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
}

type Scanner interface {
	Name() string
	Description() string
	Scan(ctx context.Context) ([]Target, error)
	Risk() RiskLevel
}
