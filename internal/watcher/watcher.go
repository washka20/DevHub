package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// skipDirs contains directory names that are never watched.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, ".superpowers": true,
	".claude": true, ".idea": true, ".vscode": true,
	".worktrees": true,
}

// Event is the payload broadcast to WebSocket clients on file changes.
type Event struct {
	Type    string   `json:"type"`
	Project string   `json:"project"`
	Paths   []string `json:"paths"`
}

// BroadcastFunc is called with a file-change event after debouncing.
type BroadcastFunc func(event Event)

// pending groups buffered changes per project.
type pending struct {
	paths map[string]bool
	timer *time.Timer
}

// Watcher watches a directory tree and broadcasts debounced change events.
type Watcher struct {
	fw        *fsnotify.Watcher
	root      string
	broadcast BroadcastFunc
	mu        sync.Mutex
	projects  map[string]*pending // keyed by project name (first path segment)
	done      chan struct{}
}

// New creates a new Watcher with the given broadcast callback.
func New(broadcast BroadcastFunc) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		fw:        fw,
		broadcast: broadcast,
		projects:  make(map[string]*pending),
		done:      make(chan struct{}),
	}, nil
}

// Watch recursively adds dir and all non-skipped subdirectories to the watcher.
func (w *Watcher) Watch(dir string) error {
	w.mu.Lock()
	w.root = dir
	w.mu.Unlock()

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible dirs
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] || (strings.HasPrefix(name, ".") && path != dir) {
				return filepath.SkipDir
			}
			if addErr := w.fw.Add(path); addErr != nil {
				// Permission denied or too many watchers — skip silently
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})
}

// Start launches the background event loop. Call Watch before Start.
func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.fw.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}

				w.mu.Lock()

				// Automatically watch newly created directories.
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						name := filepath.Base(event.Name)
						if !skipDirs[name] && !strings.HasPrefix(name, ".") {
							_ = w.fw.Add(event.Name)
						}
					}
				}

				rel, err := filepath.Rel(w.root, event.Name)
				if err != nil || rel == "" || rel == "." {
					w.mu.Unlock()
					continue
				}

				// Derive project name from the first path segment.
				project := strings.SplitN(rel, string(filepath.Separator), 2)[0]

				p, ok := w.projects[project]
				if !ok {
					p = &pending{paths: make(map[string]bool)}
					w.projects[project] = p
				}
				p.paths[rel] = true

				if p.timer != nil {
					p.timer.Stop()
				}
				// Capture for closure.
				proj := project
				p.timer = time.AfterFunc(500*time.Millisecond, func() {
					w.flush(proj)
				})

				w.mu.Unlock()

			case err, ok := <-w.fw.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)

			case <-w.done:
				return
			}
		}
	}()
}

// flush collects pending paths for a project and broadcasts the event.
func (w *Watcher) flush(project string) {
	w.mu.Lock()
	p, ok := w.projects[project]
	if !ok || len(p.paths) == 0 {
		w.mu.Unlock()
		return
	}
	paths := make([]string, 0, len(p.paths))
	for path := range p.paths {
		paths = append(paths, path)
	}
	p.paths = make(map[string]bool)
	w.mu.Unlock()

	w.broadcast(Event{
		Type:    "files_changed",
		Project: project,
		Paths:   paths,
	})
}

// Close shuts down the watcher and releases resources.
func (w *Watcher) Close() {
	close(w.done)
	_ = w.fw.Close()
}
