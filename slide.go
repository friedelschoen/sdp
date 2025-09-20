package sdp

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	xdraw "golang.org/x/image/draw"
)

var formats = []struct {
	Decode   func(io.Reader) (image.Image, error)
	Offset   int
	Patterns [][]int
}{
	{png.Decode, 0, [][]int{{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}}},
	{jpeg.Decode, 0, [][]int{
		{0xFF, 0xD8, 0xFF, 0xDB},
		{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
		{0xFF, 0xD8, 0xFF, 0xEE},
		{0xFF, 0xD8, 0xFF, 0xE1, 0x100, 0x100, 0x45, 0x78, 0x69, 0x66, 0x00, 0x00},
		{0xFF, 0xD8, 0xFF, 0xE0},
	}},
	{gif.Decode, 0, [][]int{
		{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},
		{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
	}},
}

func decoderImage(content []byte) func(io.Reader) (image.Image, error) {
	for _, form := range formats {
		for _, pat := range form.Patterns {
			if form.Offset+len(pat) > len(content) {
				continue
			}
			off := form.Offset
			found := true
			for i, chr := range pat {
				if chr <= 0xff && byte(chr) != content[off+i] {
					found = false
					break
				}
			}
			if found {
				return form.Decode
			}
		}
	}
	return nil
}

type ImageSlide struct {
	src image.Image
}

func NewImageSlide(pat string) (*ImageSlide, error) {
	content, err := os.ReadFile(pat)
	if err != nil {
		return nil, err
	}
	decoder := decoderImage(content)
	if decoder == nil {
		return nil, fmt.Errorf("invalid image-format of %s", pat)
	}
	img, err := decoder(bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}
	return &ImageSlide{src: img}, nil
}

// positionImage inside W×H (contain). Never exceed the box.
func positionImage(src image.Rectangle, box image.Rectangle, align Alignment, valign VerticalAlignment) image.Rectangle {
	if src.Empty() || box.Empty() {
		return image.Rectangle{}
	}
	bw, bh := box.Dx(), box.Dy()
	sw, sh := float64(src.Dx()), float64(src.Dy())

	/* factor */
	s := min(float64(bw)/sw, float64(bh)/sh)

	/* new width&height, capped to actual width and height */
	w := min(int(sw*s), bw)
	h := min(int(sh*s), bh)

	var x, y int
	switch align {
	case Left:
		x = 0
	case Center:
		if w < bw {
			x = bw/2 - w/2
		}
		if h < bh {
			y = bh/2 - h/2
		}
	case Right:
		if w < bw {
			x = bw - w
		}
	}
	switch valign {
	case Top:
		y = 0
	case Middle:
		if h < bh {
			y = bh/2 - h/2
		}
	case Bottom:
		if y < bh {
			y = bh - h
		}
	}

	return image.Rectangle{box.Min.Add(image.Point{x, y}), box.Min.Add(image.Point{x + w, y + h})}
}

func (s *ImageSlide) Draw(img draw.Image, bounds image.Rectangle, attr PresConfig) {
	bounds = attr.Margin.Apply(bounds)
	imgr := positionImage(s.src.Bounds(), bounds, attr.Align, attr.VAlign)
	xdraw.BiLinear.Scale(img, imgr, s.src, s.src.Bounds(), draw.Over, nil)
}

func FinalSlide(cfg PresConfig) Slide {
	cfg.Background = image.NewUniform(color.Gray{50})
	cfg.Foreground = image.NewUniform(color.Gray{200})
	cfg.FontSize = 3
	cfg.VAlign = Top

	return Slide{cfg, "", []SlideContent{
		MarkupText{
			Markup{
				Attr: Bold,
				Text: "End of Presentation",
			},
		},
	}}
}
