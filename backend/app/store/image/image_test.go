package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math/big"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_SaveAndLoad(t *testing.T) {
	store := StoreMock{
		SaveFunc: func(string, []byte) error {
			return nil
		},
		LoadFunc: func(string) ([]byte, error) {
			return nil, nil
		},
	}
	svc := NewService(&store, ServiceParams{MaxSize: 1500, MaxWidth: 32, MaxHeight: 32})

	err := svc.SaveWithID("test_id", gopherPNG())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(store.SaveCalls()))
	assert.Equal(t, "test_id", store.SaveCalls()[0].ID)

	img, err := svc.Load("test_id")
	assert.NoError(t, err)
	assert.Nil(t, img)
	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "test_id", store.LoadCalls()[0].ID)
}

// the resized dimensions are what resize promises; the encoded length is whatever the compressor
// in the toolchain happens to produce, and pinning it fails on a go release that changes it
func TestService_Resize(t *testing.T) {
	img, err := readAndValidateImage(gopherPNG(), 1500)
	assert.NoError(t, err)
	assert.NotEmpty(t, img)

	img = resize(img, 32, 32)
	assertImageFits(t, img, 32, 32)
}

func TestService_ResizeJpeg(t *testing.T) {
	fh, err := os.Open("testdata/circles.jpg")
	defer func() { assert.NoError(t, fh.Close()) }()
	assert.NoError(t, err)

	img, err := readAndValidateImage(fh, 32000)
	assert.NoError(t, err)
	assert.NotEmpty(t, img)

	img = resize(img, 400, 300)
	assertImageFits(t, img, 400, 300)
}

// assertImageFits decodes the image and checks it is inside the box resize was given, and that it
// touches one side of it, which is what fitting to a box rather than merely shrinking means
func assertImageFits(t *testing.T, data []byte, limitW, limitH int) {
	t.Helper()

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err, "the resized image does not decode")

	assert.LessOrEqual(t, cfg.Width, limitW, "wider than the box it was resized into")
	assert.LessOrEqual(t, cfg.Height, limitH, "taller than the box it was resized into")
	assert.True(t, cfg.Width == limitW || cfg.Height == limitH,
		"%dx%d touches neither side of the %dx%d box, so it was not fitted to it", cfg.Width, cfg.Height, limitW, limitH)
}

