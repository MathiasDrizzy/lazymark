package theme

import (
	"fmt"
	"image/color"

	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

// hex devuelve un color como "#rrggbb", que es como Glamour recibe los colores.
func hex(c color.Color) *string {
	r, g, b, _ := c.RGBA()
	s := fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	return &s
}

// MarkdownStyle devuelve el estilo de Glamour de la vista previa con los colores del
// tema activo. Parte del estilo "dark" de Glamour (sangrías, prefijos, márgenes) y le
// cambia todos los colores por los de la paleta: así los títulos, enlaces, código,
// citas y tablas cambian con el tema y no usan los índices fijos de 256 colores.
func MarkdownStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig // copia por valor; solo se reemplazan punteros, nunca se escribe en los de Glamour

	s.Document.Color = hex(ColorText)
	s.BlockQuote.Color = hex(ColorSubtext0)
	s.Heading.Color = hex(ColorBlue)
	s.H1.Color, s.H1.BackgroundColor = hex(ColorBase), hex(ColorPeach)
	s.H2.Color = hex(ColorBlue)
	s.H3.Color = hex(ColorMauve)
	s.H4.Color = hex(ColorTeal)
	s.H5.Color = hex(ColorGreen)
	s.H6.Color = hex(ColorSubtext0)
	s.HorizontalRule.Color = hex(ColorOverlay0)
	s.Link.Color = hex(ColorBlue)
	s.LinkText.Color = hex(ColorTeal)
	s.Image.Color = hex(ColorMauve)
	s.ImageText.Color = hex(ColorOverlay0)
	s.Code.Color, s.Code.BackgroundColor = hex(ColorPeach), hex(ColorSurface0)
	s.Strong.Color = hex(ColorText)
	s.Emph.Color = hex(ColorText)
	s.Strikethrough.Color = hex(ColorOverlay0)
	s.Task.Color = hex(ColorGreen)
	s.Item.Color = hex(ColorPeach)
	s.Enumeration.Color = hex(ColorPeach)
	s.DefinitionTerm.Color = hex(ColorBlue)

	// Los bloques de código usan un estilo de Chroma propio por tema. Con Chroma
	// dentro del estilo de Glamour, este lo registra una sola vez con el nombre "charm" y
	// ya no lo actualiza: el código se quedaría con los colores del primer tema.
	s.CodeBlock.Color = hex(ColorSubtext0)
	s.CodeBlock.Chroma = nil
	s.CodeBlock.Theme = registerChromaStyle()
	return s
}

// ChromaFormatter es el formateador de Chroma para los bloques de código: truecolor, para
// que los colores sean exactamente los de la paleta y no su aproximación a 256 colores.
const ChromaFormatter = "terminal16m"

// registerChromaStyle registra (una vez por tema) el estilo de Chroma con los colores
// de la paleta activa y devuelve su nombre.
func registerChromaStyle() string {
	name := "lazymark-" + CurrentThemeName
	if _, ok := chromastyles.Registry[name]; ok {
		return name
	}
	fg := func(c color.Color) string { return *hex(c) }
	chromastyles.Register(chroma.MustNewStyle(name, chroma.StyleEntries{
		chroma.Text:                fg(ColorText),
		chroma.Error:               fg(ColorBase) + " bg:" + fg(ColorRed),
		chroma.Comment:             fg(ColorOverlay0),
		chroma.CommentPreproc:      fg(ColorPeach),
		chroma.Keyword:             fg(ColorBlue),
		chroma.KeywordReserved:     fg(ColorMauve),
		chroma.KeywordNamespace:    fg(ColorRed),
		chroma.KeywordType:         fg(ColorTeal),
		chroma.Operator:            fg(ColorRed),
		chroma.Punctuation:         fg(ColorSubtext0),
		chroma.Name:                fg(ColorText),
		chroma.NameBuiltin:         fg(ColorMauve),
		chroma.NameTag:             fg(ColorMauve),
		chroma.NameAttribute:       fg(ColorBlue),
		chroma.NameClass:           fg(ColorYellow) + " bold underline",
		chroma.NameConstant:        fg(ColorPeach),
		chroma.NameDecorator:       fg(ColorYellow),
		chroma.NameException:       fg(ColorRed),
		chroma.NameFunction:        fg(ColorGreen),
		chroma.NameOther:           fg(ColorText),
		chroma.Literal:             fg(ColorPeach),
		chroma.LiteralNumber:       fg(ColorTeal),
		chroma.LiteralDate:         fg(ColorTeal),
		chroma.LiteralString:       fg(ColorGreen),
		chroma.LiteralStringEscape: fg(ColorTeal),
		chroma.GenericDeleted:      fg(ColorRed),
		chroma.GenericEmph:         "italic",
		chroma.GenericInserted:     fg(ColorGreen),
		chroma.GenericStrong:       "bold",
		chroma.GenericSubheading:   fg(ColorOverlay0),
		chroma.Background:          "bg:" + fg(ColorSurface0),
	}))
	return name
}
