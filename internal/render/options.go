// Package render provides markdown rendering utilities for terminal output.
package render

// Options configures the markdown renderer behavior.
type Options struct {
	// Width defines the maximum output width (default: 80)
	Width int

	// Style defines the theme: "dark", "light", or path to JSON file
	Style string

	// EnableEmoji converts :emoji: to unicode characters
	EnableEmoji bool

	// PreserveNewLines preserves original line breaks
	PreserveNewLines bool

	// TableWrap enables word wrap in table cells (glamour v0.10.0+)
	TableWrap bool

	// InlineTableLinks renders links inline in tables (glamour v0.10.0+)
	InlineTableLinks bool
}

// DefaultOptions returns the default configuration.
func DefaultOptions() Options {
	return Options{
		Width:            80,
		Style:            "dark",
		EnableEmoji:      true,
		PreserveNewLines: true,
		TableWrap:        true,
		InlineTableLinks: false,
	}
}

// WithWidth returns Options with the specified width.
func (o Options) WithWidth(width int) Options {
	o.Width = width
	return o
}

// WithStyle returns Options with the specified style.
func (o Options) WithStyle(style string) Options {
	o.Style = style
	return o
}

// WithEmoji returns Options with emoji support enabled/disabled.
func (o Options) WithEmoji(enabled bool) Options {
	o.EnableEmoji = enabled
	return o
}

// WithPreserveNewLines returns Options with newline preservation enabled/disabled.
func (o Options) WithPreserveNewLines(enabled bool) Options {
	o.PreserveNewLines = enabled
	return o
}

// WithTableWrap returns Options with table wrap enabled/disabled.
func (o Options) WithTableWrap(enabled bool) Options {
	o.TableWrap = enabled
	return o
}

// WithInlineTableLinks returns Options with inline table links enabled/disabled.
func (o Options) WithInlineTableLinks(enabled bool) Options {
	o.InlineTableLinks = enabled
	return o
}
