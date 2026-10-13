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
	s.BlockQuote.Color = hex(readable(ColorSubtext0, ColorBase))
	s.Heading.Color = hex(readable(ColorBlue, ColorBase))
	h1fg, h1bg := h1Colors()
	s.H1.Color, s.H1.BackgroundColor = hex(h1fg), hex(h1bg)
	s.H2.Color = hex(readable(ColorBlue, ColorBase))
	s.H3.Color = hex(readable(ColorMauve, ColorBase))
	s.H4.Color = hex(readable(ColorTeal, ColorBase))
	s.H5.Color = hex(readable(ColorGreen, ColorBase))
	s.H6.Color = hex(readable(ColorSubtext0, ColorBase))
	s.HorizontalRule.Color = hex(readable(ColorMuted, ColorBase))
	s.Link.Color = hex(readable(ColorBlue, ColorBase))
	s.LinkText.Color = hex(readable(ColorTeal, ColorBase))
	s.Image.Color = hex(readable(ColorMauve, ColorBase))
	s.ImageText.Color = hex(readable(ColorMuted, ColorBase))
	s.Code.Color, s.Code.BackgroundColor = hex(readable(ColorPeach, ColorSurface0)), hex(ColorSurface0)
	s.Strong.Color = hex(ColorText)
	s.Emph.Color = hex(ColorText)
	s.Strikethrough.Color = hex(readable(ColorMuted, ColorBase))
	s.Task.Color = hex(readable(ColorGreen, ColorBase))
	s.Item.Color = hex(readable(ColorPeach, ColorBase))
	s.Enumeration.Color = hex(readable(ColorPeach, ColorBase))
	s.DefinitionTerm.Color = hex(readable(ColorBlue, ColorBase))

	// Los bloques de código usan un estilo de Chroma propio por tema. Con Chroma
	// dentro del estilo de Glamour, este lo registra una sola vez con el nombre "charm" y
	// ya no lo actualiza: el código se quedaría con los colores del primer tema.
	s.CodeBlock.Color = hex(readable(ColorSubtext0, ColorSurface0))
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
		chroma.Text:                fg(readable(ColorText, ColorSurface0)),
		chroma.Error:               fg(ColorBase) + " bg:" + fg(ColorRed),
		chroma.Comment:             fg(readable(ColorMuted, ColorSurface0)),
		chroma.CommentPreproc:      fg(readable(ColorPeach, ColorSurface0)),
		chroma.Keyword:             fg(readable(ColorBlue, ColorSurface0)),
		chroma.KeywordReserved:     fg(readable(ColorMauve, ColorSurface0)),
		chroma.KeywordNamespace:    fg(readable(ColorRed, ColorSurface0)),
		chroma.KeywordType:         fg(readable(ColorTeal, ColorSurface0)),
		chroma.Operator:            fg(readable(ColorRed, ColorSurface0)),
		chroma.Punctuation:         fg(readable(ColorSubtext0, ColorSurface0)),
		chroma.Name:                fg(readable(ColorText, ColorSurface0)),
		chroma.NameBuiltin:         fg(readable(ColorMauve, ColorSurface0)),
		chroma.NameTag:             fg(readable(ColorMauve, ColorSurface0)),
		chroma.NameAttribute:       fg(readable(ColorBlue, ColorSurface0)),
		chroma.NameClass:           fg(readable(ColorYellow, ColorSurface0)) + " bold underline",
		chroma.NameConstant:        fg(readable(ColorPeach, ColorSurface0)),
		chroma.NameDecorator:       fg(readable(ColorYellow, ColorSurface0)),
		chroma.NameException:       fg(readable(ColorRed, ColorSurface0)),
		chroma.NameFunction:        fg(readable(ColorGreen, ColorSurface0)),
		chroma.NameOther:           fg(readable(ColorText, ColorSurface0)),
		chroma.Literal:             fg(readable(ColorPeach, ColorSurface0)),
		chroma.LiteralNumber:       fg(readable(ColorTeal, ColorSurface0)),
		chroma.LiteralDate:         fg(readable(ColorTeal, ColorSurface0)),
		chroma.LiteralString:       fg(readable(ColorGreen, ColorSurface0)),
		chroma.LiteralStringEscape: fg(readable(ColorTeal, ColorSurface0)),
		chroma.GenericDeleted:      fg(readable(ColorRed, ColorSurface0)),
		chroma.GenericEmph:         "italic",
		chroma.GenericInserted:     fg(readable(ColorGreen, ColorSurface0)),
		chroma.GenericStrong:       "bold",
		chroma.GenericSubheading:   fg(readable(ColorMuted, ColorSurface0)),
		chroma.Background:          "bg:" + fg(ColorSurface0),
	}))
	return name
}
