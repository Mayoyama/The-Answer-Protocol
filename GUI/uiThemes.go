package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

type SmallTextTheme struct {
	fyne.Theme
}

type ButtonColorTheme struct {
	fyne.Theme
	ButtonColor color.Color
}

func (stt SmallTextTheme) Size(adjustment fyne.ThemeSizeName) float32 {
	if adjustment == theme.SizeNameText {
		return 8
	}

	if adjustment == theme.SizeNameInnerPadding {
		return 4
	}

	return stt.Theme.Size(adjustment)
}

func (bct ButtonColorTheme) Color(fyneColor fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if fyneColor == theme.ColorNameButton {
		return bct.ButtonColor
	}

	return bct.Theme.Color(fyneColor, variant)
}
