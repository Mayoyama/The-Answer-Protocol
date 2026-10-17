package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

// SmallTextTheme is the default theme with smaller text and inner padding, for small buttons.
type SmallTextTheme struct {
	fyne.Theme
}

// ButtonColorTheme is the default theme with ButtonColor as the button background.
type ButtonColorTheme struct {
	fyne.Theme
	ButtonColor color.Color
}

// Size returns the smaller text and inner padding sizes, and the embedded theme's
// sizes for everything else.
func (stt SmallTextTheme) Size(adjustment fyne.ThemeSizeName) float32 {
	if adjustment == theme.SizeNameText {
		return 8
	}

	if adjustment == theme.SizeNameInnerPadding {
		return 4
	}

	return stt.Theme.Size(adjustment)
}

// Color returns ButtonColor for button backgrounds, and the embedded theme's colours for
// everything else.
func (bct ButtonColorTheme) Color(fyneColor fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if fyneColor == theme.ColorNameButton {
		return bct.ButtonColor
	}

	return bct.Theme.Color(fyneColor, variant)
}
