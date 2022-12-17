package main

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ListItem struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	URL      string `json:"url"`
}

type List []ListItem

func (l *List) UpdateFunc(tl *tview.List) func() {
	return func() {
		tl.Clear()

		for _, item := range *l {
			tl.AddItem(item.Title, item.Subtitle, 0, nil)
			// lv.AddItem(item.Title, item.Subtitle, 0, func() {
			// 	app.Stop()
			// })
		}
	}
}

func (l *List) Run(ctx context.Context) (string, error) {
	app := tview.NewApplication()

	list := tview.NewList().
		SetSelectedFocusOnly(true)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			app.Stop()
		}

		return event
	})

	l.UpdateFunc(list)()

	if err := app.SetRoot(list, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}

	return "", nil
}
