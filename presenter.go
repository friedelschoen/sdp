package sdp

import (
	"image"
	"image/color"
	"image/draw"
)

func DrawPresenter(img draw.Image, bounds image.Rectangle, pres *Presentation, index int) {
	slides := pres.Slides[index:]

	curR := bounds
	nextR := bounds
	noteR := bounds

	curR.Max.Y -= bounds.Dy() / 2

	nextR.Max.X -= bounds.Dx() / 2
	nextR.Min.Y += bounds.Dy() / 2

	noteR.Min.X += bounds.Dx() / 2
	noteR.Min.Y += bounds.Dy() / 2

	bg := image.NewUniform(color.Gray{50})
	fg := image.NewUniform(color.Gray{200})

	slides[0].Draw(img, curR)
	if len(slides) > 1 {
		slides[1].Draw(img, nextR)
	} else {
		draw.Draw(img, nextR, bg, image.Point{}, draw.Src)
	}
	if slides[0].Notes != "" {
		notecfg := pres.Conf
		notecfg.Foreground = fg
		notecfg.Background = bg
		noteslide := Slide{notecfg, "", []SlideContent{
			MarkupText{
				Markup{
					Text: slides[0].Notes,
				},
			},
		}}
		noteslide.Draw(img, noteR)
	} else {
		draw.Draw(img, noteR, bg, image.Point{}, draw.Src)
	}
}
