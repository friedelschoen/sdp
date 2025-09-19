package main

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"strings"
	"unicode"
)

type Presentation struct {
	Conf   PresConfig
	Slides []Slide
}

type Slide struct {
	Conf    PresConfig
	Content []SlideContent
}

func (s *Slide) Draw(img draw.Image, bounds image.Rectangle) {
	dw := img.Bounds().Dx() / len(s.Content)
	for i, cnt := range s.Content {
		cnt.Draw(img, image.Rect(bounds.Min.X+dw*i, bounds.Min.Y, bounds.Min.X+dw*(i+1), bounds.Max.Y), s.Conf)
	}
}

type SlideContent interface {
	Draw(img draw.Image, bounds image.Rectangle, attr PresConfig)
}

func ParsePresentation(r io.Reader) (*Presentation, error) {
	scanner := bufio.NewScanner(r)
	var pres Presentation
	var markup MarkupBuilder

	var slides []SlideContent

	var globalconf = defaultConf()
	var slideconf = globalconf

	for scanner.Scan() {
		line := scanner.Text()
		/* strip trailin whitespaces */
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		switch {
		case line == "":
			markup.Feed("\n")
		case line[0] == '#':
			/* ignore line -> comment */
			continue
		case strings.HasPrefix(line, "%set "):
			line = strings.TrimLeftFunc(line[4:], unicode.IsSpace)
			globalconf.AddAttribute(line)
			if markup.Dirty() {
				fmt.Fprintf(os.Stderr, "option not at beginning of slide\n")
			}
		case strings.HasPrefix(line, "%"):
			line = strings.TrimLeftFunc(line[1:], unicode.IsSpace)
			slideconf.AddAttribute(line)
			if markup.Dirty() {
				fmt.Fprintf(os.Stderr, "option not at beginning of slide\n")
			}
		case line[0] == '@':
			if markup.Dirty() {
				slides = append(slides, markup.Text())
				markup.Reset()
			}
			path := line[1:]
			slide, err := NewImageSlide(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERR: %v\n", err)
				os.Exit(1)
			}
			slides = append(slides, slide)
		case line == "%%%":
			if markup.Dirty() {
				slides = append(slides, markup.Text())
				markup.Reset()
			}
		case line == "---":
			if markup.Dirty() {
				slides = append(slides, markup.Text())
				markup.Reset()
			}
			pres.Slides = append(pres.Slides, Slide{slideconf, slides})
			slides = nil
			slideconf = globalconf
		default:
			markup.Feed(line)
		}
	}
	if markup.Dirty() {
		slides = append(slides, markup.Text())
		markup.Reset()
	}
	pres.Slides = append(pres.Slides, Slide{slideconf, slides})

	return &pres, scanner.Err()
}
