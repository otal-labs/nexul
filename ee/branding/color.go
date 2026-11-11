package branding

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// rgb is an sRGB colour with channels in 0..1.
type rgb [3]float64

// lch is a colour in OKLCH: lightness 0..1, chroma, hue in degrees.
type lch struct{ L, C, H float64 }

func decode(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func encode(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func cube(v float64) float64 { return v * v * v }

func clamp01(v float64) float64 { return math.Min(1, math.Max(0, v)) }

// parseHex reads #rgb, #rgba, #rrggbb and #rrggbbaa, the only forms a VS Code theme uses.
func parseHex(s string) (c rgb, alpha float64, ok bool) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(h) {
	case 3, 4:
		var b strings.Builder
		for _, r := range h {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		h = b.String()
	case 6, 8:
	default:
		return rgb{}, 0, false
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return rgb{}, 0, false
	}
	alpha = 1
	if len(h) == 8 {
		alpha = float64(n&0xff) / 255
		n >>= 8
	}
	return rgb{float64(n>>16&0xff) / 255, float64(n>>8&0xff) / 255, float64(n&0xff) / 255}, alpha, true
}

func (c rgb) hex() string {
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(clamp01(c[0])*255)), int(math.Round(clamp01(c[1])*255)), int(math.Round(clamp01(c[2])*255)))
}

// over flattens a translucent colour onto an opaque one, since every role Nexul paints is opaque.
func (c rgb) over(base rgb, alpha float64) rgb {
	return rgb{c[0]*alpha + base[0]*(1-alpha), c[1]*alpha + base[1]*(1-alpha), c[2]*alpha + base[2]*(1-alpha)}
}

func (c rgb) luminance() float64 {
	return 0.2126*decode(c[0]) + 0.7152*decode(c[1]) + 0.0722*decode(c[2])
}

func contrast(a, b rgb) float64 {
	hi, lo := a.luminance(), b.luminance()
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

func (c rgb) lch() lch {
	r, g, b := decode(c[0]), decode(c[1]), decode(c[2])
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L := 0.2104542553*l + 0.793617785*m - 0.0040720468*s
	a := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	bb := 0.0259040371*l + 0.7827717662*m - 0.808675766*s
	h := math.Atan2(bb, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return lch{L, math.Hypot(a, bb), h}
}

// raw converts without clipping, so a caller can tell whether the colour fits sRGB.
func (c lch) raw() (r, g, b float64) {
	h := c.H * math.Pi / 180
	a, bb := c.C*math.Cos(h), c.C*math.Sin(h)
	l := cube(c.L + 0.3963377774*a + 0.2158037573*bb)
	m := cube(c.L - 0.1055613458*a - 0.0638541728*bb)
	s := cube(c.L - 0.0894841775*a - 1.291485548*bb)
	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.707614701*s
}

// fit pulls the chroma down until the colour is inside sRGB, so what is measured is what a browser paints.
func (c lch) fit() lch {
	for range 40 {
		r, g, b := c.raw()
		if r >= -0.0005 && r <= 1.0005 && g >= -0.0005 && g <= 1.0005 && b >= -0.0005 && b <= 1.0005 {
			return c
		}
		c.C *= 0.93
	}
	c.C = 0
	return c
}

func (c lch) rgb() rgb {
	r, g, b := c.fit().raw()
	return rgb{encode(clamp01(r)), encode(clamp01(g)), encode(clamp01(b))}
}

// css writes the colour as the OKLCH string the web app's tokens hold.
func (c lch) css() string {
	f := c.fit()
	return fmt.Sprintf("oklch(%s %s %s)", trim(f.L, 4), trim(f.C, 4), trim(f.H, 2))
}

func trim(v float64, places int) string {
	return strconv.FormatFloat(math.Round(v*math.Pow10(places))/math.Pow10(places), 'f', -1, 64)
}

// withL moves lightness only, keeping the hue and as much chroma as still fits.
func (c lch) withL(l float64) lch {
	c.L = math.Min(1, math.Max(0, l))
	return c
}