func TestService_SaveTooLarge(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}
	svc.MaxSize = 2000
	_, err := svc.Save("user2", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
	err = svc.SaveWithID("test_id", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestService_WrongFormat(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}

	_, err := svc.Save("user1", strings.NewReader("blah blah bad image"))
	assert.Error(t, err)
}

func TestService_ExtractPictures(t *testing.T) {
	svc := Service{ImageAPI: "/blah/", ProxyAPI: "/non_existent"}
	html := `blah <img src="/blah/user1/pic1.png"/> foo
<img src="/blah/user2/pic3.png"/> xyz <p>123</p> <img src="/pic3.png"/> <img src="https://i.ibb.co/0cqqqnD/ezgif-5-3b07b6b97610.png" alt="">`
	ids := svc.ExtractPictures(html)
	require.Equal(t, 2, len(ids), "two images")
	assert.Equal(t, "user1/pic1.png", ids[0])
	assert.Equal(t, "user2/pic3.png", ids[1])

	svc = Service{ImageAPI: "https://remark42.radio-t.com/api/v1/picture/", ProxyAPI: "https://remark42.radio-t.com/api/v1/img"}
	html = `<p>TLDR: такое в go пока правильно посчитать трудно. То, что они считают это общее количество go packages в коде.
</p>\n\n<p>Пакеты в го это средство организации кода, они могут быть связанны друг с другом в рамках одной библиотеки (модуля).
Например одна из моих вот так выглядит на libraries.io:</p>\n\n
<p><img src="https://remark42.radio-t.com/api/v1/picture/github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png" alt="bjtr0-201906-08110846-i324c.png"/></p>\n\n<p>
По форме все верно, это все packages, но по сути это все одна библиотека организованная таким образом. При ее импорте, например посредством go mod, она выглядит как один модуль, т.е.
<code>github.com/go-pkgz/auth v0.5.2</code>.</p>\n`
	ids = svc.ExtractPictures(html)
	require.Equal(t, 1, len(ids), "one image in")
	assert.Equal(t, "github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png", ids[0])

	// proxied image
	html = `<img src="https://remark42.radio-t.com/api/v1/img?src=aHR0cHM6Ly9ob21lcGFnZXMuY2FlLndpc2MuZWR1L35lY2U1MzMvaW1hZ2VzL2JvYXQucG5n" alt="cat.png">`
	ids = svc.ExtractPictures(html)
	require.Equal(t, 1, len(ids), "one image in")
	assert.Equal(t, "cached_images/12318fbd4c55e9d177b8b5ae197bc89c5afd8e07-a41fcb00643f28d700504256ec81cbf2e1aac53e", ids[0])
	require.Empty(t, svc.ExtractNonProxiedPictures(html), "no non-proxied images expected to be found")

	// bad url
	html = `<img src=" https://remark42.radio-t.com/api/v1/img">`
	ids = svc.ExtractPictures(html)
	require.Empty(t, ids)

	// bad src
	html = `<img src="https://remark42.radio-t.com/api/v1/img?src=bad">`
	ids = svc.ExtractPictures(html)
	require.Empty(t, ids)

	// good src with bad content
	badURL := base64.URLEncoding.EncodeToString([]byte(" http://foo.bar"))
	html = fmt.Sprintf(`<img src="https://remark42.radio-t.com/api/v1/img?src=%s">`, badURL)
	ids = svc.ExtractPictures(html)
	require.Empty(t, ids)

	// traversal segments in a comment img src reached the fs store as an image id
	html = `<img src="https://remark42.radio-t.com/api/v1/picture/../remark.db">
<img src="https://remark42.radio-t.com/api/v1/picture/../..">
<img src="https://remark42.radio-t.com/api/v1/picture/user1/..">
<img src="https://remark42.radio-t.com/api/v1/picture/./pic.png">
<img src="https://remark42.radio-t.com/api/v1/picture/user1/">
<img src="https://remark42.radio-t.com/api/v1/picture/user1/pic` + "\x01" + `.png">
<img src="https://remark42.radio-t.com/api/v1/picture/user1/pic1.png">`
	assert.Equal(t, []string{"user1/pic1.png"}, svc.ExtractPictures(html))
	assert.Equal(t, []string{"user1/pic1.png"}, svc.ExtractNonProxiedPictures(html))
}

func TestService_SetSizes(t *testing.T) {
	encodePNG := func(w, h int) []byte {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h))))
		return buf.Bytes()
	}
	var plainJPEG bytes.Buffer
	require.NoError(t, jpeg.Encode(&plainJPEG, image.NewGray(image.Rect(0, 0, 240, 180)), nil))

	cachedURL := "https://example.com/cached.png"
	cachedID, err := CachedImgID(cachedURL)
	require.NoError(t, err)
	stored := map[string][]byte{
		"user1/wide.png": encodePNG(600, 200), cachedID: encodePNG(600, 200), "user1/tall.png": encodePNG(200, 900),
		"user1/turned.jpg":  withExifOrientation(t, plainJPEG.Bytes(), binary.BigEndian, 6),
		"user1/upright.jpg": withExifOrientation(t, plainJPEG.Bytes(), binary.LittleEndian, 3),
		"user1/broken.png":  []byte("not an image"),
		"user1/sliver.png":  encodePNG(50, 700),
		"user1/dense.jpg":   withExifTIFF(t, plainJPEG.Bytes(), exifTIFFWithPixels(binary.LittleEndian, 1, 120, 90)),
		"user1/exact.jpg":   withExifTIFF(t, plainJPEG.Bytes(), exifTIFFWithPixels(binary.BigEndian, 6, 240, 180)),
		"user1/dense.png":   withPNGExif(t, encodePNG(400, 200), exifTIFFWithPixels(binary.BigEndian, 1, 200, 100)),
		"user1/shot.png":    withPNGExif(t, encodePNG(400, 200), exifTIFFWithPixels(binary.BigEndian, 1, 400, 200)),
		"user1/long.jpg":    withExifTIFF(t, plainJPEG.Bytes(), exifTIFFOrientationAs(binary.BigEndian, 4, 1)),
		"user1/stray.jpg":   append([]byte{0xff, 0xd8, 0x00}, withExifOrientation(t, plainJPEG.Bytes(), binary.BigEndian, 6)[2:]...),
		"user1/small.gif":   gifWithFrame(t, 400, 200, image.Rect(0, 0, 100, 50)),
		"user1/full.gif":    gifWithFrame(t, 400, 200, image.Rect(0, 0, 400, 200)),
		"user1/swapped.jpg": withExifTIFF(t, plainJPEG.Bytes(), exifTIFFWithPixels(binary.BigEndian, 6, 180, 240)),
		"user1/turned.png":  withPNGExifOrientation(t, encodePNG(240, 180), 6),
		"user1/upside.png":  withPNGExifOrientation(t, encodePNG(240, 180), 3),
	}
	store := StoreMock{LoadFunc: func(id string) ([]byte, error) {
		if data, ok := stored[id]; ok {
			return data, nil
		}
		return nil, fmt.Errorf("can't load image %s", id)
	}}
	svc := NewService(&store, ServiceParams{ImageAPI: "https://r.example.com/api/v1/picture/", ProxyAPI: "https://r.example.com/api/v1/img"})
	pic := func(id string) string { return "https://r.example.com/api/v1/picture/" + id }
	proxied := func(u string) string {
		return "https://r.example.com/api/v1/img?src=" + base64.URLEncoding.EncodeToString([]byte(u))
	}
	unchanged := func(img string) [2]string { return [2]string{img, img} }

	tbl := []struct {
		name  string
		inOut [2]string
	}{
		{"uploaded", [2]string{`<img src="` + pic("user1/wide.png") + `" alt="a"/>`,
			`<img src="` + pic("user1/wide.png") + `" alt="a" width="600" height="200" class="sized"/>`}},
		{"taller than drawn, scaled down", [2]string{`<img src="` + pic("user1/tall.png") + `"/>`,
			`<img src="` + pic("user1/tall.png") + `" width="67" height="300" class="sized"/>`}},
		{"width of a capped image rounded up", [2]string{`<img src="` + pic("user1/sliver.png") + `"/>`,
			`<img src="` + pic("user1/sliver.png") + `" width="22" height="300" class="sized"/>`}},
		{"class of its own replaced", [2]string{`<img src="` + pic("user1/wide.png") + `" class="x"/>`,
			`<img src="` + pic("user1/wide.png") + `" class="sized" width="600" height="200"/>`}},
		{"jpeg with exif pixel dimensions other than its size", unchanged(`<img src="` + pic("user1/dense.jpg") + `"/>`)},
		{"jpeg with exif pixel dimensions of its size, turned", [2]string{`<img src="` + pic("user1/exact.jpg") + `"/>`,
			`<img src="` + pic("user1/exact.jpg") + `" width="180" height="240" class="sized"/>`}},
		{"jpeg with exif pixel dimensions of its turned size", [2]string{`<img src="` + pic("user1/swapped.jpg") + `"/>`,
			`<img src="` + pic("user1/swapped.jpg") + `" width="180" height="240" class="sized"/>`}},
		{"png with exif pixel dimensions other than its size", unchanged(`<img src="` + pic("user1/dense.png") + `"/>`)},
		{"png with exif pixel dimensions of its size", [2]string{`<img src="` + pic("user1/shot.png") + `"/>`,
			`<img src="` + pic("user1/shot.png") + `" width="400" height="200" class="sized"/>`}},
		{"jpeg orientation as a LONG, which browsers ignore", [2]string{`<img src="` + pic("user1/long.jpg") + `"/>`,
			`<img src="` + pic("user1/long.jpg") + `" width="240" height="180" class="sized"/>`}},
		{"jpeg with a stray byte before its exif", unchanged(`<img src="` + pic("user1/stray.jpg") + `"/>`)},
		{"gif with a frame smaller than its screen", unchanged(`<img src="` + pic("user1/small.gif") + `"/>`)},
		{"gif with a frame covering its screen", [2]string{`<img src="` + pic("user1/full.gif") + `"/>`,
			`<img src="` + pic("user1/full.gif") + `" width="400" height="200" class="sized"/>`}},
		{"png turned by exif", unchanged(`<img src="` + pic("user1/turned.png") + `"/>`)},
		{"png upside down by exif", [2]string{`<img src="` + pic("user1/upside.png") + `"/>`,
			`<img src="` + pic("user1/upside.png") + `" width="240" height="180" class="sized"/>`}},
		{"jpeg turned by exif", [2]string{`<img src="` + pic("user1/turned.jpg") + `"/>`,
			`<img src="` + pic("user1/turned.jpg") + `" width="180" height="240" class="sized"/>`}},
		{"jpeg upside down by exif", [2]string{`<img src="` + pic("user1/upright.jpg") + `"/>`,
			`<img src="` + pic("user1/upright.jpg") + `" width="240" height="180" class="sized"/>`}},
		{"proxied and cached", [2]string{`<img src="` + proxied(cachedURL) + `"/>`,
			`<img src="` + proxied(cachedURL) + `" width="600" height="200" class="sized"/>`}},
		{"proxied, not cached yet", unchanged(`<img src="` + proxied("https://example.com/new.png") + `"/>`)},
		{"uploaded, missing", unchanged(`<img src="` + pic("user1/gone.png") + `"/>`)},
		{"stored, not an image", unchanged(`<img src="` + pic("user1/broken.png") + `"/>`)},
		{"external", unchanged(`<img src="https://example.com/cached.png"/>`)},
		{"external with a picture url in its query",
			unchanged(`<img src="https://thumbs.example.com/x?u=` + pic("user1/wide.png") + `"/>`)},
		{"own width", unchanged(`<img src="` + pic("user1/wide.png") + `" width="10"/>`)},
		{"own height", unchanged(`<img src="` + pic("user1/wide.png") + `" height="10"/>`)},
		{"two images and text", [2]string{`<p>one <img src="` + pic("user1/wide.png") + `"/> two <img src="https://example.com/x.png"/></p>`,
			`<p>one <img src="` + pic("user1/wide.png") + `" width="600" height="200" class="sized"/> two <img src="https://example.com/x.png"/></p>`}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.inOut[1], svc.SetSizes(tt.inOut[0]))
		})
	}

	t.Run("a picture shown many times is read once", func(t *testing.T) {
		before := len(store.LoadCalls())
		res := svc.SetSizes(strings.Repeat(`<img src="`+pic("user1/wide.png")+`"/>`, 300))
		assert.Equal(t, 300, strings.Count(res, `class="sized"`))
		assert.Equal(t, 1, len(store.LoadCalls())-before)
	})

	t.Run("no more than maxSizedPictures different pictures are read", func(t *testing.T) {
		var html strings.Builder
		for n := range maxSizedPictures + 5 {
			stored[fmt.Sprintf("user2/p%d.png", n)] = stored["user1/wide.png"]
			fmt.Fprintf(&html, `<img src="%s"/>`, pic(fmt.Sprintf("user2/p%d.png", n)))
		}
		before := len(store.LoadCalls())
		res := svc.SetSizes(html.String())
		assert.Equal(t, maxSizedPictures, strings.Count(res, `class="sized"`))
		assert.Equal(t, maxSizedPictures, len(store.LoadCalls())-before)
	})
}

