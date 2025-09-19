package main

import (
	"errors"
	"image/color"

	"golang.org/x/image/colornames"
)

var (
	ErrInvalidHexColor = errors.New("invalid hex color")
	ErrUnknownColor    = errors.New("unknown color name")
)

func parseColor(str string) (color.Color, error) {
	if str[0] != '#' {
		color, ok := colornames.Map[str]
		if !ok {
			return nil, ErrUnknownColor
		}
		return color, nil
	}
	str = str[1:] /* skip over hash */

	var r, g, b, a uint8
	switch len(str) {
	case 8: // rr gg bb aa
		var ok bool
		if r, ok = parseByte(str[0], str[1]); !ok {
			return nil, ErrInvalidHexColor
		}
		if g, ok = parseByte(str[2], str[3]); !ok {
			return nil, ErrInvalidHexColor
		}
		if b, ok = parseByte(str[4], str[5]); !ok {
			return nil, ErrInvalidHexColor
		}
		if a, ok = parseByte(str[6], str[7]); !ok {
			return nil, ErrInvalidHexColor
		}
	case 6: // rr gg bb
		var ok bool
		if r, ok = parseByte(str[0], str[1]); !ok {
			return nil, ErrInvalidHexColor
		}
		if g, ok = parseByte(str[2], str[3]); !ok {
			return nil, ErrInvalidHexColor
		}
		if b, ok = parseByte(str[4], str[5]); !ok {
			return nil, ErrInvalidHexColor
		}
		a = 0xFF
	case 4: // r g b a  (each nibble doubled)
		var ok bool
		if r, ok = parseByte(str[0]); !ok {
			return nil, ErrInvalidHexColor
		}
		if g, ok = parseByte(str[1]); !ok {
			return nil, ErrInvalidHexColor
		}
		if b, ok = parseByte(str[2]); !ok {
			return nil, ErrInvalidHexColor
		}
		if a, ok = parseByte(str[3]); !ok {
			return nil, ErrInvalidHexColor
		}
	case 3: // r g b (each nibble doubled)
		var ok bool
		if r, ok = parseByte(str[0]); !ok {
			return nil, ErrInvalidHexColor
		}
		if g, ok = parseByte(str[1]); !ok {
			return nil, ErrInvalidHexColor
		}
		if b, ok = parseByte(str[2]); !ok {
			return nil, ErrInvalidHexColor
		}
		a = 0xFF
	default:
		return nil, ErrInvalidHexColor
	}
	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

func parseByte(c ...byte) (uint8, bool) {
	var h, l byte
	switch len(c) {
	case 1:
		h = c[0]
		l = c[0]
	case 2:
		h = c[0]
		l = c[1]
	}

	hi, ok1 := hexNibble(h)
	lo, ok2 := hexNibble(l)
	if !ok1 || !ok2 {
		return 0, false
	}
	return (hi << 4) | lo, true
}

func hexNibble(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
