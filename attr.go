package sdp

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

type Margins struct{ Left, Right, Top, Bottom float64 }

/* Apply applies the margin-boundaries to `r` and returns a copy */
func (m Margins) Apply(r image.Rectangle) image.Rectangle {
	w, h := r.Dx(), r.Dy()
	r.Min.X += int(float64(w) * m.Left)
	r.Min.Y += int(float64(h) * m.Top)
	r.Max.X -= int(float64(w) * m.Right)
	r.Max.Y -= int(float64(h) * m.Bottom)
	return r
}

type Alignment int

const (
	Left Alignment = iota
	Center
	Right
)

type VerticalAlignment int

const (
	Top VerticalAlignment = iota
	Middle
	Bottom
)

type FontCollection struct {
	Regular    *opentype.Font
	Bold       *opentype.Font
	Italic     *opentype.Font
	BoldItalic *opentype.Font
}

type PresConfig struct {
	Foreground     image.Image /* uniform */
	Background     image.Image /* uniform */
	Fonts          FontCollection
	MonoFonts      FontCollection
	Margin         Margins
	Align          Alignment
	VAlign         VerticalAlignment
	TabSize        int
	NewlineSpacing float64
	BigText        float64
	FontSize       float64 /* percent of diagonal px */
}

func (c *PresConfig) AddAttribute(str string) error {
	key, value, hasValue := strings.Cut(str, "=")
	switch key {
	case "foreground", "fg":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		color, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("error in `%s`: %w", value, err)
		}
		c.Foreground = image.NewUniform(color)
	case "background", "bg":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		color, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("error in `%s`: %w", value, err)
		}
		c.Background = image.NewUniform(color)
	case "left":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "%")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Left = float64(px) / 100
	case "right":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "%")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Right = float64(px) / 100
	case "top":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "%")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Top = float64(px) / 100
	case "bottom":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "%")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Bottom = float64(px) / 100
	case "margin":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		first, second, hasSecond := strings.Cut(value, " ")

		first = strings.TrimSuffix(first, "%")
		firstPx, err := strconv.Atoi(first)
		if err != nil {
			return err
		}
		firstPxf := float64(firstPx) / 100
		if !hasSecond {
			c.Margin.Left = firstPxf
			c.Margin.Right = firstPxf
			c.Margin.Top = firstPxf
			c.Margin.Bottom = firstPxf
		} else {
			second = strings.TrimSuffix(second, "%")
			secondPx, err := strconv.Atoi(second)
			if err != nil {
				return err
			}
			secondPxf := float64(secondPx) / 100
			c.Margin.Left = secondPxf
			c.Margin.Right = secondPxf
			c.Margin.Top = firstPxf
			c.Margin.Bottom = firstPxf
		}
	case "align":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		switch value {
		case "left":
			c.Align = Left
		case "center", "middle":
			c.Align = Center
		case "right":
			c.Align = Right
		default:
			return fmt.Errorf("invalid alignment `%s`", value)
		}
	case "valign":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		switch value {
		case "top":
			c.VAlign = Top
		case "center", "middle":
			c.VAlign = Middle
		case "right":
			c.VAlign = Bottom
		default:
			return fmt.Errorf("invalid alignment `%s`", value)
		}
	case "tabsize":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		times, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.TabSize = times
	case "newline-spacing":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		times, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		c.NewlineSpacing = times
	case "bigtext":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		times, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		c.BigText = times
	default:
		return fmt.Errorf("invalid attribute `%s`", key)
	}
	return nil
}

func defaultConf() PresConfig {
	makeFace := func(data []byte) *opentype.Font {
		font, err := opentype.Parse(data)
		if err != nil {
			panic(fmt.Errorf("unable to parse Go-font: %w", err))
		}
		return font
	}
	return PresConfig{
		Foreground: image.Black,
		Background: image.White,
		Fonts: FontCollection{
			Regular:    makeFace(goregular.TTF),
			Bold:       makeFace(gobold.TTF),
			Italic:     makeFace(goitalic.TTF),
			BoldItalic: makeFace(gobolditalic.TTF),
		},
		MonoFonts: FontCollection{
			Regular:    makeFace(gomono.TTF),
			Bold:       makeFace(gomonobold.TTF),
			Italic:     makeFace(gomonoitalic.TTF),
			BoldItalic: makeFace(gomonobolditalic.TTF),
		},
		Margin:         Margins{0.1, 0.1, 0.1, 0.1},
		Align:          Center,
		VAlign:         Middle,
		TabSize:        4,
		NewlineSpacing: 1,
		BigText:        1.2,
	}
}
