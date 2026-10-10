package main

import (
	//"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	login_win_w       float32 = 400
	login_win_h       float32 = 200
	backdrop_w        float32 = 1280
	backdrop_h        float32 = 720
	element_padding_h float32 = 180
)

func main() {
	guiApp := app.New()
	win := guiApp.NewWindow("The Answer Protocol")

	loginRequests := make(chan string, 1)
	tapConnRequests := make(chan string, 16)
	quitRequest := make(chan struct{}, 1)

	win.SetCloseIntercept(func() {
		select {
		case quitRequest <- struct{}{}:
		default:
		}
	})

	if errors := loadZoneImages(); len(errors) != 0 {
		_, _ = fmt.Println("Errors while loading zone images:")

		for i, err := range errors {
			_, _ = fmt.Printf("%d: %v\n", i, err)
		}
	}

	login := buildLoginScreen(loginRequests)
	game := buildGameScreen(tapConnRequests)

	win.SetContent(login.Content)
	win.Resize(fyne.NewSize(login_win_w, login_win_h))

	cr := &CommandRouter{
		Win:             win,
		Login:           login,
		Game:            game,
		LoginRequests:   loginRequests,
		TapConnRequests: tapConnRequests,
		QuitRequest:     quitRequest,
	}

	go cr.run()

	win.ShowAndRun()
}
