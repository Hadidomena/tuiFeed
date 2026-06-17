package bsk

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"

	sixel "github.com/mattn/go-sixel"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ImageProto int

const (
	ProtoNone ImageProto = iota
	ProtoKitty
	ProtoSixel
)

var detectedProto ImageProto = -1

func DetectImageProtocol() ImageProto {
	if detectedProto >= 0 {
		return detectedProto
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		detectedProto = ProtoKitty
		return detectedProto
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "WezTerm":
		detectedProto = ProtoKitty
		return detectedProto
	case "iTerm.app":
		detectedProto = ProtoSixel
		return detectedProto
	}

	term := os.Getenv("TERM")
	if term == "xterm-kitty" {
		detectedProto = ProtoKitty
		return detectedProto
	}

	detectedProto = ProtoSixel
	return detectedProto
}

func RenderTerminalImage(data []byte, yOffset int) error {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	img = resizeToFit(img, 40, 18)

	proto := DetectImageProtocol()

	switch proto {
	case ProtoKitty:
		return renderKitty(img, yOffset)
	case ProtoSixel:
		return renderSixel(img, yOffset)
	default:
		return RenderImageExternal(data)
	}
}

func RenderImageExternal(data []byte) error {
	f, err := os.CreateTemp("", "tuifeed-img-*.png")
	if err != nil {
		return fmt.Errorf("temp file failed: %w", err)
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(name)
		return fmt.Errorf("write temp failed: %w", err)
	}
	f.Close()

	return exec.Command("xdg-open", name).Start()
}

func resizeToFit(img image.Image, maxCols int, maxRows int) image.Image {
	bounds := img.Bounds()
	iw, ih := bounds.Dx(), bounds.Dy()

	cellW := 9
	cellH := 20

	targetW := maxCols * cellW
	targetH := maxRows * cellH

	scaleW := float64(targetW) / float64(iw)
	scaleH := float64(targetH) / float64(ih)

	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	if scale >= 1.0 {
		return img
	}

	newW := int(float64(iw) * scale)
	newH := int(float64(ih) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)
	return resized
}

func ClearImages() {
	if DetectImageProtocol() != ProtoKitty {
		return
	}
	os.Stdout.Write([]byte("\033_Ga=d,d=A\033\\"))
}

func renderKitty(img image.Image, yOffset int) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("png encode failed: %w", err)
	}
	imgBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	cols := img.Bounds().Dx() / 9
	rows := img.Bounds().Dy() / 20
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	y := yOffset + 1
	x := 1

	settings := fmt.Sprintf("a=T,t=d,f=100,X=%d,Y=%d,c=%d,r=%d,z=2,C=1,", x, y, cols, rows)

	var out bytes.Buffer
	kittyLimit := 4096
	lenB64 := len(imgBase64)

	for i := 0; i < (lenB64-1)/kittyLimit; i++ {
		chunk := imgBase64[i*kittyLimit : (i+1)*kittyLimit]
		out.WriteString(fmt.Sprintf("\033_G%sC=1,m=1;%s\033\\", settings, chunk))
		settings = ""
	}
	out.WriteString(fmt.Sprintf("\033_G%sC=1,m=0;%s\033\\", settings, imgBase64[((lenB64-1)/kittyLimit)*kittyLimit:]))

	_, err := os.Stdout.Write(out.Bytes())
	return err
}

func renderSixel(img image.Image, yOffset int) error {
	bounds := img.Bounds()
	cols := bounds.Dx() / 9
	if cols < 1 {
		cols = 1
	}

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Dither = true
	enc.Width = cols
	if err := enc.Encode(img); err != nil {
		return fmt.Errorf("sixel encode failed: %w", err)
	}

	y := yOffset + 1
	x := 1

	out := fmt.Sprintf("\033[%d;%dH\033[?8452h%s", y, x, buf.String())
	_, err := os.Stdout.Write([]byte(out))
	return err
}

func init() {
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("gif", "GIF8", gif.Decode, gif.DecodeConfig)
}