func TestGifFrameCoversScreen(t *testing.T) {
	full := gifWithFrame(t, 40, 20, image.Rect(0, 0, 40, 20))
	assert.True(t, gifFrameCoversScreen(full, 40, 20))
	assert.False(t, gifFrameCoversScreen(gifWithFrame(t, 40, 20, image.Rect(0, 0, 10, 5)), 40, 20), "smaller frame")
	assert.False(t, gifFrameCoversScreen(gifWithFrame(t, 40, 20, image.Rect(5, 0, 40, 20)), 40, 20), "frame at an offset")

	// two frames bring the looping application extension and graphic control extensions before the first frame
	var buf bytes.Buffer
	pal := color.Palette{color.Black, color.White}
	frames := []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 40, 20), pal), image.NewPaletted(image.Rect(0, 0, 40, 20), pal)}
	require.NoError(t, gif.EncodeAll(&buf, &gif.GIF{Image: frames, Delay: []int{10, 10}, Disposal: []byte{gif.DisposalBackground, 0},
		Config: image.Config{ColorModel: pal, Width: 40, Height: 20}}))
	animated := buf.Bytes()
	require.True(t, bytes.Contains(animated, []byte("NETSCAPE2.0")))
	assert.True(t, gifFrameCoversScreen(animated, 40, 20))

	for n := range len(animated) { // a file cut anywhere is read without a panic and is not sized
		if gifFrameCoversScreen(animated[:n], 40, 20) {
			idx := bytes.IndexByte(animated[13+6:], 0x2c) // first image separator after the color table
			require.GreaterOrEqual(t, n, 13+6+idx+9, "sized only once the image descriptor is whole")
		}
	}
}

// jpegExifTags reads the EXIF tags of a jpeg
func jpegExifTags(data []byte) exifTags {
	tiff, _ := jpegExif(data)
	return readExif(tiff)
}

// pictureExifTags reads the EXIF tags of a picture
func pictureExifTags(data []byte, format string) exifTags {
	tiff, _ := pictureExif(data, format)
	return readExif(tiff)
}

// exifTIFF returns a TIFF structure with IFD0 holding one orientation entry
func exifTIFF(order binary.AppendByteOrder, orientation uint16) []byte {
	tiff := []byte("II")
	if order == binary.AppendByteOrder(binary.BigEndian) {
		tiff = []byte("MM")
	}
	tiff = order.AppendUint16(tiff, 42)
	tiff = order.AppendUint32(tiff, 8)           // IFD0 right after the header
	tiff = order.AppendUint16(tiff, 1)           // one entry
	tiff = order.AppendUint16(tiff, 0x0112)      // orientation
	tiff = order.AppendUint16(tiff, 3)           // SHORT
	tiff = order.AppendUint32(tiff, 1)           // one value
	tiff = order.AppendUint16(tiff, orientation) // the value, left-justified in four bytes
	return append(tiff, 0, 0, 0, 0, 0, 0)        // padding, then no next IFD
}

