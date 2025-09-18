package main

import (
	"image"
	"image/draw"
	"iter"
	"slices"
	"strings"
	"unicode"

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
//   - Code blocks:    ```lang\n...\n```  -> Code with CodeSyntax="lang"
//   - Inline code:    `code`              -> Code
//   - Bold:           **text**
//   - Italic:         *text* or _text_
//   - Underline:      __text__            (non-standard, chosen to avoid ** bold conflict)
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

	isLineStart := true
	i := 0
	n := len(content)

	readWhile := func(pred func(byte) bool) string {
		j := i
		for j < n && pred(content[j]) {
			j++
		}
		s := content[i:j]
		i = j
		return s
	}

	for i < n {
		// Fenced code block alleen op line start: ```lang\n ... \n```
		if isLineStart && strings.HasPrefix(content[i:], "```") {
			emit()
			i += 3
			// optionele newline na opening fence
			if i < n && (content[i] == '\r' || content[i] == '\n') {
				i++
				if i < n && content[i-1] == '\r' && content[i] == '\n' {
					i++
				}
			}

			start := i
			for i < n {
				if content[i] == '\n' {
					j := i + 1
					if strings.HasPrefix(content[j:], "```") {
						// inhoud tot net voor deze '\n'
						block := content[start:i]
						// tijdelijk naar Code‐state schakelen
						prev := state
						state = Code
						buf.WriteString(block)
						emit()
						state = prev

						// consume sluitende ``` en rest van de regel
						i = j + 3
						_ = readWhile(func(b byte) bool { return b != '\n' && b != '\r' })
						if i < n && (content[i] == '\r' || content[i] == '\n') {
							i++
							if i < n && content[i-1] == '\r' && content[i] == '\n' {
								i++
							}
						}
						isLineStart = true
						goto continueOuter
					}
				}
				i++
			}

			// EOF zonder sluitende fence: rest is code
			block := content[start:n]
			prev := state
			state = Code
			buf.WriteString(block)
			emit()
			state = prev
			break
		}

		// Escapes
		if content[i] == '\\' && i+1 < n {
			i++
			buf.WriteByte(content[i])
			i++
			isLineStart = content[i-1] == '\n'
			continue
		}

		// Inline code tot volgende *ongeslashede* backtick
		if content[i] == '`' {
			emit()
			i++
			start := i
			for i < n {
				if content[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if content[i] == '`' {
					break
				}
				i++
			}
			codeTxt := content[start:i]
			prev := state
			state = Code
			buf.WriteString(codeTxt)
			emit()
			state = prev
			if i < n && content[i] == '`' {
				i++
			}
			isLineStart = i > 0 && content[i-1] == '\n'
			continue
		}

		// Markers—langste eerst: **, __, ~~, dan *, _
		switch {
		case strings.HasPrefix(content[i:], "**"):
			emit()
			state ^= Bold
			i += 2
			continue
		case strings.HasPrefix(content[i:], "__"):
			emit()
			state ^= Underline
			i += 2
			continue
		case strings.HasPrefix(content[i:], "~~"):
			emit()
			state ^= Strikethrough
			i += 2
			continue
		case content[i] == '*':
			emit()
			state ^= Italic
			i++
			continue
		case content[i] == '_':
			emit()
			// enkele underscore → italic (zoals gebruikelijk)
			state ^= Italic
			i++
			continue
		default:
			ch := content[i]
			buf.WriteByte(ch)
			isLineStart = ch == '\n'
			i++
		}
	continueOuter:
	}

	emit()
	return out
}

func (a MarkupAttribute) font(cfg PresConfig) font.Face {
	switch {
	case a&(Code|Bold|Italic) == Code|Bold|Italic:
		return cfg.MonoFonts.BoldItalic
	case a&(Code|Bold) == Code|Bold:
		return cfg.MonoFonts.Bold
	case a&(Code|Italic) == Code|Italic:
		return cfg.MonoFonts.Italic
	case a&Code == Code:
		return cfg.MonoFonts.Regular
	case a&(Bold|Italic) == Bold|Italic:
		return cfg.Fonts.BoldItalic
	case a&Bold == Bold:
		return cfg.Fonts.Bold
	case a&Italic == Italic:
		return cfg.Fonts.Italic
	default:
		return cfg.Fonts.Regular
	}
}

// MeasureText was misspelled as MessureText; fixed and call sites updated.
func (a MarkupAttribute) MeasureText(s string, cfg PresConfig) fixed.Int26_6 {
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

func (m MarkupText) Words() iter.Seq2[MarkupAttribute, []rune] {
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
		for attr, word := range m.Words() {
			if nl := slices.Index(word, '\n'); nl != -1 {
				if !yield(width, line) {
					return
				}
				line = nil
				width = 0
				word = word[nl:]
				if len(word) == 0 {
					continue
				}
			}
			adv := attr.MeasureText(string(word), cfg)
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

	// hulpfunctie om een horizontale lijn te tekenen (x0..x1 op y), dikte t in px
	drawHLine := func(x0, x1 fixed.Int26_6, y fixed.Int26_6, t int) {
		if x1 <= x0 {
			return
		}
		// naar device pixels, inclusief bounds offset
		x0px := bounds.Min.X + x0.Ceil()
		x1px := bounds.Min.X + x1.Ceil()
		ypx := bounds.Min.Y + y.Ceil()
		if t < 1 {
			t = 1
		}
		r := image.Rect(x0px, ypx, x1px, ypx+t)
		draw.Draw(img, r, cfg.Foreground, image.Point{}, draw.Over)
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

		// Huidige runs voor lijnen
		type lineRun struct {
			active bool
			start  fixed.Int26_6
			face   font.Face
		}
		ul := lineRun{} // underline-run
		st := lineRun{} // strikethrough-run

		// helper om een run te sluiten en te tekenen tot currentX
		closeRun := func(run *lineRun, baseline fixed.Int26_6, currentX fixed.Int26_6, isUnderline bool) {
			if !run.active {
				return
			}
			met := run.face.Metrics()
			// dynamische dikte: ~5% van font height, min 1px
			thick := max(met.Height.Ceil()/20, 1)
			var y fixed.Int26_6
			if isUnderline {
				// iets onder de baseline
				y = baseline + fixed.I(thick)
			} else {
				// strikethrough ongeveer halverwege de x-height (≈ helft van ascent)
				y = baseline - met.Ascent/2
			}
			drawHLine(run.start, currentX, y, thick)
			run.active = false
		}

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
				closeRun(&ul, dot.Y, dot.X, true)
			}

			// start strikethrough-run als nodig
			if hasST && !st.active {
				st.active = true
				st.start = dot.X
				st.face = face
			}
			// sluit strikethrough-run als stijl wegvalt
			if !hasST && st.active {
				closeRun(&st, dot.Y, dot.X, false)
			}

			for _, r := range part.Text {
				if r == '\n' {
					// sluit lopende runs tot nu toe en ga naar volgende visuele regel
					if ul.active {
						closeRun(&ul, dot.Y, dot.X, true)
					}
					if st.active {
						closeRun(&st, dot.Y, dot.X, false)
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
			closeRun(&ul, dot.Y, dot.X, true)
		}
		if st.active {
			closeRun(&st, dot.Y, dot.X, false)
		}

		yOffset += h
	}
}
