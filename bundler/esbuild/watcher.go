//go:build esbuild

package esbuild

import (
	"math/rand/v2"
	"os"
	"sync"
	"time"
)

// Polling constants matching esbuild's watch algorithm.
const (
	pollInterval             = 100 * time.Millisecond
	maxRecentItems           = 16
	minItemsPerIter          = 64
	maxIntervalsBeforeUpdate = 20
	modKeySafetyGap          = 3 * time.Second
)

// modEntry stores the last-known state of a watched file.
type modEntry struct {
	size    int64
	mtime   time.Time
	mode    os.FileMode
	content []byte // set only when mtime is within the safety gap
	missing bool   // true if the file did not exist at last check
}

// filePoller implements esbuild's two-tier polling watch algorithm.
// Recent files (up to 16) are checked every 100ms tick. The remaining
// files are randomly sampled so the full set is scanned every ~2 seconds.
type filePoller struct {
	mu          sync.Mutex
	items       map[string]modEntry // path → last-known state
	recentPaths []string            // up to maxRecentItems recently-changed paths
	recentSet   map[string]bool     // fast lookup for recent membership
	scanOrder   []string            // shuffled non-recent paths
	scanIndex   int                 // current position in scanOrder
	onChange    func()              // called when any file changes
	done        chan struct{}
}

// newFilePoller creates a poller that calls onChange when any watched file
// is modified. Call SetPaths to specify watched files, then Start.
func newFilePoller(onChange func()) *filePoller {
	return &filePoller{
		items:     make(map[string]modEntry),
		recentSet: make(map[string]bool),
		onChange:  onChange,
		done:      make(chan struct{}),
	}
}

// SetPaths replaces the set of watched file paths. Safe to call while polling.
func (p *filePoller) SetPaths(paths []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Build new items map, preserving existing entries.
	newItems := make(map[string]modEntry, len(paths))
	for _, path := range paths {
		if existing, ok := p.items[path]; ok {
			newItems[path] = existing
		} else {
			newItems[path] = p.snapshot(path)
		}
	}
	p.items = newItems

	// Prune recent list of removed paths.
	kept := p.recentPaths[:0]
	for _, rp := range p.recentPaths {
		if _, ok := newItems[rp]; ok {
			kept = append(kept, rp)
		} else {
			delete(p.recentSet, rp)
		}
	}
	p.recentPaths = kept

	// Rebuild scan order from non-recent paths.
	p.rebuildScanOrder()
}

// Start launches the polling goroutine.
func (p *filePoller) Start() {
	go p.run()
}

// Stop signals the polling goroutine to exit.
func (p *filePoller) Stop() {
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

func (p *filePoller) run() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			if p.poll() {
				p.onChange()
			}
		}
	}
}

// poll runs a single polling iteration. Returns true if any file changed.
func (p *filePoller) poll() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Tier 1: check all recent items every tick.
	for _, path := range p.recentPaths {
		if p.checkPath(path) {
			return true
		}
	}

	// Tier 2: sample non-recent items.
	n := p.itemsPerIter()
	for range n {
		if p.scanIndex >= len(p.scanOrder) {
			p.shuffleScanOrder()
			p.scanIndex = 0
			if len(p.scanOrder) == 0 {
				break
			}
		}
		path := p.scanOrder[p.scanIndex]
		p.scanIndex++
		if p.checkPath(path) {
			p.promoteToRecent(path)
			return true
		}
	}
	return false
}

// checkPath stats a file and compares against the stored modEntry.
// Returns true if the file has changed. Caller must hold p.mu.
func (p *filePoller) checkPath(path string) bool {
	prev, ok := p.items[path]
	if !ok {
		return false
	}
	cur := p.snapshot(path)

	changed := false
	if cur.missing != prev.missing {
		changed = true
	} else if !cur.missing {
		if cur.size != prev.size || cur.mode != prev.mode {
			changed = true
		} else if !cur.mtime.Equal(prev.mtime) {
			changed = true
		} else if cur.content != nil && prev.content != nil {
			// Both within safety gap — compare content.
			if len(cur.content) != len(prev.content) {
				changed = true
			} else {
				for i := range cur.content {
					if cur.content[i] != prev.content[i] {
						changed = true
						break
					}
				}
			}
		}
	}

	if changed {
		p.items[path] = cur
	}
	return changed
}

// snapshot captures the current state of a file.
func (p *filePoller) snapshot(path string) modEntry {
	info, err := os.Stat(path)
	if err != nil {
		return modEntry{missing: true}
	}
	entry := modEntry{
		size:  info.Size(),
		mtime: info.ModTime(),
		mode:  info.Mode(),
	}
	// If the file was modified very recently, its mtime might not have
	// changed yet on file systems with coarse time resolution. Fall back
	// to content comparison (same as esbuild's 3-second safety gap).
	if time.Since(entry.mtime) < modKeySafetyGap {
		entry.content, _ = os.ReadFile(path)
	}
	return entry
}

// promoteToRecent adds a path to the recent list (checked every tick).
// Caller must hold p.mu.
func (p *filePoller) promoteToRecent(path string) {
	if p.recentSet[path] {
		return
	}
	if len(p.recentPaths) >= maxRecentItems {
		// Evict the oldest recent item back to the scan pool.
		evicted := p.recentPaths[0]
		p.recentPaths = p.recentPaths[1:]
		delete(p.recentSet, evicted)
		p.scanOrder = append(p.scanOrder, evicted)
	}
	p.recentPaths = append(p.recentPaths, path)
	p.recentSet[path] = true
	// Remove from scan order.
	p.rebuildScanOrder()
}

// itemsPerIter returns how many non-recent items to check this tick.
// Matches esbuild: max(minItemsPerIter, ceil(total / maxIntervalsBeforeUpdate)).
func (p *filePoller) itemsPerIter() int {
	total := len(p.scanOrder)
	if total == 0 {
		return 0
	}
	perIter := (total + maxIntervalsBeforeUpdate - 1) / maxIntervalsBeforeUpdate
	if perIter < minItemsPerIter {
		perIter = minItemsPerIter
	}
	if perIter > total {
		perIter = total
	}
	return perIter
}

// rebuildScanOrder rebuilds scanOrder from items that are not in the recent set
// and shuffles it. Caller must hold p.mu.
func (p *filePoller) rebuildScanOrder() {
	p.scanOrder = p.scanOrder[:0]
	for path := range p.items {
		if !p.recentSet[path] {
			p.scanOrder = append(p.scanOrder, path)
		}
	}
	p.shuffleScanOrder()
	p.scanIndex = 0
}

// shuffleScanOrder performs a Fisher-Yates shuffle on scanOrder.
func (p *filePoller) shuffleScanOrder() {
	for i := len(p.scanOrder) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		p.scanOrder[i], p.scanOrder[j] = p.scanOrder[j], p.scanOrder[i]
	}
}
