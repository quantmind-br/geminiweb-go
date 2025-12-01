package render

// Markdown renders markdown content for terminal display.
// Uses a pooled renderer for better performance and thread safety.
func Markdown(content string, opts Options) (string, error) {
	renderer, err := globalPool.get(opts)
	if err != nil {
		return "", err
	}
	defer globalPool.put(opts, renderer)

	return renderer.Render(content)
}

// MarkdownWithWidth is a convenience function for rendering with specific width.
// Uses default options with the specified width.
func MarkdownWithWidth(content string, width int) (string, error) {
	opts := DefaultOptions().WithWidth(width)
	return Markdown(content, opts)
}
