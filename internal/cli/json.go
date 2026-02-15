package cli

import (
	"time"

	"github.com/lu-zhengda/macbroom/internal/dupes"
	"github.com/lu-zhengda/macbroom/internal/history"
	"github.com/lu-zhengda/macbroom/internal/scancache"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/trends"
)

// ---------------------------------------------------------------------------
// Scan JSON types
// ---------------------------------------------------------------------------

type scanJSON struct {
	Version     string             `json:"version"`
	Timestamp   time.Time          `json:"timestamp"`
	Categories  []scanCategoryJSON `json:"categories"`
	TotalSize   int64              `json:"total_size"`
	TotalItems  int                `json:"total_items"`
	RiskSummary riskJSON           `json:"risk_summary"`
	Diff        *diffJSON          `json:"diff,omitempty"`
}

type scanCategoryJSON struct {
	Name    string       `json:"name"`
	Size    int64        `json:"size"`
	Items   int          `json:"items"`
	Risk    string       `json:"risk"`
	Targets []targetJSON `json:"targets"`
}

type targetJSON struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Risk string `json:"risk"`
}

type riskJSON struct {
	Safe     int64 `json:"safe"`
	Moderate int64 `json:"moderate"`
	Risky    int64 `json:"risky"`
}

type diffJSON struct {
	PreviousTimestamp time.Time                       `json:"previous_timestamp"`
	TotalDelta        int64                           `json:"total_delta"`
	Categories        map[string]scancache.CategoryDiff `json:"categories"`
}

