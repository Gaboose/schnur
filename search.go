package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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

func (s *Search) Run(ctx context.Context, loader *Loader) (string, error) {
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
			log.Println("DEBUG " + s.URL + url.PathEscape(text))

			rc, err := loader.Load(ctx, s.URL+url.PathEscape(text))
			if errors.Is(err, context.Canceled) {
				log.Println("Debounce context canceled")
				return
			} else if err != nil {
				panic(err)
			}
			defer rc.Close()

			spec := Spec{}

			if err := json.Unmarshal(rc.Bytes, &spec); err != nil {
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
