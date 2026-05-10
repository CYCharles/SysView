package gui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// LoadingScreen creates a centered loading animation with title and status text
func LoadingScreen(title, initialStatus string) (screen *fyne.Container, setStatus func(string)) {
	statusLabel := widget.NewLabelWithStyle(initialStatus, fyne.TextAlignCenter, fyne.TextStyle{})
	statusLabel.Importance = widget.MediumImportance

	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Animated loading dots
	dotLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(dotLabel),
		widget.NewLabel(""),
		statusLabel,
		layout.NewSpacer(),
	)

	screen = container.NewPadded(container.New(&centerLayout{}, container.NewVBox(
		titleLabel,
		content,
	)))

	setStatus = func(text string) {
		statusLabel.SetText(text)
		canvas.Refresh(statusLabel)
	}

	// Animate: cycling dots
	go func() {
		dots := []string{".  ", ".. ", "...", " ..", "  .", "   "}
		di := 0
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			dotLabel.SetText(dots[di])
			canvas.Refresh(dotLabel)
			di++
			if di >= len(dots) {
				di = 0
			}
		}
	}()

	return screen, setStatus
}

// centerLayout centers its single child
type centerLayout struct{}

func (c *centerLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		size := o.MinSize()
		if size.Width > w {
			w = size.Width
		}
		if size.Height > h {
			h = size.Height
		}
	}
	return fyne.NewSize(w+40, h+20)
}

func (c *centerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		oSize := o.MinSize()
		o.Resize(oSize)
		o.Move(fyne.NewPos(
			(size.Width-oSize.Width)/2,
			(size.Height-oSize.Height)/2,
		))
	}
}