// buildScanJSON groups targets by category and builds a JSON-serializable structure.
func buildScanJSON(targets []scanner.Target, diff *scancache.DiffResult) scanJSON {
	grouped := make(map[string][]scanner.Target)
	for _, t := range targets {
		grouped[t.Category] = append(grouped[t.Category], t)
	}

	var categories []scanCategoryJSON
	for name, items := range grouped {
		var catSize int64
		var catTargets []targetJSON
		// Determine dominant risk for the category.
		var maxRisk scanner.RiskLevel
		for _, item := range items {
			catSize += item.Size
			catTargets = append(catTargets, targetJSON{
				Path: item.Path,
				Size: item.Size,
				Risk: item.Risk.String(),
			})
			if item.Risk > maxRisk {
				maxRisk = item.Risk
			}
		}
		categories = append(categories, scanCategoryJSON{
			Name:    name,
			Size:    catSize,
			Items:   len(items),
			Risk:    maxRisk.String(),
			Targets: catTargets,
		})
	}

	rb := riskSummary(targets)

	result := scanJSON{
		Version:    version,
		Timestamp:  time.Now().UTC(),
		Categories: categories,
		TotalSize:  rb.Total,
		TotalItems: len(targets),
		RiskSummary: riskJSON{
			Safe:     rb.Safe,
			Moderate: rb.Moderate,
			Risky:    rb.Risky,
		},
	}

	if diff != nil {
		result.Diff = &diffJSON{
			PreviousTimestamp: diff.PreviousTimestamp,
			TotalDelta:        diff.TotalDelta,
			Categories:        diff.Categories,
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Clean JSON type
// ---------------------------------------------------------------------------

type cleanJSON struct {
	scanJSON
	DeletedSize  int64 `json:"deleted_size"`
	DeletedItems int   `json:"deleted_items"`
	Errors       int   `json:"errors"`
}

// ---------------------------------------------------------------------------
// Dupes JSON types
// ---------------------------------------------------------------------------

type dupesJSON struct {
	Version    string          `json:"version"`
	Timestamp  time.Time       `json:"timestamp"`
	Groups     []dupeGroupJSON `json:"groups"`
	TotalFiles int             `json:"total_files"`
	TotalWaste int64           `json:"total_waste"`
}

type dupeGroupJSON struct {
	Size  int64    `json:"size"`
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
}

// buildDupesJSON converts duplicate groups into a JSON-serializable structure.
func buildDupesJSON(groups []dupes.Group) dupesJSON {
	var jsonGroups []dupeGroupJSON
	var totalFiles int
	var totalWaste int64

	for _, g := range groups {
		jsonGroups = append(jsonGroups, dupeGroupJSON{
			Size:  g.Size,
			Hash:  g.Hash,
			Files: g.Files,
		})
		totalFiles += len(g.Files)
		totalWaste += g.Size * int64(len(g.Files)-1)
	}

	return dupesJSON{
		Version:    version,
		Timestamp:  time.Now().UTC(),
		Groups:     jsonGroups,
		TotalFiles: totalFiles,
		TotalWaste: totalWaste,
	}
}

// ---------------------------------------------------------------------------
// Stats JSON type
// ---------------------------------------------------------------------------

type statsJSON struct {
	Version       string                         `json:"version"`
	TotalFreed    int64                          `json:"total_freed"`
	TotalCleanups int                            `json:"total_cleanups"`
	ByCategory    map[string]history.CategoryStats `json:"by_category"`
	Recent        []history.Entry                `json:"recent"`
}

// buildStatsJSON converts history stats into a JSON-serializable structure.
func buildStatsJSON(stats history.Stats) statsJSON {
	return statsJSON{
		Version:       version,
		TotalFreed:    stats.TotalFreed,
		TotalCleanups: stats.TotalCleanups,
		ByCategory:    stats.ByCategory,
		Recent:        stats.Recent,
	}
}

// ---------------------------------------------------------------------------
// SpaceLens JSON type
// ---------------------------------------------------------------------------

type spaceLensJSON struct {
	Version   string                `json:"version"`
	Timestamp time.Time             `json:"timestamp"`
	Path      string                `json:"path"`
	Nodes     []scanner.SpaceLensNode `json:"nodes"`
}

// buildSpaceLensJSON converts spacelens analysis into a JSON-serializable structure.
func buildSpaceLensJSON(path string, nodes []scanner.SpaceLensNode) spaceLensJSON {
	return spaceLensJSON{
		Version:   version,
		Timestamp: time.Now().UTC(),
		Path:      path,
		Nodes:     nodes,
	}
}

// ---------------------------------------------------------------------------
// Uninstall JSON type
// ---------------------------------------------------------------------------

type uninstallJSON struct {
	Version   string       `json:"version"`
	Timestamp time.Time    `json:"timestamp"`
	AppName   string       `json:"app_name"`
	Items     int          `json:"items"`
	TotalSize int64        `json:"total_size"`
	Targets   []targetJSON `json:"targets"`
}

// buildUninstallJSON converts uninstall targets into a JSON-serializable structure.
func buildUninstallJSON(appName string, targets []scanner.Target) uninstallJSON {
	var totalSize int64
	jsonTargets := make([]targetJSON, 0, len(targets))
	for _, t := range targets {
		totalSize += t.Size
		jsonTargets = append(jsonTargets, targetJSON{
			Path: t.Path,
			Size: t.Size,
			Risk: t.Risk.String(),
		})
	}
	return uninstallJSON{
		Version:   version,
		Timestamp: time.Now().UTC(),
		AppName:   appName,
		Items:     len(targets),
		TotalSize: totalSize,
		Targets:   jsonTargets,
	}
}

// ---------------------------------------------------------------------------
// Report JSON type
// ---------------------------------------------------------------------------

type reportJSON struct {
	Version       string           `json:"version"`
	Timestamp     time.Time        `json:"timestamp"`
	TotalFreed    int64            `json:"total_freed"`
	TotalCleanups int              `json:"total_cleanups"`
	ByCategory    map[string]int64 `json:"by_category"`
	Recent        []history.Entry  `json:"recent"`
}

// ---------------------------------------------------------------------------
// Watch JSON type
// ---------------------------------------------------------------------------

type watchAlertJSON struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	FreeBytes int64     `json:"free_bytes"`
	Threshold int64     `json:"threshold"`
	Alert     bool      `json:"alert"`
	Message   string    `json:"message"`
}

// ---------------------------------------------------------------------------
// Trends JSON types
// ---------------------------------------------------------------------------

type trendsJSON struct {
	Version   string             `json:"version"`
	Timestamp time.Time          `json:"timestamp"`
	Report    trends.TrendReport `json:"report"`
}

type trendsRecordJSON struct {
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Snapshot  trends.StorageSnapshot `json:"snapshot"`
}
