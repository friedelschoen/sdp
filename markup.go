package main

import (
	"image"
	"image/draw"
	"iter"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type MarkupAttribute int

const (
	Bold MarkupAttribute = 1 << iota
	Italic
	Underline
	Strikethrough
	Code
)

type Markup struct {
	Attr MarkupAttribute /* attributes of following text */
	Text string          /* actual content */
}

type MarkupText []Markup

// ParseMarkup parses a limited subset of Markdown into MarkupText.
//
// Supported:
//   - Code:           `code`
//   - Bold:           **text**
//   - Italic:         *text* or _text_
//   - Underline:      __text__
//   - Strikethrough:  ~~text~~
//   - Escapes:        \*, \_, \~, \`, \\, \#, etc.
//
// Nesting works for the non-code styles in a straightforward way; inline code is
// treated as opaque until the next unescaped backtick.
func ParseMarkup(content string) MarkupText {
	var out MarkupText
	var buf strings.Builder
	var state MarkupAttribute

	emit := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, Markup{
			Attr: state,
			Text: buf.String(),
		})
		buf.Reset()
	}

	for len(content) > 0 {
		// Markers—langste eerst: **, __, ~~, dan *, _
		switch {
		case state&Code == 0 && strings.HasPrefix(content, "\\**"):
			buf.WriteString("**")
			content = content[3:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "\\__"):
			buf.WriteString("__")
			content = content[3:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "\\~~"):
			buf.WriteString("~~")
			content = content[3:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "\\*"):
			buf.WriteRune('*')
			content = content[2:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "\\_"):
			buf.WriteRune('_')
			content = content[2:]
			continue
		case strings.HasPrefix(content, "\\`"):
			buf.WriteRune('`')
			content = content[2:]
			continue
		case strings.HasPrefix(content, "\\\\"):
			buf.WriteRune('\\')
			content = content[2:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "**"):
			emit()
			state ^= Bold
			content = content[2:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "__"):
			emit()
			state ^= Underline
			content = content[2:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "~~"):
			emit()
			state ^= Strikethrough
			content = content[2:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "*"):
			emit()
			state ^= Italic
			content = content[1:]
			continue
		case state&Code == 0 && strings.HasPrefix(content, "_"):
			emit()
			state ^= Italic
			content = content[1:]
			continue
		case strings.HasPrefix(content, "`"):
			emit()
			state ^= Code
			content = content[1:]
			continue
		default:
			chr, sz := utf8.DecodeRuneInString(content)
			buf.WriteRune(chr)
			content = content[sz:]
		}
	}

	emit()
	return out
}

func hasBit[T ~int](base, has T) bool {
	return base&has == has
}

func (a MarkupAttribute) font(cfg PresConfig) font.Face {
	switch {
	case hasBit(a, Code|Bold|Italic):
		return cfg.MonoFonts.BoldItalic
	case hasBit(a, Code|Bold):
		return cfg.MonoFonts.Bold
	case hasBit(a, Code|Italic):
		return cfg.MonoFonts.Italic
	case hasBit(a, Code):
		return cfg.MonoFonts.Regular
	case hasBit(a, Bold|Italic):
		return cfg.Fonts.BoldItalic
	case hasBit(a, Bold):
		return cfg.Fonts.Bold
	case hasBit(a, Italic):
		return cfg.Fonts.Italic
	default:
		return cfg.Fonts.Regular
	}
}

// measureText was misspelled as MessureText; fixed and call sites updated.
func (a MarkupAttribute) measureText(s string, cfg PresConfig) fixed.Int26_6 {
	face := a.font(cfg)

	var x fixed.Int26_6
	prevRune := rune(-1)
	for _, r := range s {
		if prevRune != -1 {
			x += face.Kern(prevRune, r)
		}
		switch r {
		case '\t':
			adv, _ := face.GlyphAdvance(' ')
			x += adv * fixed.Int26_6(cfg.TabSize)
		default:
			adv, _ := face.GlyphAdvance(r)
			x += adv
		}
		prevRune = r
	}
	return x
}

func (m MarkupText) words() iter.Seq2[MarkupAttribute, []rune] {
	return func(yield func(MarkupAttribute, []rune) bool) {
		for _, part := range m {
			var runes []rune
			wasSpace := false
			for _, r := range part.Text {
				isSpace := unicode.IsSpace(r)
				if len(runes) > 0 && wasSpace != isSpace {
					if !yield(part.Attr, runes) {
						return
					}
					runes = runes[:0]
				}
				wasSpace = isSpace
				runes = append(runes, r)
			}
			if len(runes) > 0 && !yield(part.Attr, runes) {
				return
			}
		}
	}
}

func (m MarkupText) wrapLines(bounds image.Rectangle, cfg PresConfig) iter.Seq2[fixed.Int26_6, MarkupText] {
	return func(yield func(fixed.Int26_6, MarkupText) bool) {
		var width fixed.Int26_6
		var line MarkupText
		for attr, word := range m.words() {
			if nl := slices.Index(word, '\n'); nl != -1 {
				if !yield(width, line) {
					return
				}
				line = nil
				width = 0
				word = word[nl+1:]
				if len(word) == 0 {
					continue
				}
			}
			adv := attr.measureText(string(word), cfg)
			if (width + adv).Ceil() > bounds.Dx() {
				if !yield(width, line) {
					return
				}
				line = nil
				width = 0
				if unicode.IsSpace(word[0]) {
					continue
				}
			}
			width += adv
			line = append(line, Markup{attr, string(word)})
		}
		if !yield(width, line) {
			return
		}
	}
}

