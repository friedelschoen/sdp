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
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type MarkupAttribute int

const (
	Bold MarkupAttribute = 1 << iota
	Italic
	Underline
	Strikethrough
	Code
	BigText
)

type Markup struct {
	Attr MarkupAttribute /* attributes of following text */
	Text string          /* actual content */
}

type MarkupText []Markup

// ParseMarkup parses a limited subset of Markdown into MarkupText.
//
// Supported:
//   - Big:			   ==text==
//   - Code:           `code`
//   - Bold:           **text**
//   - Italic:         *text* or _text_
//   - Underline:      __text__
//   - Strikethrough:  ~~text~~
type MarkupBuilder struct {
	out   MarkupText
	buf   []rune
	state MarkupAttribute
}

func (b *MarkupBuilder) emit() {
	if len(b.buf) == 0 {
		return
	}
	b.out = append(b.out, Markup{
		Attr: b.state,
		Text: string(b.buf),
	})
	b.buf = b.buf[:0]
}

func (b *MarkupBuilder) Feed(content string) {
	if !b.Dirty() {
		content = strings.TrimLeft(content, "\n")
	}
	for len(content) > 0 {
		// Markers—langste eerst: **, __, ~~, dan *, _
		switch {
		case b.state&Code == 0 && strings.HasPrefix(content, "\\**"):
			b.buf = append(b.buf, '*', '*')
			content = content[3:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "\\__"):
			b.buf = append(b.buf, '_', '_')
			content = content[3:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "\\~~"):
			b.buf = append(b.buf, '~', '~')
			content = content[3:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "\\=="):
			b.buf = append(b.buf, '=', '=')
			content = content[3:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "\\*"):
			b.buf = append(b.buf, '*')
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "\\_"):
			b.buf = append(b.buf, '_')
			content = content[2:]
			continue
		case strings.HasPrefix(content, "\\`"):
			b.buf = append(b.buf, '`')
			content = content[2:]
			continue
		case strings.HasPrefix(content, "\\\\"):
			b.buf = append(b.buf, '\\')
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "**"):
			b.emit()
			b.state ^= Bold
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "__"):
			b.emit()
			b.state ^= Underline
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "~~"):
			b.emit()
			b.state ^= Strikethrough
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "=="):
			b.emit()
			b.state ^= BigText
			content = content[2:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "*"):
			b.emit()
			b.state ^= Italic
			content = content[1:]
			continue
		case b.state&Code == 0 && strings.HasPrefix(content, "_"):
			b.emit()
			b.state ^= Italic
			content = content[1:]
			continue
		case strings.HasPrefix(content, "`"):
			b.emit()
			b.state ^= Code
			content = content[1:]
			continue
		default:
			chr, sz := utf8.DecodeRuneInString(content)
			b.buf = append(b.buf, chr)
			content = content[sz:]
		}
	}
	b.emit()
}

func (b *MarkupBuilder) Text() MarkupText {
	b.emit() /* flush all contents */

	return b.out
}

func (b *MarkupBuilder) Dirty() bool {
	return len(b.out) != 0 || len(b.buf) != 0 || b.state != 0
}

func (b *MarkupBuilder) Reset() {
	b.out = nil
	b.buf = nil
	b.state = 0
}

func hasBit[T ~int](base, has T) bool {
	return base&has == has
}

func (a MarkupAttribute) font(cfg PresConfig) *opentype.Font {
	switch {
	case hasBit(a, Code|Bold|Italic):
		if cfg.MonoFonts.BoldItalic != nil {
			return cfg.MonoFonts.BoldItalic
		}
		fallthrough
	case hasBit(a, Code|Bold):
		if cfg.MonoFonts.Bold != nil {
			return cfg.MonoFonts.Bold
		}
		fallthrough
	case hasBit(a, Code|Italic):
		if cfg.MonoFonts.Italic != nil {
			return cfg.MonoFonts.Italic
		}
		fallthrough
	case hasBit(a, Code):
		if cfg.MonoFonts.Regular != nil {
			return cfg.MonoFonts.Regular
		}
		fallthrough
	case hasBit(a, Bold|Italic):
		if cfg.Fonts.BoldItalic != nil {
			return cfg.Fonts.BoldItalic
		}
		fallthrough
	case hasBit(a, Bold):
		if cfg.Fonts.Bold != nil {
			return cfg.Fonts.Bold
		}
		fallthrough
	case hasBit(a, Italic):
		if cfg.Fonts.Italic != nil {
			return cfg.Fonts.Italic
		}
		fallthrough
	default:
		return cfg.Fonts.Regular
	}
}

func (a MarkupAttribute) face(size float64, cfg PresConfig) font.Face {
	font := a.font(cfg)

	if hasBit(a, BigText) {
		size *= cfg.BigText
	}
	face, _ := opentype.NewFace(font, &opentype.FaceOptions{DPI: 72, Size: size})
	return face
}

// measureText was misspelled as MessureText; fixed and call sites updated.
func (a MarkupAttribute) measureText(s string, size float64, cfg PresConfig) fixed.Int26_6 {
	var x fixed.Int26_6
	face := a.face(size, cfg)
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
			if part.Attr&(Code|BigText) != 0 {
				/* do not split code-sections when code-section of bigtext-section */
				if !yield(part.Attr, []rune(part.Text)) {
					return
				}
				continue
			}

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

func (m MarkupText) wrapLines(bounds image.Rectangle, size float64, cfg PresConfig) iter.Seq2[fixed.Int26_6, MarkupText] {
	return func(yield func(fixed.Int26_6, MarkupText) bool) {
		var width fixed.Int26_6
		var line MarkupText
		for attr, word := range m.words() {
			if nl := slices.Index(word, '\n'); nl != -1 {
				if !yield(width, line) {
					return
				}
				if !yield(0, nil) {
					return
				}
				line = nil
				width = 0
				word = word[nl+1:]
				if len(word) == 0 {
					continue
				}
			}
			adv := attr.measureText(string(word), size, cfg)
			if (width + adv).Ceil() > bounds.Dx() {
				if width == 0 {
					/* only one word already exceeds the line */
					yield(-1, nil)
					return
				}
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

func (m MarkupText) height(size float64, cfg PresConfig) (h, asc fixed.Int26_6) {
	for _, part := range m {
		face := part.Attr.face(size, cfg)
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
		y = dot.Y - met.Ascent/3
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

func (m MarkupText) totalHeight(bounds image.Rectangle, size float64, cfg PresConfig) (totalHeight fixed.Int26_6, ok bool) {
	ok = true
	for w, text := range m.wrapLines(bounds, size, cfg) {
		if w == -1 {
			ok = false
			return
		}
		if text == nil {
			totalHeight += fixed.I(int(size * cfg.NewlineSpacing))
			continue
		}
		h, _ := text.height(size, cfg)
		totalHeight += h
	}
	return
}

func (m MarkupText) Draw(img draw.Image, bounds image.Rectangle, cfg PresConfig) {
	bounds = cfg.Margin.Apply(bounds)

	var totalHeight fixed.Int26_6
	var size float64
	for i := float64(1); i < 100; i += 1 {
		h, ok := m.totalHeight(bounds, i, cfg)
		if !ok || h.Ceil() >= bounds.Dy() {
			break
		}

		totalHeight = h
		size = i
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

	for width, text := range m.wrapLines(bounds, size, cfg) {
		if text == nil {
			yOffset += fixed.I(int(size * cfg.NewlineSpacing))
			continue
		}
		h, asc := text.height(size, cfg)

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
			face := part.Attr.face(size, cfg)

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
