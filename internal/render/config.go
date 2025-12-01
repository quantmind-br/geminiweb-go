package render

import (
	"os"

	"github.com/diogo/geminiweb/internal/config"
)

// LoadOptionsFromConfig loads render options from user configuration.
// Environment variables take precedence over config file values.
func LoadOptionsFromConfig() Options {
	opts := DefaultOptions()

	// Load from config file
	cfg, err := config.LoadConfig()
	if err == nil {
		md := cfg.Markdown
		// Only apply non-zero values from config
		if md.Style != "" {
			opts.Style = md.Style
		}
		// These booleans always overwrite defaults since they have explicit defaults in config
		opts.EnableEmoji = md.EnableEmoji
		opts.PreserveNewLines = md.PreserveNewLines
		opts.TableWrap = md.TableWrap
		opts.InlineTableLinks = md.InlineTableLinks
	}

	// Environment variable takes highest precedence for style
	if style := os.Getenv("GLAMOUR_STYLE"); style != "" {
		opts.Style = style
	}

	return opts
}

// LoadOptionsFromConfigWithWidth loads options from config with a specific width.
func LoadOptionsFromConfigWithWidth(width int) Options {
	opts := LoadOptionsFromConfig()
	opts.Width = width
	return opts
}
