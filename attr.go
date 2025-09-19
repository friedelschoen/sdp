package main

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"golang.org/x/image/font"
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

type Margins struct{ Left, Right, Top, Bottom int }

/* Apply applies the margin-boundaries to `r` and returns a copy */
func (m Margins) Apply(r image.Rectangle) image.Rectangle {
	r.Min.X += m.Left
	r.Min.Y += m.Top
	r.Max.X -= m.Right
	r.Max.Y -= m.Bottom
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
	Regular    font.Face
	Bold       font.Face
	Italic     font.Face
	BoldItalic font.Face
}

type PresConfig struct {
	Foreground image.Image /* uniform */
	Background image.Image /* uniform */
	Fonts      FontCollection
	MonoFonts  FontCollection
	Margin     Margins
	Align      Alignment
	VAlign     VerticalAlignment
	TabSize    int
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
		value = strings.TrimSuffix(value, "px")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Left = px
	case "right":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "px")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Right = px
	case "top":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "px")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Top = px
	case "bottom":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		value = strings.TrimSuffix(value, "px")
		px, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Margin.Bottom = px
	case "margin":
		if !hasValue {
			return fmt.Errorf("`%s` requires a value", key)
		}
		first, second, hasSecond := strings.Cut(value, " ")

		first = strings.TrimSuffix(first, "px")
		firstPx, err := strconv.Atoi(first)
		if err != nil {
			return err
		}
		if !hasSecond {
			c.Margin.Left = firstPx
			c.Margin.Right = firstPx
			c.Margin.Top = firstPx
			c.Margin.Bottom = firstPx
		} else {
			second = strings.TrimSuffix(second, "px")
			secondPx, err := strconv.Atoi(second)
			if err != nil {
				return err
			}
			c.Margin.Left = secondPx
			c.Margin.Right = secondPx
			c.Margin.Top = firstPx
			c.Margin.Bottom = firstPx
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
	default:
		return fmt.Errorf("invalid attribute `%s`", key)
	}
	return nil
}

func defaultConf() PresConfig {
	makeFace := func(data []byte) font.Face {
		font, err := opentype.Parse(data)
		if err != nil {
			panic(fmt.Errorf("unable to parse Go-font: %w", err))
		}
		face, err := opentype.NewFace(font, &opentype.FaceOptions{
			DPI:  72,
			Size: 15,
		})
		if err != nil {
			panic(fmt.Errorf("unable to parse Go-font: %w", err))
		}
		return face
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
		Margin: Margins{10, 10, 10, 10},
		Align:  Center,
		VAlign: Middle,
	}
}
