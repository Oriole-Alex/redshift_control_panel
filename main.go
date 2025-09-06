package main

import (
	"context"
	"fmt"
	"image/color"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	debounce = 250 * time.Millisecond
	timeout  = 3 * time.Second
)

type uiState struct {
	tempK      *LabeledSlider
	brightness *LabeledSlider
	gamma      *LabeledSlider
	out        *widget.Label
	resetBtn   *widget.Button

	timer   *time.Timer
	cancel  context.CancelFunc
	silence bool // prevent handlers when changing sliders programmatically
}

// ---- Custom theme for app-wide background (#313131) ----

type bgTheme struct{ fyne.Theme }

func (t bgTheme) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		// #313131
		return color.NRGBA{R: 0x31, G: 0x31, B: 0x31, A: 0xFF}
	}
	return t.Theme.Color(name, v)
}

// -------------------------------------------------------

func main() {
	a := app.New()
	a.Settings().SetTheme(bgTheme{Theme: theme.DefaultTheme()})

	w := a.NewWindow("Screen Dimmer")
	w.Resize(fyne.NewSize(400, 320))

	out := widget.NewLabel("Ready.")

	// Build our reusable sliders
	temp := NewLabeledSlider("Temperature (K)", 1000, 10000, 100, 6500, "%.0f", "K")
	bright := NewLabeledSlider("Brightness", 0.10, 1.00, 0.01, 1.00, "%.2f", "")
	gamma := NewLabeledSlider("Gamma", 0.50, 2.50, 0.01, 1.00, "%.2f", "")

	u := &uiState{tempK: temp, brightness: bright, gamma: gamma, out: out}
u.resetBtn = widget.NewButtonWithIcon("Reset to defaults", theme.ViewRefreshIcon(), func() { go u.reset() })

	// Debounced live apply while dragging (snapshot values on UI thread)
	onChange := func() {
		if u.silence {
			return
		}
		t := int(temp.Value())
		b := bright.Value()
		g := gamma.Value()
		u.scheduleApply(t, b, g)
	}
	temp.SetOnChanged(func(_ float64) { onChange() })
	bright.SetOnChanged(func(_ float64) { onChange() })
	gamma.SetOnChanged(func(_ float64) { onChange() })

	// ----- Header bar (#494949) -----
	headerContent := container.NewHBox(u.resetBtn)

	headerBG := canvas.NewRectangle(color.NRGBA{R: 0x49, G: 0x49, B: 0x49, A: 0xFF}) // #494949
	header := container.NewStack(
		headerBG,
		container.NewPadded(headerContent), // nice inner spacing
	)

	// ----- Settings panel -----
	// Rounded panel with bg #414141, thin border #373737, and white dividers between items
	panelInner := container.NewVBox(
		bright.View(),
		thinDivider(color.White),
		temp.View(),
		thinDivider(color.White),
		gamma.View(),
	)
	panelPadded := container.NewPadded(panelInner)

	panelBG := newRoundRect(
		color.NRGBA{R: 0x41, G: 0x41, B: 0x41, A: 0xFF}, // fill #414141
		color.NRGBA{R: 0x37, G: 0x37, B: 0x37, A: 0xFF}, // stroke #373737
		1.0,                                              // stroke width
		10,                                               // corner radius
	)

	settingsPanel := container.NewMax(panelBG, panelPadded)

	// ----- Page content -----
	w.SetContent(container.NewVBox(
		header,
		container.NewPadded(settingsPanel),
		out,
	))

	if _, err := exec.LookPath("redshift"); err != nil {
		out.SetText("Error: 'redshift' not found in PATH. Install it (e.g., sudo apt install redshift).")
	}

	w.ShowAndRun()
}

func (u *uiState) scheduleApply(t int, b, g float64) {
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
	if u.timer != nil {
		u.timer.Stop()
	}
	u.timer = time.AfterFunc(debounce, func() {
		go u.apply(t, b, g)
	})
}

func (u *uiState) apply(t int, b, g float64) {
	args := []string{
		"-m", "randr", // force X11 method; avoids Wayland probe
		"-P",          // clear previous ramps so changes aren't compounded
		"-O", fmt.Sprintf("%d", t),
		"-g", fmt.Sprintf("%.2f:%.2f:%.2f", g, g, g),
		"-b", fmt.Sprintf("%.2f", b),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	u.cancel = cancel
	defer cancel()

	cmd := exec.CommandContext(ctx, "redshift", args...)
	outBytes, err := cmd.CombinedOutput()

	msg := strings.TrimSpace(string(outBytes))
	if ctx.Err() == context.DeadlineExceeded {
		msg = "Timed out applying settings."
	} else if err != nil && msg == "" {
		msg = "redshift error: " + err.Error()
	} else if err != nil {
		msg = "redshift error: " + msg
	} else if msg == "" {
		msg = "Applied."
	}
	fyne.Do(func() { u.out.SetText(msg) })
}

func (u *uiState) reset() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	u.cancel = cancel
	defer cancel()

	cmd := exec.CommandContext(ctx, "redshift", "-x")
	outBytes, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(outBytes))
	if err != nil && msg == "" {
		msg = "reset error: " + err.Error()
	} else if err != nil {
		msg = "reset error: " + msg
	} else {
		msg = "Reset to defaults."
	}

	fyne.Do(func() {
		u.silence = true
		u.tempK.SetValue(6500)
		u.brightness.SetValue(1.00)
		u.gamma.SetValue(1.00)
		u.silence = false
		u.out.SetText(msg)
	})
}

// ---------- helpers ----------

// thinDivider returns a full-width thin horizontal line with the given color.
func thinDivider(c color.Color) *fyne.Container {
	r := canvas.NewRectangle(c)
	r.SetMinSize(fyne.NewSize(0, 1)) // thin
	return container.NewMax(r)       // expand to full width
}

// newRoundRect builds a rounded rectangle canvas object with fill, stroke and radius.
// It expands automatically inside a container.NewMax(...).
func newRoundRect(fill, stroke color.Color, strokeWidth float32, radius float32) *canvas.Rectangle {
	r := canvas.NewRectangle(fill)
	// If your Fyne version supports rounded corners on Rectangle:
	// r.CornerRadius = radius
	// Otherwise, change to canvas.NewRoundedRectangle(...) if available in your Fyne version.
	r.StrokeColor = stroke
	r.StrokeWidth = strokeWidth
	// Best effort: if CornerRadius is supported, set it via a type assertion that won’t panic on older versions.
	type cornerable interface{ SetCornerRadius(float32) }
	if cr, ok := any(r).(cornerable); ok {
		cr.SetCornerRadius(radius)
	}
	return r
}
