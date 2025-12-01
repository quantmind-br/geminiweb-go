package render

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
)

// rendererPool uses sync.Pool for thread-safe renderer reuse.
// Note: glamour.TermRenderer is NOT thread-safe for concurrent Render() calls,
// so we use sync.Pool to efficiently reuse renderers without sharing them.
type rendererPool struct {
	mu    sync.RWMutex
	pools map[string]*sync.Pool
}

var globalPool = &rendererPool{
	pools: make(map[string]*sync.Pool),
}

// cacheKey generates a unique key based on options.
func cacheKey(opts Options) string {
	return fmt.Sprintf("%s:%d:%t:%t:%t:%t",
		opts.Style,
		opts.Width,
		opts.EnableEmoji,
		opts.PreserveNewLines,
		opts.TableWrap,
		opts.InlineTableLinks,
	)
}

// getPool returns or creates a pool for the given options.
func (p *rendererPool) getPool(opts Options) *sync.Pool {
	key := cacheKey(opts)

	// Try fast read
	p.mu.RLock()
	if pool, ok := p.pools[key]; ok {
		p.mu.RUnlock()
		return pool
	}
	p.mu.RUnlock()

	// Create new pool
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if pool, ok := p.pools[key]; ok {
		return pool
	}

	pool := &sync.Pool{
		New: func() interface{} {
			renderer, err := createRenderer(opts)
			if err != nil {
				return nil
			}
			return renderer
		},
	}
	p.pools[key] = pool
	return pool
}

// get retrieves a renderer from the pool.
func (p *rendererPool) get(opts Options) (*glamour.TermRenderer, error) {
	pool := p.getPool(opts)
	renderer := pool.Get()
	if renderer == nil {
		// Pool's New function failed, try creating directly
		return createRenderer(opts)
	}
	return renderer.(*glamour.TermRenderer), nil
}

// put returns a renderer to the pool.
func (p *rendererPool) put(opts Options, renderer *glamour.TermRenderer) {
	if renderer == nil {
		return
	}
	pool := p.getPool(opts)
	pool.Put(renderer)
}

// createRenderer creates a new TermRenderer with the specified options.
func createRenderer(opts Options) (*glamour.TermRenderer, error) {
	style := opts.Style

	// Handle custom built-in themes (dark, tokyonight, catppuccin have custom versions with separators)
	if opts.Style == ThemeDark || opts.Style == ThemeTokyoNight || opts.Style == ThemeCatppuccin {
		tmpFile, err := WriteThemeToTempFile(opts.Style)
		if err != nil {
			return nil, err
		}
		if tmpFile != "" {
			style = tmpFile
		}
	}

	rendererOpts := []glamour.TermRendererOption{
		glamour.WithStylePath(style),
		glamour.WithWordWrap(opts.Width),
		glamour.WithTableWrap(opts.TableWrap),
		glamour.WithInlineTableLinks(opts.InlineTableLinks),
	}

	if opts.EnableEmoji {
		rendererOpts = append(rendererOpts, glamour.WithEmoji())
	}

	if opts.PreserveNewLines {
		rendererOpts = append(rendererOpts, glamour.WithPreservedNewLines())
	}

	return glamour.NewTermRenderer(rendererOpts...)
}

// ClearCache clears the renderer pools and theme cache (useful for testing).
func ClearCache() {
	globalPool.mu.Lock()
	globalPool.pools = make(map[string]*sync.Pool)
	globalPool.mu.Unlock()
	ClearThemeCache()
}

// CacheSize returns the number of unique pool configurations.
func CacheSize() int {
	globalPool.mu.RLock()
	defer globalPool.mu.RUnlock()
	return len(globalPool.pools)
}
