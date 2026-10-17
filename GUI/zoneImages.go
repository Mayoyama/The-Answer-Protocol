package main

import (
	"bytes"
	"embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// Screen holds the zone background: the image stack and the image widget inside it.
type Screen struct {
	zoneStack *fyne.Container
	zoneImage *canvas.Image
}

// zoneList holds the bare zone keys from world.yaml, one per embedded image.
var zoneList = []string{
	"taverne",
	"fountain_square",
	"chapel",
	"churchyard",
	"old_well",
	"cistern",
	"fallow_field",
	"market_square",
	"smithy",
	"mill",
}

// zoneImages maps a zone key to its decoded background image.
// It is filled once by loadZoneImages and only read afterwards.
var zoneImages = make(map[string]image.Image)

// embeddedFiles holds the zone background PNGs, compiled into the program
// so they are found regardless of the working directory.
//
//go:embed assets/chapel.png
//go:embed assets/churchyard.png
//go:embed assets/cistern.png
//go:embed assets/fallow_field.png
//go:embed assets/fountain_square.png
//go:embed assets/market_square.png
//go:embed assets/mill.png
//go:embed assets/old_well.png
//go:embed assets/smithy.png
//go:embed assets/taverne.png
var embeddedFiles embed.FS

// loadZoneImages decodes the embedded PNG of every zone in zoneList into zoneImages.
// It returns one error per image that could not be read or decoded; those
// zones simply have no entry and fall back to the plain colour.
func loadZoneImages() []error {
	var (
		img  image.Image
		errs []error
	)

	for _, z := range zoneList {
		path := fmt.Sprintf("assets/%s.png", z)
		imgFile, err := embeddedFiles.ReadFile(path)

		if err != nil {
			errs = append(errs, err)

			continue
		}

		if img, err = png.Decode(bytes.NewReader(imgFile)); err != nil {
			pngErr := fmt.Errorf("%s PNG error: %w", z, err)
			errs = append(errs, pngErr)

			continue
		}

		zoneImages[z] = img
	}

	return errs
}

// buildNewZoneImage builds the zone background: a plain colour with the zone image on top,
// both sized w × h. The image starts hidden.
func buildNewZoneImage(w, h float32) *Screen {
	backgroundRect := canvas.NewRectangle(color.RGBA{R: 40, G: 40, B: 50, A: 255})
	backgroundRect.SetMinSize(fyne.NewSize(w, h))

	imgWidget := canvas.NewImageFromImage(nil)
	imgWidget.ScaleMode = canvas.ImageScalePixels
	imgWidget.FillMode = canvas.ImageFillContain
	imgWidget.SetMinSize(fyne.NewSize(w, h))
	imgWidget.Hide()

	zStack := container.NewStack(backgroundRect, imgWidget)

	newScreen := &Screen{
		zoneStack: zStack,
		zoneImage: imgWidget,
	}

	return newScreen
}

// showZoneImage shows the background image for zoneID, or hides it (leaving the plain
// colour) if the zone has no image.
func (s *Screen) showZoneImage(zoneID string) {
	key := strings.TrimPrefix(zoneID, "zone.")

	img, ok := zoneImages[key]

	fyne.Do(func() {
		if ok {
			s.zoneImage.Image = img
			s.zoneImage.Refresh()
			s.zoneImage.Show()

		} else {
			s.zoneImage.Hide()
		}
	})
}