func (m MarkupText) height(cfg PresConfig) (h, asc fixed.Int26_6) {
	for _, part := range m {
		face := part.Attr.font(cfg)
		h = max(h, face.Metrics().Height)
		asc = max(asc, face.Metrics().Ascent)
	}
	return
}

// Huidige runs voor lijnen
type lineRun struct {
	underline bool
	active    bool
	start     fixed.Int26_6
	face      font.Face
}

// helper om een run te sluiten en te tekenen tot currentX
func (run *lineRun) closeRun(dot fixed.Point26_6) (image.Rectangle, bool) {
	if !run.active {
		return image.Rectangle{}, false
	}
	met := run.face.Metrics()
	// dynamische dikte: ~5% van font height, min 1px
	thick := max(met.Height.Ceil()/20, 1)
	var y fixed.Int26_6
	if run.underline {
		// iets onder de baseline
		y = dot.Y + fixed.I(thick)
	} else {
		// strikethrough ongeveer halverwege de x-height (≈ helft van ascent)
		y = dot.Y - met.Ascent/2
	}
	run.active = false
	if dot.X <= run.start {
		return image.Rectangle{}, false
	}
	// naar device pixels, inclusief bounds offset
	x0px := run.start.Ceil()
	x1px := dot.X.Ceil()
	ypx := y.Ceil()
	if y < 1 {
		y = 1
	}
	return image.Rect(x0px, ypx, x1px, ypx+thick), true
}

func (m MarkupText) String() string {
	var buf strings.Builder
	for _, parts := range m {
		buf.WriteString(parts.Text)
	}
	return buf.String()
}

func (m MarkupText) Draw(img draw.Image, bounds image.Rectangle, cfg PresConfig) {
	draw.Draw(img, bounds, cfg.Background, image.Point{}, draw.Src)
	bounds = cfg.Margin.Apply(bounds)

	var totalHeight fixed.Int26_6
	for _, text := range m.wrapLines(bounds, cfg) {
		h, _ := text.height(cfg)
		totalHeight += h
	}

	var dot fixed.Point26_6
	var yOffset fixed.Int26_6

	switch cfg.VAlign {
	case Top:
		yOffset = 0
	case Middle:
		yOffset = fixed.I(bounds.Dy()/2) - totalHeight/2
	case Bottom:
		yOffset = fixed.I(bounds.Dy()) - totalHeight
	}

	for width, text := range m.wrapLines(bounds, cfg) {
		h, asc := text.height(cfg)

		switch cfg.Align {
		case Left:
			dot.X = 0
		case Center:
			dot.X = fixed.I(bounds.Dx()/2) - width/2
		case Right:
			dot.X = fixed.I(bounds.Dx()) - width
		}
		dot.Y = yOffset + asc

		prevRune := rune(-1)

		ul := lineRun{underline: true}  // underline-run
		st := lineRun{underline: false} // strikethrough-run

		for _, part := range text {
			face := part.Attr.font(cfg)

			// start/stop runs op stijlwissel per part
			hasUL := part.Attr&Underline != 0
			hasST := part.Attr&Strikethrough != 0

			// start underline-run als nodig
			if hasUL && !ul.active {
				ul.active = true
				ul.start = dot.X
				ul.face = face
			}
			// sluit underline-run als stijl wegvalt
			if !hasUL && ul.active {
				line, ok := ul.closeRun(dot)
				if ok {
					line = line.Add(bounds.Min)
					draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
				}
			}

			// start strikethrough-run als nodig
			if hasST && !st.active {
				st.active = true
				st.start = dot.X
				st.face = face
			}
			// sluit strikethrough-run als stijl wegvalt
			if !hasST && st.active {
				line, ok := st.closeRun(dot)
				if ok {
					line = line.Add(bounds.Min)
					draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
				}
			}

			for _, r := range part.Text {
				if r == '\n' {
					// sluit lopende runs tot nu toe en ga naar volgende visuele regel
					if ul.active {
						line, ok := ul.closeRun(dot)
						if ok {
							line = line.Add(bounds.Min)
							draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
						}
					}
					if st.active {
						line, ok := st.closeRun(dot)
						if ok {
							line = line.Add(bounds.Min)
							draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
						}
					}
					yOffset += h
					dot.X = 0
					dot.Y = yOffset + asc
					prevRune = -1
					continue
				}
				if prevRune != -1 {
					dot.X += face.Kern(prevRune, r)
				}

				switch r {
				case '\t':
					advSpace, _ := face.GlyphAdvance(' ')
					dot.X += advSpace * fixed.Int26_6(cfg.TabSize)
				default:
					dr, mask, maskp, advance, _ := face.Glyph(dot, r)
					dr = dr.Add(bounds.Min)
					draw.DrawMask(img, dr, cfg.Foreground, image.Point{}, mask, maskp, draw.Over)
					dot.X += advance
				}
				prevRune = r
			}
		}

		// Einde van de visuele regel: open runs sluiten
		if ul.active {
			line, ok := ul.closeRun(dot)
			if ok {
				line = line.Add(bounds.Min)
				draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
			}
		}
		if st.active {
			line, ok := st.closeRun(dot)
			if ok {
				line = line.Add(bounds.Min)
				draw.Draw(img, line, cfg.Foreground, image.Point{}, draw.Src)
			}
		}

		yOffset += h
	}
}
