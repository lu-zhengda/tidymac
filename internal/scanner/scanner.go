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
	Path        string
	Size        int64
	Category    string
	Description string
	Risk        RiskLevel
	ModTime     time.Time
	IsDir       bool
}

type Scanner interface {
	Name() string
	Description() string
	Scan(ctx context.Context) ([]Target, error)
	Risk() RiskLevel
}