// exifTIFFWithPixels returns a TIFF structure with IFD0 holding the orientation and a pointer to an Exif
// IFD holding PixelXDimension as a LONG and PixelYDimension as a SHORT
func exifTIFFWithPixels(order binary.AppendByteOrder, orientation uint16, pixW uint32, pixH uint16) []byte {
	tiff := []byte("II")
	if order == binary.AppendByteOrder(binary.BigEndian) {
		tiff = []byte("MM")
	}
	tiff = order.AppendUint16(tiff, 42)
	tiff = order.AppendUint32(tiff, 8)
	tiff = order.AppendUint16(tiff, 2)
	tiff = order.AppendUint16(tiff, 0x0112) // orientation, SHORT
	tiff = order.AppendUint16(tiff, 3)
	tiff = order.AppendUint32(tiff, 1)
	tiff = order.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0)
	tiff = order.AppendUint16(tiff, 0x8769) // Exif IFD pointer, LONG
	tiff = order.AppendUint16(tiff, 4)
	tiff = order.AppendUint32(tiff, 1)
	tiff = order.AppendUint32(tiff, 8+2+2*12+4) // right after IFD0 and its next-IFD offset
	tiff = append(tiff, 0, 0, 0, 0)
	tiff = order.AppendUint16(tiff, 2)
	tiff = order.AppendUint16(tiff, 0xa002) // PixelXDimension, LONG
	tiff = order.AppendUint16(tiff, 4)
	tiff = order.AppendUint32(tiff, 1)
	tiff = order.AppendUint32(tiff, pixW)
	tiff = order.AppendUint16(tiff, 0xa003) // PixelYDimension, SHORT
	tiff = order.AppendUint16(tiff, 3)
	tiff = order.AppendUint32(tiff, 1)
	tiff = order.AppendUint16(tiff, pixH)
	return append(tiff, 0, 0, 0, 0, 0, 0)
}

// withExifTIFF returns the jpeg with an Exif APP1 segment holding the TIFF structure put in after SOI
func withExifTIFF(t *testing.T, jpg, tiff []byte) []byte {
	t.Helper()
	seg := append([]byte("Exif\x00\x00"), tiff...)
	require.True(t, bytes.HasPrefix(jpg, []byte{0xff, 0xd8}))
	res := append([]byte{0xff, 0xd8, 0xff, 0xe1, byte((len(seg) + 2) >> 8), byte(len(seg) + 2)}, seg...)
	return append(res, jpg[2:]...)
}

// withExifOrientation returns the jpeg with an Exif APP1 segment holding the orientation put in after SOI
func withExifOrientation(t *testing.T, jpg []byte, order binary.AppendByteOrder, orientation uint16) []byte {
	t.Helper()
	seg := append([]byte("Exif\x00\x00"), exifTIFF(order, orientation)...)
	require.True(t, bytes.HasPrefix(jpg, []byte{0xff, 0xd8}))
	res := append([]byte{0xff, 0xd8, 0xff, 0xe1, byte((len(seg) + 2) >> 8), byte(len(seg) + 2)}, seg...)
	return append(res, jpg[2:]...)
}

// pngChunk returns a png chunk with its length and crc
func pngChunk(kind string, data []byte) []byte {
	c := binary.BigEndian.AppendUint32(nil, uint32(len(data))) //nolint:gosec // a few bytes
	c = append(append(c, kind...), data...)
	return binary.BigEndian.AppendUint32(c, crc32.ChecksumIEEE(c[4:]))
}

// withPNGExifOrientation returns the png with an eXIf chunk holding the orientation put in after IHDR
func withPNGExifOrientation(t *testing.T, img []byte, orientation uint16) []byte {
	t.Helper()
	return withPNGExif(t, img, exifTIFF(binary.BigEndian, orientation))
}

// withPNGExif returns the png with an eXIf chunk holding the TIFF structure put in after IHDR
func withPNGExif(t *testing.T, img, tiff []byte) []byte {
	t.Helper()
	ihdrEnd := 8 + 8 + 13 + 4 // signature, IHDR length and type, its data, its crc
	require.Equal(t, "IHDR", string(img[12:16]))
	return append(append(append([]byte{}, img[:ihdrEnd]...), pngChunk("eXIf", tiff)...), img[ihdrEnd:]...)
}

// exifTIFFOrientationAs returns a TIFF structure with IFD0 holding orientation 6 as an entry of the
// given type and count, the value in the first two or four bytes of the value field
func exifTIFFOrientationAs(order binary.AppendByteOrder, typ uint16, count uint32) []byte {
	const value = 6
	tiff := []byte("II")
	if order == binary.AppendByteOrder(binary.BigEndian) {
		tiff = []byte("MM")
	}
	tiff = order.AppendUint16(tiff, 42)
	tiff = order.AppendUint32(tiff, 8)
	tiff = order.AppendUint16(tiff, 1)
	tiff = order.AppendUint16(tiff, 0x0112)
	tiff = order.AppendUint16(tiff, typ)
	tiff = order.AppendUint32(tiff, count)
	if typ == 4 {
		tiff = order.AppendUint32(tiff, value)
	} else {
		tiff = order.AppendUint16(tiff, value)
		tiff = append(tiff, 0, 0)
	}
	return append(tiff, 0, 0, 0, 0)
}

// gifWithFrame returns a one-frame gif with a logical screen of the given size and a frame at rect
func gifWithFrame(t *testing.T, screenW, screenH int, rect image.Rectangle) []byte {
	t.Helper()
	var buf bytes.Buffer
	frame := image.NewPaletted(rect, color.Palette{color.Black, color.White})
	require.NoError(t, gif.EncodeAll(&buf, &gif.GIF{Image: []*image.Paletted{frame}, Delay: []int{0},
		Config: image.Config{ColorModel: frame.Palette, Width: screenW, Height: screenH}}))
	return buf.Bytes()
}

func TestPNGAndWebPOrientation(t *testing.T) {
	webpOrientation := func(data []byte) int { return readExif(webpExif(data)).orientation }
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8))))
	plain := buf.Bytes()
	assert.Equal(t, 0, readExif(pngExif(plain)).orientation, "no eXIf")
	turned := withPNGExifOrientation(t, plain, 6)
	_, format, err := image.DecodeConfig(bytes.NewReader(turned))
	require.NoError(t, err, "the chunk leaves a valid png")
	assert.Equal(t, 6, pictureExifTags(turned, format).orientation)
	late := append(append(append([]byte{}, plain[:len(plain)-12]...), pngChunk("eXIf", exifTIFF(binary.BigEndian, 6))...), plain[len(plain)-12:]...)
	assert.Equal(t, 0, readExif(pngExif(late)).orientation, "eXIf after IDAT does not count")

	riff := func(chunks ...[]byte) []byte {
		body := []byte("WEBP")
		for _, c := range chunks {
			body = append(body, c...)
		}
		return append(binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body))), body...) //nolint:gosec // a few bytes
	}
	chunk := func(kind string, data []byte) []byte {
		c := binary.LittleEndian.AppendUint32([]byte(kind), uint32(len(data))) //nolint:gosec // a few bytes
		c = append(c, data...)
		if len(data)%2 == 1 {
			c = append(c, 0)
		}
		return c
	}
	assert.Equal(t, 0, webpOrientation(riff(chunk("VP8X", make([]byte, 10)))), "no EXIF")
	assert.Equal(t, 8, webpOrientation(riff(chunk("VP8X", make([]byte, 10)), chunk("ICCP", []byte{1, 2, 3}),
		chunk("EXIF", exifTIFF(binary.LittleEndian, 8)))), "after an odd-sized chunk")
	assert.Equal(t, 6, webpOrientation(riff(chunk("EXIF", append([]byte("Exif\x00\x00"), exifTIFF(binary.BigEndian, 6)...)))),
		"with the jpeg-style header")
	full := riff(chunk("EXIF", exifTIFF(binary.BigEndian, 6)))
	for n := range len(turned) - 1 { // a file cut anywhere is read without a panic
		_ = webpOrientation(full[:min(n, len(full))])
		_ = readExif(pngExif(turned[:n])).orientation
	}
}

