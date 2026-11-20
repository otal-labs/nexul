// Package still turns an uploaded picture into a small still PNG, so no client animates a large GIF to draw a 22pt avatar.
package still

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // registers the GIF decoder; image.Decode reads a GIF's first frame only
	_ "image/jpeg"
	"image/png"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// maxSize is the largest edge a still is made at; a larger request is clamped to it.
const maxSize = 512

// maxPixels refuses a picture whose header claims more than this, before decoding allocates for it.
const maxPixels = 40_000_000

// Fit applies a ?size= value: absent leaves the picture as uploaded, a number makes a still of at most that edge.
func Fit(contentType string, data []byte, rawSize string) (string, []byte, error) {
	size, err := parseSize(rawSize)
	if err != nil || size == 0 {
		return contentType, data, err
	}
	return of(contentType, data, size)
}

func parseSize(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%w: size must be a whole number of pixels", apperrs.ErrInvalid)
	}
	return min(n, maxSize), nil
}

// of returns the picture as a PNG that fits size by size, its first frame when animated. A format the standard
// library can't decode (WebP, AVIF) comes back as it was.
func of(contentType string, data []byte, size int) (string, []byte, error) {
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/gif" {
		return contentType, data, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", nil, fmt.Errorf("%w: read picture header: %v", apperrs.ErrInvalid, err)
	}
	if cfg.Width*cfg.Height > maxPixels {
		return "", nil, fmt.Errorf("%w: picture is %dx%d, too large to shrink", apperrs.ErrInvalid, cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", nil, fmt.Errorf("%w: decode picture: %v", apperrs.ErrInvalid, err)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, shrink(src, size)); err != nil {
		return "", nil, fmt.Errorf("encode still: %w", err)
	}
	return "image/png", out.Bytes(), nil
}

// shrink averages each box of source pixels into one, keeping the aspect ratio; a picture already small enough keeps its size.
func shrink(src image.Image, size int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	flat := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(flat, flat.Bounds(), src, b.Min, draw.Src)
	long := max(w, h)
	if long <= size {
		return flat
	}
	dw, dh := max(1, w*size/long), max(1, h*size/long)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range dh {
		for x := range dw {
			average(flat, image.Rect(x*w/dw, y*h/dh, (x+1)*w/dw, (y+1)*h/dh), dst.Pix[dst.PixOffset(x, y):])
		}
	}
	return dst
}

// average writes the mean of box's premultiplied RGBA pixels into the first four bytes of into.
func average(src *image.RGBA, box image.Rectangle, into []byte) {
	var sum [4]int
	for y := box.Min.Y; y < box.Max.Y; y++ {
		row := src.Pix[src.PixOffset(box.Min.X, y):src.PixOffset(box.Max.X, y)]
		for i := 0; i < len(row); i += 4 {
			sum[0] += int(row[i])
			sum[1] += int(row[i+1])
			sum[2] += int(row[i+2])
			sum[3] += int(row[i+3])
		}
	}
	n := max(1, box.Dx()*box.Dy())
	for c := range sum {
		into[c] = byte(sum[c] / n)
	}
}
