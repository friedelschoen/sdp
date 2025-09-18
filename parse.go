package main

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"io"
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

type Presentation struct {
	Conf   PresConfig
	Slides []Slide
}

type Slide struct {
	Conf    PresConfig
	Content SlideContent
}

func (s *Slide) Draw(img draw.Image, bounds image.Rectangle) {
	s.Content.Draw(img, bounds, s.Conf)
}

type SlideContent interface {
	Draw(img draw.Image, bounds image.Rectangle, attr PresConfig)
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

func parseContent(str string) (SlideContent, error) {
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return nil, nil
	}
	if trimmed == "---" {
		return EmptySlide{}, nil
	}
	if trimmed[0] == '@' {
		return NewImageSlide(trimmed[1:])
	}
	return ParseMarkup(str), nil
}

func ParsePresentation(r io.Reader) (*Presentation, error) {
	scan := bufio.NewScanner(r)
	var pres Presentation
	var content strings.Builder
	globalconf := defaultConf()
	slideconf := globalconf
	for scan.Scan() {
		line := scan.Text()
		if idx := strings.IndexRune(line, '#'); idx != -1 {
			line = line[:idx]
		}
		if strings.TrimSpace(line) == "" {
			/* make new slide */
			drawable, err := parseContent(content.String())
			if err != nil {
				return nil, fmt.Errorf("unable to parse content: %w", err)
			}
			if drawable != nil {
				pres.Slides = append(pres.Slides, Slide{slideconf, drawable})
			}

			/* reset state */
			content.Reset()
			slideconf = globalconf
		}
		if strings.HasPrefix(line, "global!") {
			line = line[1:]
			globalconf.AddAttribute(line)
			continue
		}
		if strings.HasPrefix(line, "!") {
			line = line[1:]
			slideconf.AddAttribute(line)
			continue
		}
		content.WriteString(line)
		content.WriteRune('\n')
	}
	drawable, err := parseContent(content.String())
	if err != nil {
		return nil, fmt.Errorf("unable to parse content: %w", err)
	}
	if drawable != nil {
		pres.Slides = append(pres.Slides, Slide{slideconf, drawable})
	}

	pres.Conf = globalconf
	return &pres, scan.Err()
}