func TestJpegOrientation(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), nil))
	plain := buf.Bytes()
	assert.Equal(t, 0, jpegExifTags(plain).orientation, "no exif")
	for _, o := range []uint16{1, 5, 6, 8} {
		assert.Equal(t, int(o), jpegExifTags(withExifOrientation(t, plain, binary.BigEndian, o)).orientation, "big endian %d", o)
		assert.Equal(t, int(o), jpegExifTags(withExifOrientation(t, plain, binary.LittleEndian, o)).orientation, "little endian %d", o)
	}
	assert.Equal(t, 0, jpegExifTags(withExifOrientation(t, plain, binary.BigEndian, 9)).orientation, "out of range")
	tags := jpegExifTags(withExifTIFF(t, plain, exifTIFFWithPixels(binary.BigEndian, 8, 70000, 300)))
	assert.Equal(t, exifTags{orientation: 8, pixelWidth: 70000, pixelHeight: 300}, tags, "big endian, LONG and SHORT")
	tags = jpegExifTags(withExifTIFF(t, plain, exifTIFFWithPixels(binary.LittleEndian, 3, 640, 480)))
	assert.Equal(t, exifTags{orientation: 3, pixelWidth: 640, pixelHeight: 480}, tags, "little endian")
	assert.Equal(t, 0, readExif(exifTIFFOrientationAs(binary.BigEndian, 4, 1)).orientation, "LONG")
	assert.Equal(t, 0, readExif(exifTIFFOrientationAs(binary.LittleEndian, 3, 2)).orientation, "two SHORTs")
	assert.Equal(t, 6, readExif(exifTIFFOrientationAs(binary.LittleEndian, 3, 1)).orientation, "one SHORT")
	full := exifTIFFWithPixels(binary.LittleEndian, 3, 640, 480)
	for n := range len(full) { // a structure cut anywhere is read without a panic
		readExif(full[:n])
	}
	turned := withExifOrientation(t, plain, binary.BigEndian, 6)
	for n := range len(turned) - 1 { // a file cut anywhere is read without a panic
		_ = jpegExifTags(turned[:n]).orientation
	}
}

func TestSafeIDSegment(t *testing.T) {
	tbl := []struct {
		seg  string
		safe bool
	}{
		{"user1", true},
		{"bjttt8ahajfmrhsula10.png", true},
		{"github_ef0f706a79cc24b17bbbb374cd234a691d034128", true},
		{"pic%00.png", true},
		{"", false},
		{".", false},
		{"..", false},
		{"..remark.db", false},
		{"a/b", false},
		{`a\b`, false},
		{"pic\x00.png", false},
		{"pic\n.png", false},
		{"pic\t.png", false},
	}
	for _, tt := range tbl {
		t.Run(strconv.Quote(tt.seg), func(t *testing.T) {
			assert.Equal(t, tt.safe, SafeIDSegment(tt.seg))
		})
	}
}

// a picture id must not let anyone work out the id of another upload: each is 128 random bits in hex,
// with no timestamp, per-process constant or counter in it
func TestService_SaveMakesRandomID(t *testing.T) {
	store := StoreMock{SaveFunc: func(string, []byte) error { return nil }}
	svc := NewService(&store, ServiceParams{MaxSize: 1500})

	const n = 64
	ids := make([]string, n)
	for i := range ids {
		id, err := svc.Save("user1", gopherPNG())
		require.NoError(t, err)
		require.Regexp(t, `^user1/[0-9a-f]{32}$`, id)
		assert.Equal(t, id, store.SaveCalls()[i].ID)
		ids[i] = strings.TrimPrefix(id, "user1/")
	}

	// a counter shows as consecutive ids a small step apart; two random ones come within 2^64 of each
	// other with a chance of about 2^-63
	near := new(big.Int).Lsh(big.NewInt(1), 64)
	for i := 1; i < n; i++ {
		prev, ok := new(big.Int).SetString(ids[i-1], 16)
		require.True(t, ok)
		curr, ok := new(big.Int).SetString(ids[i], 16)
		require.True(t, ok)
		assert.Positive(t, new(big.Int).Sub(curr, prev).CmpAbs(near),
			"consecutive ids %s and %s are less than 2^64 apart", ids[i-1], ids[i])
	}

	// a byte fixed per process or per second would show as a position that never changes
	for pos := range len(ids[0]) {
		fixed := true
		for _, id := range ids[1:] {
			if id[pos] != ids[0][pos] {
				fixed = false
				break
			}
		}
		assert.False(t, fixed, "position %d holds %q in all %d ids", pos, ids[0][pos], n)
	}
}

func TestService_Cleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CleanupFunc: func(context.Context, time.Duration) error {
				return nil
			},
		}

		svc := NewService(&store, ServiceParams{EditDuration: 20 * time.Millisecond})
		// cancel context after 2.1 cleanup TTLs
		ctx, cancel := context.WithTimeout(context.Background(), svc.EditDuration/100*15*21)
		defer cancel()
		svc.Cleanup(ctx)
		assert.Equal(t, 2, len(store.CleanupCalls()))
	})
}

func TestService_Submit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{ImageAPI: "/blah/", EditDuration: time.Millisecond * 100})
		svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
		assert.Equal(t, 3, len(store.ResetCleanupTimerCalls()))
		err := svc.Commit(func() []string { return []string{"id4", "id5"} })
		assert.NoError(t, err)
		svc.Submit(func() []string { return []string{"id6", "id7"} })
		assert.Equal(t, 5, len(store.ResetCleanupTimerCalls()))
		svc.Submit(nil)
		assert.Equal(t, 2, len(store.CommitCalls()))
		time.Sleep(time.Millisecond * 175)
		assert.Equal(t, 7, len(store.CommitCalls()))
		svc.Close(context.TODO())
	})
}

