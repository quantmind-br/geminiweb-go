package render

import "testing"

var benchmarkContent = "# Hello World\n\n" +
	"This is **bold** and *italic* text with some `inline code`.\n\n" +
	"## Code Block\n\n" +
	"```go\n" +
	"func main() {\n" +
	"    fmt.Println(\"Hello, World!\")\n" +
	"}\n" +
	"```\n\n" +
	"## List\n\n" +
	"- Item 1\n" +
	"- Item 2\n" +
	"- Item 3\n\n" +
	"## Table\n\n" +
	"| Name | Age |\n" +
	"|------|-----|\n" +
	"| Alice | 30 |\n" +
	"| Bob | 25 |\n\n" +
	":smile: :heart: :rocket:\n"

func BenchmarkMarkdownNoCache(b *testing.B) {
	opts := DefaultOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate without cache (clear cache each iteration)
		ClearCache()
		_, err := Markdown(benchmarkContent, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarkdownWithCache(b *testing.B) {
	opts := DefaultOptions()

	// Pre-populate cache
	_, err := Markdown(benchmarkContent, opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Markdown(benchmarkContent, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarkdownParallel(b *testing.B) {
	opts := DefaultOptions()

	// Pre-populate cache
	_, err := Markdown(benchmarkContent, opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Markdown(benchmarkContent, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
