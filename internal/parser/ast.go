package parser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// CodeBlock represents a parsed code block from markdown content.
type CodeBlock struct {
	// Hint is the content of the paragraph immediately preceding the code block.
	Hint string
	// Lang is the language identifier of the code block (e.g., "go", "diff").
	Lang string
	// Content is the raw text inside the code block.
	Content string
}

// ExtractCodeBlocks uses a markdown AST to find all fenced code blocks
// and their preceding paragraph, which is treated as a hint.
func ExtractCodeBlocks(source []byte) ([]CodeBlock, error) {
	var blocks []CodeBlock
	parser := goldmark.DefaultParser()
	root := parser.Parse(text.NewReader(source))

	walker := func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fencedCodeBlock, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		var block CodeBlock
		if fencedCodeBlock.Info != nil {
			block.Lang = string(fencedCodeBlock.Info.Text(source))
		}

		var content bytes.Buffer
		lines := fencedCodeBlock.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			content.Write(line.Value(source))
		}
		block.Content = content.String()

		if prev := fencedCodeBlock.PreviousSibling(); prev != nil {
			if p, ok := prev.(*ast.Paragraph); ok {
				block.Hint = strings.TrimSpace(string(p.Text(source)))
			}
		}

		blocks = append(blocks, block)
		return ast.WalkSkipChildren, nil
	}

	if err := ast.Walk(root, walker); err != nil {
		return nil, err
	}

	return blocks, nil
}