func TestService_Close(t *testing.T) {
	store := StoreMock{
		CommitFunc:            func(string) error { return nil },
		ResetCleanupTimerFunc: func(string) error { return nil },
	}
	svc := NewService(&store, ServiceParams{ImageAPI: "/blah/", EditDuration: time.Hour * 24})
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	assert.Equal(t, 5, len(store.ResetCleanupTimerCalls()))
	svc.Close(context.TODO())
	assert.Equal(t, 5, len(store.CommitCalls()))
}

func TestService_SubmitDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc: func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error {
				return nil
			},
		}
		svc := NewService(&store, ServiceParams{EditDuration: 20 * time.Millisecond})
		svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
		time.Sleep(150 * time.Millisecond) // let first batch to pass TTL
		svc.Submit(func() []string { return []string{"id4", "id5"} })
		svc.Submit(nil)
		assert.Equal(t, 5, len(store.ResetCleanupTimerCalls()))
		assert.Equal(t, 3, len(store.CommitCalls()))
		svc.Close(context.TODO())
		assert.Equal(t, 5, len(store.CommitCalls()))
	})
}

func TestService_SubmitNeverBlocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
		const submits = 10000 // all inside one edit window, so none of them is committed while the loop runs
		start := time.Now()
		for i := range submits {
			svc.Submit(func() []string { return []string{strconv.Itoa(i)} })
		}
		assert.Zero(t, time.Since(start), "Submit does not wait for earlier submits to pass EditDuration")
		committed := len(store.CommitCalls())
		assert.Zero(t, committed, "nothing is committed before EditDuration")
		svc.Close(context.TODO())
		commits := store.CommitCalls()
		require.Len(t, commits, submits, "Close commits every queued submit")
		for i, c := range commits {
			require.Equal(t, strconv.Itoa(i), c.ID, "commits keep the order of submits")
		}
	})
}

func TestService_SubmitWakesIdleSubmitter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
		svc.Submit(func() []string { return []string{"id1"} })
		time.Sleep(2 * time.Hour) // id1 is committed and the submitter waits for the next submit
		require.Len(t, store.CommitCalls(), 1)
		svc.Submit(func() []string { return []string{"id2"} })
		time.Sleep(2 * time.Hour)
		assert.Len(t, store.CommitCalls(), 2, "a submit made while the submitter is idle is committed after EditDuration, not at Close")
		svc.Close(context.TODO())
		assert.Len(t, store.CommitCalls(), 2)
	})
}

func TestService_Info(t *testing.T) {
	store := StoreMock{InfoFunc: func() (StoreInfo, error) {
		return StoreInfo{}, nil
	}}

	svc := Service{store: &store, ServiceParams: ServiceParams{}}
	info, err := svc.Info()
	assert.NoError(t, err)
	assert.True(t, info.FirstStagingImageTS.IsZero())
	assert.Equal(t, 1, len(store.InfoCalls()))
}

func TestService_resize(t *testing.T) {
	t.Run("empty data returns nil", func(t *testing.T) {
		assert.Nil(t, resize(nil, 100, 100))
		assert.Nil(t, resize([]byte{}, 100, 100))
	})

	t.Run("non-image bytes are refused", func(t *testing.T) {
		// previously resize would fall back to the raw bytes on decode failure, letting
		// attacker-controlled non-image content reach the store. After hardening, refuse.
		assert.Nil(t, resize([]byte("some picture bin data"), -1, -1))
		assert.Nil(t, resize([]byte("invalid image content"), 100, 100))
	})

	cases := []struct {
		file   string
		wr, hr int
	}{
		{"testdata/circles.png", 400, 300}, // full size: 800x600 px
		{"testdata/circles.jpg", 300, 400}, // full size: 600x800 px
	}

	for _, c := range cases {
		img, err := os.ReadFile(c.file)
		require.NoError(t, err, "can't open test file %s", c.file)

		// no need for resize, image dimensions are smaller than resize limit
		resized := resize(img, 800, 800)
		assert.NotNil(t, resized, "file %s", c.file)
		assert.Equal(t, resized, img)

		// resizing to half of width
		resized = resize(img, 400, 400)
		assert.NotNil(t, resized, "file %s", c.file)
		imgRz, format, err := image.Decode(bytes.NewBuffer(resized))
		assert.NoError(t, err, "file %s", c.file)
		assert.Equal(t, "png", format, "file %s", c.file)
		bounds := imgRz.Bounds()
		assert.Equal(t, c.wr, bounds.Dx(), "file %s", c.file)
		assert.Equal(t, c.hr, bounds.Dy(), "file %s", c.file)
	}
}

// TestService_SaveWithIDShortPayload guards readAndValidateImage from panicking
// on a body shorter than 512 bytes — historically it sliced data[:512] without
// a bounds check, which would panic before any decode-bomb defense could fire.
func TestService_SaveWithIDShortPayload(t *testing.T) {
	short := []byte("not an image")

	svc := Service{ImageAPI: "/blah/", MaxSize: 1500, MaxWidth: 32, MaxHeight: 32}
	err := svc.SaveWithID("test_id", bytes.NewReader(short))
	require.Error(t, err, "short non-image body must return an error, not panic")
	assert.Contains(t, err.Error(), "file format not allowed")
}

// TestService_SaveWithIDWebP confirms that WebP — listed as an allowed format
// in readAndValidateImage — still round-trips through prepareImage now that
// resize() runs image.DecodeConfig. Without registering the WebP decoder, a
// legitimate WebP upload would fail DecodeConfig and prepareImage would error.
func TestService_SaveWithIDWebP(t *testing.T) {
	webp, err := os.ReadFile("testdata/pixel.webp")
	require.NoError(t, err)

	// sanity: the fixture must be a well-formed 1x1 WebP that DecodeConfig accepts.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(webp))
	require.NoError(t, err)
	require.Equal(t, "webp", format)
	require.Equal(t, 1, cfg.Width)
	require.Equal(t, 1, cfg.Height)

	store := StoreMock{SaveFunc: func(string, []byte) error { return nil }}
	svc := Service{store: &store, MaxSize: 1500}

	err = svc.SaveWithID("webp_id", bytes.NewReader(webp))
	require.NoError(t, err, "valid WebP must round-trip through SaveWithID")
	assert.Equal(t, 1, len(store.SaveCalls()))
	assert.Equal(t, webp, store.SaveCalls()[0].Img, "no-resize path must return bytes verbatim")
}

