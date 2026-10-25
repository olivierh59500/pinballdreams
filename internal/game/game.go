// Package game connects the shared native session to Ebitengine.
package game

import (
	"image"
	"io"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
	"github.com/olivierh59500/pinballdreams/internal/touchui"
)

type Game struct {
	Session                   *session.Session
	Touch                     bool
	height                    int
	image                     *ebiten.Image
	frame                     *image.RGBA
	touchCanvas               *image.RGBA
	touches                   []ebiten.TouchID
	contacts                  []pointerContact
	keys                      []ebiten.Key
	pressedKeys, releasedKeys []ebiten.Key
	quitKeys                  quitKeys
	controls                  inputState
	navigation                touchNavigation
	pending                   pendingInput
	initialsEvents            []session.InitialsAction
	rootRequest               struct{ pending, scores bool }
	clock                     timing.HostClock
	mouseSpring               mouseSpring
	audio                     audioPlayback
	context                   *audioDevice
	back                      bool
	characters                []rune
	focused                   bool
	mouseCaptured             bool
	layoutWidth, layoutHeight int
}

func New(source fs.FS, touch bool) (*Game, error) {
	if source == nil {
		source = assets.Files()
	}
	s, err := session.New(source)
	if err != nil {
		return nil, err
	}
	return &Game{Session: s, Touch: touch, height: 256}, nil
}

func NewWithStorage(source fs.FS, touch bool, directory string) (*Game, error) {
	g, err := New(source, touch)
	if err != nil {
		return nil, err
	}
	if directory != "" {
		if err := g.Session.ConfigureScores(directory); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func (g *Game) Update() error {
	if g.context != nil {
		if err := g.context.error(); err != nil {
			return err
		}
	}
	if !g.updateFocus(ebiten.IsFocused()) {
		g.captureMouse(false)
		return nil
	}
	g.captureMouse(g.tableMouseEnabled())
	g.audio.attach(g.Session)
	g.keys = inpututil.AppendPressedKeys(g.keys[:0])
	g.pressedKeys = inpututil.AppendJustPressedKeys(g.pressedKeys[:0])
	g.releasedKeys = inpututil.AppendJustReleasedKeys(g.releasedKeys[:0])
	left, right := flipperKeys(ebiten.IsKeyPressed)
	i := session.Input{
		Left: left, Right: right,
		Launch: ebiten.IsKeyPressed(ebiten.KeyArrowDown), Nudge: ebiten.IsKeyPressed(ebiten.KeySpace),
		PauseKeyPressed: inpututil.IsKeyJustPressed(ebiten.KeyP),
		Back:            ebiten.IsKeyPressed(ebiten.KeyEscape) || g.back && g.Session.Table == nil,
		MusicPressed:    inpututil.IsKeyJustReleased(ebiten.KeyM),
		ExitTable:       (g.back || g.Touch && inpututil.IsKeyJustPressed(ebiten.KeyEscape)) && g.Session.Table != nil,
		Start:           ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeyNumpadEnter),
	}
	i.HighScores = ebiten.IsKeyPressed(ebiten.KeySpace)
	i.Backspace = ebiten.IsKeyPressed(ebiten.KeyBackspace)
	if g.Session.Playing && !g.Session.Paused && (g.Session.InChute || g.Session.PreparingBall()) {
		// The original keyboard cancels its held-ball loop on Escape release.
		i.Back = inpututil.IsKeyJustReleased(ebiten.KeyEscape)
	}
	if g.Session.Paused {
		for _, key := range g.keys {
			i.AnyKey = i.AnyKey || inpututil.IsKeyJustPressed(key)
		}
	}
	if g.Session.Table == nil && g.Session.Intro == nil {
		for _, key := range g.keys {
			if inpututil.IsKeyJustPressed(key) {
				i.MenuKey = menuLetter(key)
			}
		}
	}
	if g.Session.AskingToQuit() {
		if !g.Session.Paused {
			i.Pause = false
		}
	}
	g.quitKeys.apply(&i, g.Session.AskingToQuit(), g.Session.Paused, g.pressedKeys, g.releasedKeys)
	if g.Session.NameEntry != nil && !g.Session.Paused {
		g.characters = ebiten.AppendInputChars(g.characters[:0])
		i.Text = string(g.characters)
		i.Left = ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
		i.Right = ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	}
	g.back = false
	applyStartKeys(g.Session, &i, ebiten.IsKeyPressed, inpututil.IsKeyJustPressed)
	g.touches = ebiten.AppendTouchIDs(g.touches[:0])
	g.contacts = g.contacts[:0]
	for _, id := range g.touches {
		x, y := ebiten.TouchPosition(id)
		g.contacts = append(g.contacts, pointerContact{id: pointerID{touch: id}, x: x, y: y, pressed: inpututil.TouchPressDuration(id) == 1})
	}
	primaryMouse := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if primaryMouse && !g.tableMouseEnabled() {
		x, y := ebiten.CursorPosition()
		g.contacts = append(g.contacts, pointerContact{id: pointerID{mouse: true}, x: x, y: y, pressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)})
	}
	rightMouse := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	i.IntroExtraCredits = g.Session.Intro != nil && rightMouse
	if handled, err := g.handleTouchNavigation(i, len(g.keys) != 0 || primaryMouse || rightMouse); handled || err != nil {
		return err
	}
	g.applyTouchScreenControls(&i)
	wasWaiting := g.controls.waitForRelease
	i = g.controls.apply(g.Session, i, g.contacts, len(g.keys) != 0 || primaryMouse || rightMouse, g.height)
	if g.Touch && g.Session.Table != nil && (i.Pause || i.PauseKeyPressed) {
		g.openTouchNavigation()
		return nil
	}
	if wasWaiting || g.controls.waitForRelease {
		g.pending.reset()
	}
	_, mouseY := ebiten.CursorPositionF()
	g.mouseSpring.apply(g.Session, &i, mouseY, primaryMouse, !g.tableMouseEnabled() || g.controls.waitForRelease || !ebiten.IsFocused())
	g.pending.add(i, g.pressedKeys, g.initialsEvents...)
	next, ready := g.nextSessionInput()
	if !ready {
		return nil
	}
	i = next
	defer g.pending.finishFrame()
	previousEditor := g.Session.NameEntry
	resumingQuestion := g.Session.Paused && g.Session.AskingToQuit()
	if err := g.Session.Update(i); err != nil {
		return err
	}
	if g.Session.NameEntry != previousEditor {
		// A finished entry must not donate queued edits to the next player.
		g.pending.initials = g.pending.initials[:0]
	}
	if resumingQuestion && !g.Session.Paused && g.Session.AskingToQuit() {
		// Releases already sampled before this logical step have been consumed
		// by the pause. Only still-held resume keys need a future release barrier.
		g.pressedKeys = g.pressedKeys[:0]
		for _, key := range g.pending.keys {
			if ebiten.IsKeyPressed(key) {
				g.pressedKeys = append(g.pressedKeys, key)
			}
		}
		g.quitKeys.consumeResume(g.pressedKeys)
	}
	g.captureMouse(g.tableMouseEnabled())
	if err := g.audio.update(g.Session.Sound, g.Session.AudioOutputPaused(), g.newAudioStream); err != nil {
		return err
	}
	g.Session.AudioChanged = false
	return nil
}

