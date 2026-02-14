package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/zhengda-lu/macbroom/internal/utils"
)

type browserProfile struct {
	name    string
	paths   []string
	useBase string // "caches" or "library"
}

type BrowserScanner struct {
	cachesBase  string
	libraryBase string
}

func NewBrowserScanner(cachesBase, libraryBase string) *BrowserScanner {
	return &BrowserScanner{cachesBase: cachesBase, libraryBase: libraryBase}
}

func (b *BrowserScanner) Name() string        { return "Browser Cache" }
func (b *BrowserScanner) Description() string { return "Browser caches, cookies, and local storage" }
func (b *BrowserScanner) Risk() RiskLevel     { return Moderate }

func (b *BrowserScanner) caches() string {
	if b.cachesBase != "" {
		return b.cachesBase
	}
	return utils.LibraryPath("Caches")
}

func (b *BrowserScanner) library() string {
	if b.libraryBase != "" {
		return b.libraryBase
	}
	return utils.LibraryPath("")
}

func (b *BrowserScanner) profiles() []browserProfile {
	return []browserProfile{
		{name: "Google Chrome", paths: []string{"Google/Chrome/Default/Cache", "Google/Chrome/Default/Code Cache", "Google/Chrome/Default/Service Worker"}, useBase: "caches"},
		{name: "Safari", paths: []string{"Safari"}, useBase: "library"},
		{name: "Firefox", paths: []string{"Firefox/Profiles"}, useBase: "caches"},
		{name: "Arc", paths: []string{"Arc/Cache", "Arc/Code Cache"}, useBase: "caches"},
	}
}

func (b *BrowserScanner) Scan(ctx context.Context) ([]Target, error) {
	var targets []Target

	for _, profile := range b.profiles() {
		select {
		case <-ctx.Done():
			return targets, ctx.Err()
		default:
		}

		for _, relPath := range profile.paths {
			var fullPath string
			if profile.useBase == "caches" {
				fullPath = filepath.Join(b.caches(), relPath)
			} else {
				fullPath = filepath.Join(b.library(), relPath)
			}

			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			var size int64
			if info.IsDir() {
				size, _ = utils.DirSize(fullPath)
			} else {
				size = info.Size()
			}

			if size == 0 {
				continue
			}

			targets = append(targets, Target{
				Path:        fullPath,
				Size:        size,
				Category:    "Browser Cache",
				Description: profile.name + " cache",
				Risk:        Moderate,
				ModTime:     info.ModTime(),
				IsDir:       info.IsDir(),
			})
		}
	}

	return targets, nil
}