// TestService_resizeRejectsDecompressionBomb verifies the dimension-cap defense.
// Builds a tiny GIF that declares 65535x65535 (4 gigapixels) in its logical-screen
// header — image.DecodeConfig reads the dimensions, the int64 product overflows
// any 32-bit int wrap, and resize must refuse before image.Decode allocates ~17 GB
// of pixel memory.
func TestService_resizeRejectsDecompressionBomb(t *testing.T) {
	// minimal GIF87a header with 65535x65535 logical screen, no global color table.
	// Bytes 6-7 are the little-endian width, 8-9 are the little-endian height.
	bomb := []byte{
		'G', 'I', 'F', '8', '7', 'a',
		0xFF, 0xFF,
		0xFF, 0xFF,
		0x00,
		0x00,
		0x00,
		0x3B,
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(bomb))
	require.NoError(t, err, "bomb header must decode at the config level")
	assert.Equal(t, 65535, cfg.Width)
	assert.Equal(t, 65535, cfg.Height)

	assert.Nil(t, resize(bomb, 100, 100), "resize must refuse oversized dimensions before raster decode")
	assert.Nil(t, resize(bomb, 0, 0), "even with no-resize limits, oversized dims must be refused")

	// integration-level: SaveWithID must reject the same bomb without panicking
	// or allocating gigabytes of raster memory.
	store := StoreMock{SaveFunc: func(string, []byte) error { return nil }}
	svc := Service{store: &store, MaxSize: 1500}
	err = svc.SaveWithID("bomb_id", bytes.NewReader(bomb))
	require.Error(t, err, "SaveWithID must reject decompression bomb")
	assert.Equal(t, 0, len(store.SaveCalls()), "rejected bomb must not be stored")
}

// TestService_resizeRejectsManyScanJPEG verifies the scan cap. A progressive jpeg may
// repeat a scan any number of times, a scan that adds nothing takes a dozen bytes, and
// image.Decode passes over every block of the picture for each of them, so resize must
// refuse the file before image.Decode spends that time.
func TestService_resizeRejectsManyScanJPEG(t *testing.T) {
	allowed, bomb := manyScanJPEG(maxJPEGScans), manyScanJPEG(maxJPEGScans+1)

	// sanity: both files are well-formed jpegs, only the number of scans gets one refused
	for _, data := range [][]byte{allowed, bomb} {
		_, format, err := image.Decode(bytes.NewReader(data))
		require.NoError(t, err, "the file must decode when the scan cap is not applied")
		require.Equal(t, "jpeg", format)
	}

	assert.Equal(t, allowed, resize(allowed, 100, 100), "jpeg with the allowed number of scans must pass")
	assert.Nil(t, resize(bomb, 100, 100), "resize must refuse a jpeg with too many scans")
	assert.Nil(t, resize(bomb, 0, 0), "even with no-resize limits, too many scans must be refused")

	// the scans are counted before the full decode: refusing a picture of the largest allowed size must not
	// allocate the memory a decode of it takes, 64 MB for the coefficients of a progressive 4096x4096 picture
	large := manyScanJPEGSized(maxJPEGScans+1, 4096, 4096)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	refused := resize(large, 100, 100)
	runtime.ReadMemStats(&after)
	assert.Nil(t, refused)
	assert.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(8<<20), "refused before the full decode starts")

	// integration-level: SaveWithID refuses the file with the error a client already gets
	// for a picture over the pixel limit, and stores nothing
	store := StoreMock{SaveFunc: func(string, []byte) error { return nil }}
	svc := Service{store: &store, MaxSize: 1500}
	err := svc.SaveWithID("bomb_id", bytes.NewReader(bomb))
	require.EqualError(t, err, "image rejected: malformed or exceeds 16777216-pixel safe limit")
	assert.Empty(t, store.SaveCalls(), "rejected file must not be stored")

	require.NoError(t, svc.SaveWithID("allowed_id", bytes.NewReader(allowed)))
	require.Len(t, store.SaveCalls(), 1)
	assert.Equal(t, allowed, store.SaveCalls()[0].Img)
}

// manyScanJPEG builds an 8x8 grayscale progressive jpeg with the given number of scans,
// each of them an AC refinement scan that changes nothing. The standard library can't
// encode a progressive jpeg, so the segments are written by hand.
func manyScanJPEG(scans int) []byte { return manyScanJPEGSized(scans, 8, 8) }

// manyScanJPEGSized is manyScanJPEG with the declared picture size set. The scans cover one
// block only, so a size above 8x8 gives a file whose header is right and whose scans are not
// enough for the picture.
func manyScanJPEGSized(scans, width, height int) []byte {
	segment := func(marker byte, payload ...byte) []byte {
		return append([]byte{0xff, marker, 0, byte(len(payload) + 2)}, payload...)
	}

	res := []byte{0xff, 0xd8} // SOI
	// DQT: 8-bit table 0, all ones
	res = append(res, segment(0xdb, append([]byte{0x00}, bytes.Repeat([]byte{1}, 64)...)...)...)
	// SOF2 (progressive): 8-bit samples, 8x8, one component without subsampling, table 0
	res = append(res, segment(0xc2, 8, byte(height>>8), byte(height), byte(width>>8), byte(width), 1, 1, 0x11, 0)...)
	// DHT: AC table 0, its only code is one bit long and means "end of band" for a run of blocks
	res = append(res, segment(0xc4, append(append([]byte{0x10, 1}, make([]byte, 15)...), 0xe0)...)...)
	for range scans {
		res = append(res, segment(0xda, 1, 1, 0x00, 1, 63, 0x10)...) // SOS: refine coefficients 1..63
		res = append(res, 0, 0)                                      // one end-of-band run, covers the only block
	}
	return append(res, 0xff, 0xd9) // EOI
}

// baselineJPEG encodes a 16x16 grayscale picture with the standard library encoder. The
// pattern is busy enough for the entropy-coded data to hold stuffed ff bytes.
func baselineJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = byte(i * i * 37)
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}))
	require.True(t, bytes.Contains(buf.Bytes(), []byte{0xff, 0x00}), "the encoded file must hold a stuffed byte")
	return buf.Bytes()
}

