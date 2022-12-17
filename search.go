package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type debounce struct {
	dur      time.Duration
	timer    *time.Timer
	callback func()
	cancel   context.CancelFunc
}

func (d *debounce) do(ctx context.Context, cb func(ctx context.Context)) {
	if d.timer != nil {
		d.timer.Stop()
		d.cancel()
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.callback = func() {
		cb(ctx)
	}

	d.timer = time.AfterFunc(d.dur, d.callback)
}

type Search struct {
	URL string `json:"url"`
}

func (s *Search) Run(ctx context.Context) (string, error) {
	app := tview.NewApplication()
	var listSpec *List

	inputField := tview.NewInputField().
		SetLabel("Search: ")
		// SetFieldWidth(10).
		// SetAcceptanceFunc(tview.InputFieldInteger).
		// SetDoneFunc(func(key tcell.Key) {
		// 	app.Stop()
		// })

	list := tview.NewList().
		SetSelectedFocusOnly(true)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if (event.Key() == tcell.KeyUp && list.GetCurrentItem() == 0) ||
			(event.Key() == tcell.KeyDown && list.GetCurrentItem() == list.GetItemCount()-1) {
			app.SetFocus(inputField)
			return nil
		} else if event.Key() == tcell.KeyEnter {
			app.Stop()
		}

		return event
	})

	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			app.SetFocus(list)
			list.SetCurrentItem(list.GetItemCount() - 1)
		case tcell.KeyDown:
			app.SetFocus(list)
			list.SetCurrentItem(0)
		}

		return event
	})

	var inputDebounce = debounce{
		dur: 200 * time.Millisecond,
	}

	inputField.SetChangedFunc(func(text string) {
		inputDebounce.do(ctx, func(ctx context.Context) {
			u, err := resolveURL(s.URL)
			if err != nil {
				panic(err)
			}

			log.Println("DEBUG " + u + text)

			req, err := http.NewRequestWithContext(ctx, "GET", u+url.PathEscape(text), nil)
			if err != nil {
				panic(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if errors.Is(err, context.Canceled) {
				log.Println("Debounce context canceled")
				return
			} else if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			bts, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			spec := Spec{}

			if err := json.Unmarshal(bts, &spec); err != nil {
				panic(err)
			}

			listSpec = spec.List

			app.QueueUpdateDraw(spec.List.UpdateFunc(list))
		})
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(inputField, 3, 1, false).
		AddItem(list, 0, 1, false)
	if err := app.SetRoot(flex, true).SetFocus(inputField).Run(); err != nil {
		panic(err)
	}

	return (*listSpec)[list.GetCurrentItem()].URL, nil
}
