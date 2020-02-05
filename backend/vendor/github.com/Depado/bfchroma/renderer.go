// Package bfchroma provides an easy and extensible blackfriday renderer that
// uses the chroma syntax highlighter to render code blocks.
package bfchroma

import (
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	bf "github.com/russross/blackfriday/v2"
)

// Option defines the functional option type
type Option func(r *Renderer)

// Style is a function option allowing to set the style used by chroma
// Default : "monokai"
func Style(s string) Option {
	return func(r *Renderer) {
		r.Style = styles.Get(s)
	}
}

// ChromaStyle is an option to directly set the style of the renderer using a
// chroma style instead of a string
func ChromaStyle(s *chroma.Style) Option {
	return func(r *Renderer) {
		r.Style = s
	}
}

// WithoutAutodetect disables chroma's language detection when no codeblock
// extra information is given. It will fallback to a sane default instead of
// trying to detect the language.
func WithoutAutodetect() Option {
	return func(r *Renderer) {
		r.Autodetect = false
	}
}

// ChromaOptions allows to pass Chroma html.Option such as Standalone()
// WithClasses(), ClassPrefix(prefix)...
func ChromaOptions(options ...html.Option) Option {
	return func(r *Renderer) {
		r.ChromaOptions = options
	}
}

// Extend allows to specify the blackfriday renderer which is extended
func Extend(br bf.Renderer) Option {
	return func(r *Renderer) {
		r.Base = br
	}
}

// NewRenderer will return a new bfchroma renderer with sane defaults
func NewRenderer(options ...Option) *Renderer {
	r := &Renderer{
		Base: bf.NewHTMLRenderer(bf.HTMLRendererParameters{
			Flags: bf.CommonHTMLFlags,
		}),
		Style:      styles.Monokai,
		Autodetect: true,
	}
	for _, option := range options {
		option(r)
	}
	r.Formatter = html.New(r.ChromaOptions...)
	return r
}

// RenderWithChroma will render the given text to the w io.Writer
func (r *Renderer) RenderWithChroma(w io.Writer, text []byte, data bf.CodeBlockData) error {
	var lexer chroma.Lexer

	// Determining the lexer to use
	if len(data.Info) > 0 {
		lexer = lexers.Get(string(data.Info))
	} else if r.Autodetect {
		lexer = lexers.Analyse(string(text))
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, string(text))
	if err != nil {
		return err
	}
	return r.Formatter.Format(w, r.Style, iterator)
}

// Renderer is a custom Blackfriday renderer that uses the capabilities of
// chroma to highlight code with triple backtick notation
type Renderer struct {
	Base          bf.Renderer
	Autodetect    bool
	ChromaOptions []html.Option
	Style         *chroma.Style
	Formatter     *html.Formatter
}

// RenderNode satisfies the Renderer interface
func (r *Renderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	switch node.Type {
	case bf.CodeBlock:
		if err := r.RenderWithChroma(w, node.Literal, node.CodeBlockData); err != nil {
			return r.Base.RenderNode(w, node, entering)
		}
		return bf.SkipChildren
	default:
		return r.Base.RenderNode(w, node, entering)
	}
}

// RenderHeader satisfies the Renderer interface
func (r *Renderer) RenderHeader(w io.Writer, ast *bf.Node) {
	r.Base.RenderHeader(w, ast)
}

// RenderFooter satisfies the Renderer interface
func (r *Renderer) RenderFooter(w io.Writer, ast *bf.Node) {
	r.Base.RenderFooter(w, ast)
}
