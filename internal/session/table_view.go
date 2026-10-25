package session

import (
	"image"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// TableView retains the world projection independently of the current physics.
// During ordinary play, the original displays the previous ball/camera view
// alongside the current flipper artwork. Panel cells and numeric marks follow
// that presented epoch too. Cold entry and attract retain their preceding
// camera, panel and color limit. The lighting surface is retained in
// a reusable buffer; rendering needs no processor or device timing model.
type TableView struct {
	Camera, Nudge int
	Ball          image.Point
	BallPixels    [16 * 15]byte
	SpritesHidden bool
	panelCells    [20]byte
	scoreDigits   int
	hasPanel      bool
	entryLevel    int
	art           *image.Paletted
	table         *resources.Table
	valid         bool
}

func (s *Session) currentTableView() TableView {
	if s.Table == nil || s.Engine == nil {
		return TableView{}
	}
	view := TableView{
		Camera: s.Camera, Nudge: s.Engine.VisualNudge(),
		Ball:       image.Pt(s.Engine.State.PixelX()+1, s.Engine.State.PixelY()+1),
		BallPixels: s.ballSprite.pixels, SpritesHidden: s.spritesHidden,
		table: s.Table, valid: true, entryLevel: -1,
	}
	if s.entry != nil {
		view.entryLevel = s.entry.level
	}
	return view
}

func (s *Session) ordinaryTablePlay() bool {
	return s.Playing && s.NameEntry == nil && s.entry == nil && s.attract == nil && s.quit == nil &&
		!s.gameOver && s.preparation == nil && s.restoration == nil && !s.launching && !s.inBonusPresentation()
}

func (s *Session) retainsTableProjection() bool {
	return s.ordinaryTablePlay() || s.entry != nil ||
		!s.Playing && s.attract != nil && s.startPhase == tableStartIdle && s.quit == nil
}

func (s *Session) retainTableView() {
	if s.Paused {
		// Foreground suspension freezes the already published picture.
		return
	}
	if s.Table == nil || !s.retainsTableProjection() {
		s.displayedWorld = TableView{}
		return
	}
	s.displayedWorld = s.currentTableView()
	if s.panel != nil {
		s.displayedWorld.panelCells = s.panel.cells
		s.displayedWorld.scoreDigits = s.panel.scoreDigits
		s.displayedWorld.hasPanel = true
	}
	art := s.Rules.ArtView(!s.Playing, s.Engine.Tilted)
	if s.displayedArt == nil || s.displayedArt.Rect != art.Rect {
		s.displayedArt = image.NewPaletted(art.Rect, art.Palette)
	}
	s.displayedArt.Palette = art.Palette
	copy(s.displayedArt.Pix, art.Pix)
	s.displayedWorld.art = s.displayedArt
}

// PresentedTableView describes the projection used by both application hosts.
// Cold entry and attract retain the preceding camera, panel and palette too.
// The original P-key pause settles at the current, frozen world position.
func (s *Session) PresentedTableView() TableView {
	if s.displayedWorld.valid && s.displayedWorld.table == s.Table && s.retainsTableProjection() && s.tablePause == nil {
		return s.displayedWorld
	}
	return s.currentTableView()
}

// PresentedFrame returns the application's visible scene. Frame remains the
// current software scene for the separate original-routine comparisons.
func (s *Session) PresentedFrame(height int) *image.RGBA {
	if s.Table == nil {
		return s.Frame(height)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 320, height))
	s.DrawPresentedTable(frame)
	return frame
}

// DrawPresentedTable reuses the host's canvas and paints current flippers over
// the retained world projection. The view is a value snapshot, with no aliases
// to a mutable sprite mask and no per-update pixel allocation.
func (s *Session) DrawPresentedTable(frame *image.RGBA) {
	s.drawTable(frame, s.PresentedTableView())
}