func TestJpegScans(t *testing.T) {
	baseline := baselineJPEG(t)
	circles, err := os.ReadFile("testdata/circles.jpg")
	require.NoError(t, err)

	// a comment segment right after SOI with bytes that look like 200 scans and an end of image
	comment := append([]byte{0xff, 0xfe, 0x01, 0x94}, bytes.Repeat([]byte{0xff, 0xda}, 200)...)
	comment = append(comment, 0xff, 0xd9)
	commented := append([]byte{0xff, 0xd8}, append(comment, manyScanJPEG(3)[2:]...)...)

	// a scan header the walker skips by its length, so the bytes after it are entropy-coded data
	soi, sos, eoi := []byte{0xff, 0xd8}, []byte{0xff, 0xda, 0x00, 0x02}, []byte{0xff, 0xd9}
	join := func(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

	tbl := []struct {
		name  string
		data  []byte
		scans int
	}{
		{"baseline from the standard library encoder", baseline, 1},
		{"baseline file from another encoder", circles, 1},
		{"progressive with many scans", manyScanJPEG(42), 42},
		{"markers inside a comment segment are not counted", commented, 3},
		// what follows EOI reads as a segment of a length that fits and then as 200 scans, so only the stop at
		// EOI keeps them out of the count
		{"data after the end of image is not counted",
			append(append(manyScanJPEG(3), 0x00, 0x02), bytes.Repeat([]byte{0xff, 0xda, 0x00, 0x02}, 200)...), 3},
		{"stuffed byte in a scan", join(soi, sos, []byte{0x5a, 0xff, 0x00}, sos, eoi), 2},
		{"restart marker in a scan", join(soi, sos, []byte{0x5a, 0xff, 0xd0, 0x34}, sos, eoi), 2},
		{"fill bytes before a marker", join(soi, sos, []byte{0x5a, 0xff, 0xff}, sos, eoi), 2},
		{"extraneous bytes between segments", join(soi, []byte{0x01, 0x02}, sos, eoi), 1},
		{"segment length below two", join(soi, []byte{0xff, 0xda, 0x00, 0x00}, sos, sos, eoi), 1},
		{"segment length past the end", join(soi, []byte{0xff, 0xda, 0xff, 0xff}, sos, eoi), 1},
		{"cut in the segment length", join(soi, []byte{0xff, 0xda, 0x00}), 1},
		{"soi only", soi, 0},
		{"empty", nil, 0},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.scans, jpegScans(tt.data))
		})
	}

	// the segments the walker skips are skipped by the decoder too, the picture stays acceptable
	_, _, err = image.Decode(bytes.NewReader(commented))
	require.NoError(t, err)
	assert.Equal(t, commented, resize(commented, 100, 100), "marker-like bytes in a comment must not get a jpeg refused")
	assert.Equal(t, baseline, resize(baseline, 100, 100), "an ordinary baseline jpeg must pass")

	many := manyScanJPEG(200)
	assert.Zero(t, testing.AllocsPerRun(10, func() { jpegScans(many) }), "the walk must not allocate")
}

// TestJpegScansTruncated cuts valid files at every length. The walker must stop without a
// panic and never count more than the whole file holds, and resize must refuse every cut
// file: the walker passing a cut file does not get it stored, the full decode refuses it.
func TestJpegScansTruncated(t *testing.T) {
	files := map[string][]byte{
		"baseline":          baselineJPEG(t),
		"progressive":       manyScanJPEG(5),
		"over the scan cap": manyScanJPEG(maxJPEGScans + 1),
	}
	for name, data := range files {
		t.Run(name, func(t *testing.T) {
			full := jpegScans(data)
			for n := range len(data) {
				cut := data[:n]
				assert.LessOrEqual(t, jpegScans(cut), full, "cut at %d", n)
				assert.Nil(t, resize(cut, 100, 100), "cut at %d", n)
			}
		})
	}
}

func TestGetProportionalSizes(t *testing.T) {
	tbl := []struct {
		inpW, inpH     int
		limitW, limitH int
		resW, resH     int
	}{
		{10, 20, 50, 25, 10, 20},
		{400, 200, 50, 25, 50, 25},
		{100, 100, 50, 25, 25, 25},
		{100, 200, 50, 25, 12, 25},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			resW, resH := getProportionalSizes(tt.inpW, tt.inpH, tt.limitW, tt.limitH)
			assert.Equal(t, tt.resW, resW, "width")
			assert.Equal(t, tt.resH, resH, "height")
		})
	}
}

func TestCachedImgID(t *testing.T) {
	img, err := CachedImgID(" http://foo.com")
	assert.Error(t, err)
	assert.Empty(t, img)
	imgURL := "http://example.org/img/1.png"
	img, err = CachedImgID(imgURL)
	assert.NoError(t, err)
	assert.Equal(t, "cached_images/"+Sha1Str("example.org")+"-"+Sha1Str(imgURL), img)
}

func TestService_DoubleClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
		svc.Submit(func() []string { return []string{"id1"} }) // starts the submitter, so Close has one to stop
		svc.Close(context.TODO())
		assert.NotPanics(t, func() { svc.Close(context.TODO()) }, "second Close")
		assert.Len(t, store.CommitCalls(), 1)
	})
}

func TestService_SubmitAfterClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
		svc.Submit(func() []string { return []string{"id1"} })
		svc.Close(context.TODO())
		start := time.Now()
		require.NotPanics(t, func() { svc.Submit(func() []string { return []string{"id2", "id3"} }) }, "Submit after Close")
		require.Len(t, store.CommitCalls(), 3, "Submit after Close commits at once")
		assert.Equal(t, "id2", store.CommitCalls()[1].ID)
		assert.Zero(t, time.Since(start), "Submit after Close does not wait for EditDuration")
	})
}

func TestService_CloseHonoursContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		store := StoreMock{
			CommitFunc:            func(string) error { <-release; return nil }, // a commit that stalls until released
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
		svc.Submit(func() []string { return []string{"id1"} })
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		svc.Close(ctx)
		assert.Equal(t, time.Minute, time.Since(start), "Close returns at the context deadline while a commit stalls")
		close(release)
		svc.Close(context.Background()) // waits for the released commit, so nothing outlives the bubble
		assert.Len(t, store.CommitCalls(), 1)
	})
}

func TestService_SubmitRacingClose(t *testing.T) {
	store := StoreMock{
		CommitFunc:            func(string) error { return nil },
		ResetCleanupTimerFunc: func(string) error { return nil },
	}
	svc := NewService(&store, ServiceParams{EditDuration: time.Hour})
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { svc.Submit(func() []string { return []string{strconv.Itoa(i)} }) })
	}
	svc.Close(context.TODO())
	wg.Wait()
	assert.Len(t, store.CommitCalls(), 100, "every submit is committed, whether it lands before or after Close")
}

// TestSafeImgContentType now lives in the rest package alongside the SafeImgContentType
// helper itself (see backend/app/rest/image_headers_test.go).