// Captured pointer coordinates keep accumulating beyond the window edges,
// allowing a full pull. Menus, pause, focus loss and touch layouts release it.
func (g *Game) captureMouse(capture bool) {
	if g.mouseCaptured == capture {
		return
	}
	mode := ebiten.CursorModeVisible
	if capture {
		mode = ebiten.CursorModeCaptured
	}
	ebiten.SetCursorMode(mode)
	g.mouseCaptured = capture
	g.mouseSpring = mouseSpring{}
	g.pending.input.SpringMotion, g.pending.input.SpringClick = 0, false
}

// updateFocus freezes production before the platform's suspended audio output
// can accumulate stale PCM. A fresh action resumes through the shared pause UI.
func (g *Game) updateFocus(focused bool) bool {
	if !focused {
		if g.focused {
			g.Suspend()
		}
		g.focused = false
		return false
	}
	g.focused = true
	return true
}

func (g *Game) newAudioStream(source io.Reader) (audioStream, error) {
	if g.context == nil {
		var err error
		g.context, err = newAudioDevice(g.audio.timing)
		if err != nil {
			return nil, err
		}
	}
	return g.context.open(source)
}

// Suspend is invoked on the game thread after Android queues a lifecycle event.
func (g *Game) Suspend() {
	g.Session.Suspend()
	// Releases may happen while updates are unfocused. The foreground barrier
	// owns those old keys; only the next resume gesture needs release suppression.
	g.quitKeys = quitKeys{}
	g.mouseSpring = mouseSpring{}
	g.controls.waitForRelease = true
	g.rootRequest.pending = false
	g.pending.reset()
	g.clock = timing.HostClock{}
	g.back = false
	g.audio.setPaused(true)
}
func (g *Game) Back() {
	g.back = true
}

func (g *Game) Draw(screen *ebiten.Image) {
	var frame *image.RGBA
	if g.Session.Table == nil {
		frame = g.Session.Frame(g.height)
	} else {
		bounds := image.Rect(0, 0, 320, g.height)
		if g.frame == nil || g.frame.Rect != bounds {
			g.frame = image.NewRGBA(bounds)
		}
		g.Session.DrawPresentedTable(g.frame)
		frame = g.frame
	}
	frame = g.drawTouchControls(frame)
	if g.image == nil || g.image.Bounds() != frame.Bounds() {
		if g.image != nil {
			g.image.Dispose()
		}
		g.image = ebiten.NewImageFromImage(frame)
	} else {
		g.image.WritePixels(frame.Pix)
	}
	op := &ebiten.DrawImageOptions{}
	if g.Session.Table == nil && frame.Bounds().Dy() < g.height {
		op.GeoM.Translate(0, float64(g.height-frame.Bounds().Dy())/2)
	}
	screen.DrawImage(g.image, op)
}

func (g *Game) Layout(width, height int) (int, int) {
	if width != g.layoutWidth || height != g.layoutHeight {
		// Rescaling virtual cursor coordinates is not physical mouse travel.
		g.mouseSpring = mouseSpring{}
		g.pending.input.SpringMotion, g.pending.input.SpringClick = 0, false
		g.layoutWidth, g.layoutHeight = width, height
	}
	scale := 1
	canvasWidth := 320
	if g.Session.Intro != nil {
		scale = 2
		canvasWidth = g.Session.Intro.Size().X
	}
	g.height = 256
	if g.Session.Table != nil {
		g.height = g.Session.Table.Viewport.Height
	}
	if g.Touch && height > width {
		g.height = resources.FullTableHeight
	}
	g.height *= scale
	if g.Touch && g.Session.Intro == nil {
		return canvasWidth, g.height + touchui.ToolbarHeight
	}
	return canvasWidth, g.height
}
