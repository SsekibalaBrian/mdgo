package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

//go:embed styles.css
var cssContent string

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder

	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune('-')
		case r == '-':
			b.WriteRune('-')
		}
	}

	out := strings.ReplaceAll(b.String(), "--", "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}

	return out
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Kullanım: mdgo dosyaismi.md")
		os.Exit(1)
	}

	inputFile := os.Args[1]

	if !strings.HasSuffix(inputFile, ".md") {
		log.Fatalf("Hata: Girdi dosyası .md uzantılı olmalı")
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Dosya okunamadı: %v", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(false),
					html.WithClasses(true),
				),
			),
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithUnsafe(),
		),
	)

	ctx := parser.NewContext()
	doc := md.Parser().Parse(
		text.NewReader(data),
		parser.WithContext(ctx),
	)

	headingCount := make(map[string]int)

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		h, ok := n.(*ast.Heading)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}

		var buf bytes.Buffer
		for c := h.FirstChild(); c != nil; c = c.NextSibling() {
			if txt, ok := c.(*ast.Text); ok {
				buf.Write(txt.Segment.Value(data))
			}
		}

		text := buf.String()
		slug := slugify(text)

		if headingCount[slug] > 0 {
			slug = fmt.Sprintf("%s-%d", slug, headingCount[slug])
		}
		headingCount[slug]++

		h.SetAttribute([]byte("id"), []byte(slug))
		return ast.WalkContinue, nil
	})

	var out bytes.Buffer
	if err := md.Renderer().Render(&out, data, doc); err != nil {
		log.Fatalf("Render hatası: %v", err)
	}

	finalHTML := out.String()
	bodyContent := finalHTML

	if strings.Contains(bodyContent, "<body>") {
		start := strings.Index(bodyContent, "<body>") + 6
		end := strings.Index(bodyContent, "</body>")
		if start > 6 && end > start {
			bodyContent = bodyContent[start:end]
		}
	}

	fullHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
` + cssContent + `
    </style>
</head>
<body>
    <div class="markdown-body">
` + bodyContent + `
    </div>
</body>
</html>`

	// Çıktı dosya adını oluştur
	outputFile := strings.TrimSuffix(inputFile, ".md") + ".html"

	// Aynı dizine yaz
	outputPath := filepath.Join(filepath.Dir(inputFile), filepath.Base(outputFile))

	if err := os.WriteFile(outputPath, []byte(fullHTML), 0644); err != nil {
		log.Fatalf("Dosyaya yazılamadı: %v", err)
	}

	fmt.Printf("✓ HTML dosyası oluşturuldu: %s\n", outputPath)
}
