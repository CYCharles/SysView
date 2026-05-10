package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// InfoRow represents a label-value pair for display
type InfoRow struct {
	Label string
	Value string
}

// InfoSection represents a group of related info rows with a title
type InfoSection struct {
	Title string
	Rows  []InfoRow
}

// MakeInfoPanel creates a scrollable panel with sectioned info rows
func MakeInfoPanel(sections []InfoSection) *container.Scroll {
	return makeInfoPanelWithHeader(sections, "")
}

// MakeInfoPanelWithHeader creates a scrollable panel with a prominent header
func MakeInfoPanelWithHeader(sections []InfoSection, headerText string) *container.Scroll {
	return makeInfoPanelWithHeader(sections, headerText)
}

func makeInfoPanelWithHeader(sections []InfoSection, headerText string) *container.Scroll {
	var objects []fyne.CanvasObject

	// Hero header - prominent device name at top
	if headerText != "" {
		header := widget.NewLabelWithStyle(headerText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		header.Wrapping = fyne.TextWrapWord
		objects = append(objects, header)
		objects = append(objects, widget.NewSeparator())
	}

	for i, section := range sections {
		if i > 0 {
			objects = append(objects, widget.NewSeparator())
		}

		// Section title
		title := widget.NewLabelWithStyle(section.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		objects = append(objects, title)

		// Form items for this section
		var items []*widget.FormItem
		for _, row := range section.Rows {
			valueLabel := widget.NewLabel(row.Value)
			valueLabel.Wrapping = fyne.TextWrapWord
			items = append(items, &widget.FormItem{
				Text:   row.Label,
				Widget: valueLabel,
			})
		}

		if len(items) > 0 {
			form := &widget.Form{Items: items}
			objects = append(objects, form)
		}
	}

	content := container.NewVBox(objects...)
	return container.NewScroll(content)
}

// MakeCardPanel creates a panel with card-style sections
func MakeCardPanel(cards []InfoSection) *container.Scroll {
	var objects []fyne.CanvasObject

	for _, card := range cards {
		var items []*widget.FormItem
		for _, row := range card.Rows {
			valueLabel := widget.NewLabel(row.Value)
			valueLabel.Wrapping = fyne.TextWrapWord
			items = append(items, &widget.FormItem{
				Text:   row.Label,
				Widget: valueLabel,
			})
		}

		var content fyne.CanvasObject
		if len(items) > 0 {
			content = &widget.Form{Items: items}
		} else {
			content = widget.NewLabel("暂无数据")
		}

		cardWidget := widget.NewCard(card.Title, "", content)
		objects = append(objects, cardWidget)
	}

	vbox := container.NewVBox(objects...)
	return container.NewScroll(vbox)
}
