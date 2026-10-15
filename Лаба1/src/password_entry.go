package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// MaskedEntry keeps the secret separate from Entry.Text. Fyne's built-in password
// entry uses bullets and offers disclosure; the assignment requires stars always.
// Navigation, insertion, deletion and paste operate on rune positions.
type MaskedEntry struct {
	widget.Entry
	secret []rune
}

func NewMaskedEntry() *MaskedEntry {
	e := &MaskedEntry{}
	e.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	e.ExtendBaseWidget(e)
	return e
}
func (e *MaskedEntry) Value() string { return string(e.secret) }
func (e *MaskedEntry) SetText(text string) {
	e.secret = []rune(text)
	e.syncMask(len(e.secret))
}

func (e *MaskedEntry) Clear() {
	clear(e.secret)
	e.secret = nil
	e.syncMask(0)
}

func (e *MaskedEntry) syncMask(cursor int) {
	e.Entry.SetText(strings.Repeat("*", len(e.secret)))
	e.CursorRow = 0
	e.CursorColumn = cursor
	e.Refresh()
}

func (e *MaskedEntry) insert(text string) {
	cursor := min(max(e.CursorColumn, 0), len(e.secret))
	added := []rune(text)
	next := make([]rune, 0, len(e.secret)+len(added))
	next = append(next, e.secret[:cursor]...)
	next = append(next, added...)
	next = append(next, e.secret[cursor:]...)
	clear(e.secret)
	e.secret = next
	e.syncMask(cursor + len(added))
}

func (e *MaskedEntry) TypedRune(r rune) {
	if !e.Disabled() {
		e.insert(string(r))
	}
}

func (e *MaskedEntry) TypedKey(event *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}
	cursor := min(max(e.CursorColumn, 0), len(e.secret))
	switch event.Name {
	case fyne.KeyBackspace:
		if cursor > 0 {
			e.secret = append(e.secret[:cursor-1], e.secret[cursor:]...)
			cursor--
		}
	case fyne.KeyDelete:
		if cursor < len(e.secret) {
			e.secret = append(e.secret[:cursor], e.secret[cursor+1:]...)
		}
	case fyne.KeyLeft:
		cursor = max(cursor-1, 0)
	case fyne.KeyRight:
		cursor = min(cursor+1, len(e.secret))
	case fyne.KeyHome:
		cursor = 0
	case fyne.KeyEnd:
		cursor = len(e.secret)
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.OnSubmitted != nil {
			e.OnSubmitted(e.Value())
		}
		return
	default:
		return
	}
	e.syncMask(cursor)
}

func (e *MaskedEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if e.Disabled() {
		return
	}
	if paste, ok := shortcut.(*fyne.ShortcutPaste); ok {
		e.insert(paste.Clipboard.Content())
	}
	// Copy, cut, undo and select-all are intentionally unavailable for secrets.
}
func (e *MaskedEntry) Dragged(*fyne.DragEvent)             {}
func (e *MaskedEntry) DragEnd()                            {}
func (e *MaskedEntry) DoubleTapped(event *fyne.PointEvent) { e.Tapped(event) }

// Intercept mouse input above Entry's internal entryContent: that child otherwise
// forwards drag directly to embedded Entry and bypasses our Dragged override.
func (e *MaskedEntry) CreateRenderer() fyne.WidgetRenderer {
	base := e.Entry.CreateRenderer()
	overlay := &maskedMouseLayer{entry: e}
	overlay.ExtendBaseWidget(overlay)
	return &maskedRenderer{WidgetRenderer: base, overlay: overlay}
}
func (e *MaskedEntry) KeyDown(*fyne.KeyEvent)           {}
func (e *MaskedEntry) KeyUp(*fyne.KeyEvent)             {}
func (e *MaskedEntry) TappedSecondary(*fyne.PointEvent) {}

type maskedMouseLayer struct {
	widget.BaseWidget
	entry *MaskedEntry
}

func (m *maskedMouseLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (m *maskedMouseLayer) Tapped(*fyne.PointEvent) {
	if !m.entry.Disabled() {
		fyne.CurrentApp().Driver().CanvasForObject(m.entry).Focus(m.entry)
	}
}
func (m *maskedMouseLayer) MouseDown(event *desktop.MouseEvent) { m.entry.MouseDown(event) }
func (m *maskedMouseLayer) MouseUp(event *desktop.MouseEvent)   { m.entry.MouseUp(event) }
func (m *maskedMouseLayer) Dragged(*fyne.DragEvent)             {}
func (m *maskedMouseLayer) DragEnd()                            {}
func (m *maskedMouseLayer) DoubleTapped(*fyne.PointEvent)       {}
func (m *maskedMouseLayer) TappedSecondary(*fyne.PointEvent)    {}

type maskedRenderer struct {
	fyne.WidgetRenderer
	overlay *maskedMouseLayer
}

func (r *maskedRenderer) Objects() []fyne.CanvasObject {
	return append(append([]fyne.CanvasObject{}, r.WidgetRenderer.Objects()...), r.overlay)
}

func (r *maskedRenderer) Layout(size fyne.Size) {
	r.WidgetRenderer.Layout(size)
	r.overlay.Resize(size)
}
