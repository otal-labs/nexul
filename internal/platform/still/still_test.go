package still

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func animatedGIF(t *testing.T, w, h, frames int) []byte {
	t.Helper()
	anim := &gif.GIF{}
	for i := range frames {
		frame := image.NewPaletted(image.Rect(0, 0, w, h), palette.Plan9)
		for p := range frame.Pix {
			frame.Pix[p] = uint8((p + i*37) % len(palette.Plan9))
		}
		anim.Image = append(anim.Image, frame)
		anim.Delay = append(anim.Delay, 4)
	}
	var buf bytes.Buffer
	require.NoError(t, gif.EncodeAll(&buf, anim))
	return buf.Bytes()
}

// pngHeader is a PNG whose IHDR claims w by h and which holds no pixel data, the shape of a decompression bomb's header.
func pngHeader(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6
	chunk := append([]byte("IHDR"), ihdr...)
	out := []byte("\x89PNG\r\n\x1a\n")
	out = binary.BigEndian.AppendUint32(out, 13)
	out = append(out, chunk...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(chunk))
}

func decoded(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "png", format)
	return img
}

func TestFit_RefusesWhatItCannotShrink(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
		size        string
	}{
		{"a size that is not a number", "image/png", nil, "big"},
		{"a size of zero", "image/png", nil, "0"},
		{"a negative size", "image/png", nil, "-4"},
		{"bytes that are not the picture they claim", "image/gif", []byte("not a gif"), "48"},
		{"a header claiming more pixels than may be decoded", "image/png", pngHeader(20000, 20000), "48"},
		{"a header with no pixels behind it", "image/png", pngHeader(64, 64), "48"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Fit(tt.contentType, tt.data, tt.size)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestFit_WithoutASize_ServesThePictureAsUploaded(t *testing.T) {
	data := animatedGIF(t, 8, 8, 2)
	contentType, out, err := Fit("image/gif", data, "")
	require.NoError(t, err)
	assert.Equal(t, "image/gif", contentType)
	assert.Equal(t, data, out)
}

func TestFit_AFormatItCannotDecode_ComesBackUnchanged(t *testing.T) {
	for _, contentType := range []string{"image/webp", "image/avif", "application/octet-stream"} {
		contentType, out, err := Fit(contentType, []byte("<svg></svg>"), "48")
		require.NoError(t, err)
		assert.Equal(t, []byte("<svg></svg>"), out, contentType)
	}
}

func TestFit_AnAnimatedGIF_BecomesOneSmallPNGFrameKeepingItsShape(t *testing.T) {
	contentType, out, err := Fit("image/gif", animatedGIF(t, 600, 300, 12), "48")
	require.NoError(t, err)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, image.Rect(0, 0, 48, 24), decoded(t, out).Bounds())
}

func TestFit_APictureAlreadySmallEnough_KeepsItsSize(t *testing.T) {
	_, out, err := Fit("image/png", encodePNG(t, image.NewRGBA(image.Rect(0, 0, 20, 10))), "48")
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 20, 10), decoded(t, out).Bounds())
}

func TestFit_ASizeOverTheCap_IsClampedToIt(t *testing.T) {
	_, out, err := Fit("image/png", encodePNG(t, image.NewRGBA(image.Rect(0, 0, 1024, 8))), "9999")
	require.NoError(t, err)
	assert.Equal(t, maxSize, decoded(t, out).Bounds().Dx())
}

func TestFit_EachStillPixel_IsTheAverageOfTheBoxItReplaces(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{255, 255, 255, 255})
	src.Set(1, 1, color.RGBA{255, 255, 255, 255})
	src.Set(1, 0, color.RGBA{0, 0, 0, 255})
	src.Set(0, 1, color.RGBA{0, 0, 0, 255})
	_, out, err := Fit("image/png", encodePNG(t, src), "1")
	require.NoError(t, err)
	r, g, b, a := decoded(t, out).At(0, 0).RGBA()
	assert.Equal(t, [4]uint32{127, 127, 127, 255}, [4]uint32{r >> 8, g >> 8, b >> 8, a >> 8})
}
