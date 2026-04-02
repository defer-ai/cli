package skills

import (
	"os"
	"path/filepath"
	"time"
)

// DiscoverSkillDirs walks up from cwd to root, collecting .defer/skills/ directories.
// Deeper paths have higher priority (project-level overrides parent-level).
// The returned slice is ordered from shallowest (root-most) to deepest (cwd-most),
// so later entries override earlier ones when loading skills.
func DiscoverSkillDirs(cwd string) []string {
	var dirs []string
	dir := filepath.Clean(cwd)
	for {
		candidate := filepath.Join(dir, ".defer", "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			dirs = append(dirs, candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// dirs is collected deepest-first; reverse so shallowest is first.
	// This means deeper paths (higher priority) come last.
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}

	return dirs
}

// WatchSkillDirs returns a channel that emits when skill files change.
// Uses filesystem polling (every 5 seconds) since fsnotify adds complexity.
// Caller should close the returned stop channel to stop watching.
func WatchSkillDirs(dirs []string) (<-chan struct{}, chan<- struct{}) {
	return WatchSkillDirsInterval(dirs, 5*time.Second)
}

// WatchSkillDirsInterval is like WatchSkillDirs but with a configurable poll interval.
// Exported for testing with shorter intervals.
func WatchSkillDirsInterval(dirs []string, interval time.Duration) (<-chan struct{}, chan<- struct{}) {
	notify := make(chan struct{}, 1)
	stop := make(chan struct{})

	// snapshot captures mod times for all files in the watched directories.
	snapshot := func() map[string]time.Time {
		state := make(map[string]time.Time)
		for _, d := range dirs {
			entries, err := os.ReadDir(d)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				path := filepath.Join(d, entry.Name())
				state[path] = info.ModTime()
			}
		}
		return state
	}

	prev := snapshot()

	go func() {
		defer close(notify)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				curr := snapshot()
				if changed(prev, curr) {
					// Non-blocking send; if the channel already has a
					// pending notification, skip.
					select {
					case notify <- struct{}{}:
					default:
					}
					prev = curr
				}
			}
		}
	}()

	return notify, stop
}

// changed returns true if the two snapshots differ (different files or different mod times).
func changed(prev, curr map[string]time.Time) bool {
	if len(prev) != len(curr) {
		return true
	}
	for path, modTime := range prev {
		if currMod, ok := curr[path]; !ok || !currMod.Equal(modTime) {
			return true
		}
	}
	return false
}

// MergeSkills merges default skills with loaded skills.
// Loaded skills override defaults by name.
func MergeSkills(defaults map[string]Skill, loaded []Skill) map[string]Skill {
	merged := make(map[string]Skill, len(defaults)+len(loaded))

	for name, skill := range defaults {
		merged[name] = skill
	}

	for _, skill := range loaded {
		merged[skill.Name] = skill
	}

	return merged
}
