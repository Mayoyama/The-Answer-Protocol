package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"slices"
)

type ChatWindow struct {
	TextWin *widget.RichText
	Scroll  *container.Scroll
}

func buildChatWindow(w, h float32) *ChatWindow {
	cw := &ChatWindow{}

	cw.TextWin = widget.NewRichText()
	cw.TextWin.Wrapping = fyne.TextWrapWord

	cw.Scroll = container.NewVScroll(cw.TextWin)
	cw.Scroll.SetMinSize(fyne.NewSize(w, h))

	return cw
}

func (cw *ChatWindow) addLine(line string) {
	atCWBottom := cw.notScrolledUp()

	cw.appendSegment(line)
	cw.dropOldLines()

	cw.TextWin.Refresh()
	cw.Scroll.Refresh()

	if atCWBottom {
		cw.Scroll.ScrollToBottom()
	}
}

func (cw *ChatWindow) notScrolledUp() bool {
	const tolerance float32 = 4

	if cw.Scroll.Offset.Y+cw.Scroll.Size().Height >= cw.Scroll.Content.MinSize().Height-tolerance {
		return true
	}

	return false
}

func (cw *ChatWindow) appendSegment(line string) {
	cw.TextWin.Segments = append(cw.TextWin.Segments, &widget.TextSegment{Text: line, Style: widget.RichTextStyleParagraph})
}

func (cw *ChatWindow) dropOldLines() {
	const chatHistoryMessageCap int = 250

	overflow := len(cw.TextWin.Segments) - chatHistoryMessageCap

	if overflow > 0 {
		cw.TextWin.Segments = slices.Delete(cw.TextWin.Segments, 0, overflow)
	}
}
