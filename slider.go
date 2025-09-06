package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"image/color"

	"fyne.io/fyne/v2/canvas"
)

// LabeledSlider bundles a title label, a slider, and a value label.
type LabeledSlider struct {
	Label      *widget.Label
	Slider     *widget.Slider
	valueLabel *widget.Label
	minLabel   *widget.Label
	maxLabel   *widget.Label
	root       fyne.CanvasObject
	format     string
	unit       string
	onChange   func(v float64)
}

func NewLabeledSlider(title string, min, max, step, initial float64, format, unit string) *LabeledSlider {
	lbl := widget.NewLabel(title)

	s := widget.NewSlider(min, max)
	if step > 0 {
		s.Step = step
	}
	s.Value = initial

	val := widget.NewLabel("")
	val.Alignment = fyne.TextAlignTrailing

	minLbl := widget.NewLabel("")
	maxLbl := widget.NewLabel("")

	ls := &LabeledSlider{
		Label:      lbl,
		Slider:     s,
		valueLabel: val,
		minLabel:   minLbl,
		maxLabel:   maxLbl,
		format:     format,
		unit:       unit,
	}

	// Header: title on left, current value on right
	header := container.NewHBox(lbl, fixedSpacer(2), val)

	// Slider row: min at left, max at right, slider expands in the middle
	sliderRow := container.NewBorder(nil, nil, minLbl, maxLbl, s)

	ls.root = container.NewVBox(header, sliderRow)

	// Set initial texts & wire change handler
	ls.minLabel.SetText(ls.formatValue(min))
	ls.maxLabel.SetText(ls.formatValue(max))
	ls.updateValueLabel(initial)

	s.OnChanged = func(v float64) {
		ls.updateValueLabel(v)
		if ls.onChange != nil {
			ls.onChange(v)
		}
	}

	return ls
}

// View returns the root container.
func (ls *LabeledSlider) View() fyne.CanvasObject { return ls.root }

// SetOnChanged registers a callback fired after the value label updates.
func (ls *LabeledSlider) SetOnChanged(fn func(v float64)) { ls.onChange = fn }

// Value returns the current slider value.
func (ls *LabeledSlider) Value() float64 { return ls.Slider.Value }

// SetValue sets the slider and updates the label.
func (ls *LabeledSlider) SetValue(v float64) {
	ls.Slider.SetValue(v)
	ls.updateValueLabel(v)
}

func (ls *LabeledSlider) formatValue(v float64) string {
	text := fmt.Sprintf(ls.format, v)
	if ls.unit != "" {
		text += " " + ls.unit
	}
	return text
}

func (ls *LabeledSlider) updateValueLabel(v float64) {
	ls.valueLabel.SetText(ls.formatValue(v))
}

func fixedSpacer(w float32) fyne.CanvasObject {
    r := canvas.NewRectangle(color.NRGBA{0, 0, 0, 0}) // transparent
    r.SetMinSize(fyne.NewSize(w, 0))
    return r
}