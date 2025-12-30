package models

import (
	"fmt"
	"math"
)

// Color represents a LIFX color in HSBK form.
type Color struct {
	Hue        int // 0-360
	Saturation int // 0-100
	Brightness int // 0-100
	Kelvin     int // 2500-9000
}

// Clamp returns a copy of the color clamped to supported ranges.
func (c Color) Clamp() Color {
	if c.Hue < 0 {
		c.Hue = 0
	}
	if c.Hue > 360 {
		c.Hue = 360
	}
	if c.Saturation < 0 {
		c.Saturation = 0
	}
	if c.Saturation > 100 {
		c.Saturation = 100
	}
	if c.Brightness < 0 {
		c.Brightness = 0
	}
	if c.Brightness > 100 {
		c.Brightness = 100
	}
	if c.Kelvin < 2500 {
		c.Kelvin = 2500
	}
	if c.Kelvin > 9000 {
		c.Kelvin = 9000
	}
	return c
}

// RGB returns an RGB approximation for the color.
func (c Color) RGB() (r, g, b uint8) {
	c = c.Clamp()
	if c.Saturation <= 5 {
		return kelvinToRGB(c.Kelvin, c.Brightness)
	}
	return hsvToRGB(c.Hue, c.Saturation, c.Brightness)
}

// Hex returns the color as a hex string (#RRGGBB).
func (c Color) Hex() string {
	r, g, b := c.RGB()
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func hsvToRGB(h, s, v int) (r, g, b uint8) {
	hf := float64(h%360) / 60.0
	sf := float64(s) / 100.0
	vf := float64(v) / 100.0

	if sf <= 0.0001 {
		val := uint8(vf * 255)
		return val, val, val
	}

	i := math.Floor(hf)
	f := hf - i
	p := vf * (1 - sf)
	q := vf * (1 - sf*f)
	t := vf * (1 - sf*(1-f))

	var rf, gf, bf float64
	switch int(i) {
	case 0:
		rf, gf, bf = vf, t, p
	case 1:
		rf, gf, bf = q, vf, p
	case 2:
		rf, gf, bf = p, vf, t
	case 3:
		rf, gf, bf = p, q, vf
	case 4:
		rf, gf, bf = t, p, vf
	default:
		rf, gf, bf = vf, p, q
	}

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

// kelvinToRGB approximates a Kelvin temperature into RGB and scales brightness.
func kelvinToRGB(kelvin int, brightness int) (r, g, b uint8) {
	k := float64(kelvin)
	if k < 1000 {
		k = 1000
	}
	if k > 40000 {
		k = 40000
	}

	temp := k / 100.0

	var rf, gf, bf float64

	if temp <= 66 {
		rf = 255
		gf = 99.4708025861*math.Log(temp) - 161.1195681661
		if temp <= 19 {
			bf = 0
		} else {
			bf = 138.5177312231*math.Log(temp-10) - 305.0447927307
		}
	} else {
		rf = 329.698727446 * math.Pow(temp-60, -0.1332047592)
		gf = 288.1221695283 * math.Pow(temp-60, -0.0755148492)
		bf = 255
	}

	rf = clampFloat(rf, 0, 255)
	gf = clampFloat(gf, 0, 255)
	bf = clampFloat(bf, 0, 255)

	scale := float64(brightness) / 100.0
	return uint8(rf * scale), uint8(gf * scale), uint8(bf * scale)
}

func clampFloat(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
