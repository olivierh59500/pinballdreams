// Package session owns the native game lifecycle, turns and original scoring.
package session

import (
	"image"
	"io/fs"

	"github.com/olivierh59500/pinballdreams/internal/frontend"
	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/physics"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/rules"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

// OtherMenuKey identifies a pressed key outside the hidden ticker alphabet.
const OtherMenuKey byte = 255

type InitialsAction uint8

const (
	InitialsNone InitialsAction = iota
	InitialsPrevious
	InitialsNext
	InitialsChoose
	InitialsErase
)

type Input struct {
	InitialsEdit                                                InitialsAction
	Left, Right, Launch, Nudge, Start, Pause, Back, Music, Skip bool
	// StartPressed is a fresh host command, independent of another held
	// start key or contact. Start retains the held-control compatibility path.
	StartPressed bool
	// MusicPressed is one newly delivered toggle, independent of held inputs.
	MusicPressed bool
	// PauseKeyPressed is a fresh original P-key event. Nested launch and bonus
	// routines can defer it; Pause remains an immediate host/touch request.
	PauseKeyPressed bool
	// ExitTable is a platform navigation request, independent of the original
	// Escape command that can cancel a ball in the spring lane.
	ExitTable bool
	// SpringMotion is relative vertical mouse movement since the previous update.
	// SpringClick releases after motion sampling; key release precedes it.
	SpringMotion                    int
	SpringClick                     bool
	Table, Players                  int
	HighScores, Backspace           bool
	QuitConfirm, QuitCancel, AnyKey bool
	Text                            string
	// MenuKey is a newly pressed letter or OtherMenuKey. Zero means no
	// event; original key codes never enter the native session.
	MenuKey byte
	// IntroExtraCredits is the held secondary control during the opening.
	IntroExtraCredits bool
}
type Player struct{ Score, Bonus uint64 }

type Session struct {
	frameAudio                                *frameAudio
	observeFrame                              func()
	observeSpring                             func()
	springLevel, springMotion                 int
	sceneCanvas                               *image.RGBA
	panel                                     *tablePanel
	ballSprite                                ballSprite
	displayedWorld                            TableView
	displayedArt                              *image.Paletted
	Rules                                     *rules.Engine
	Bonus                                     *rules.BonusCounter
	finale                                    *rules.HeldBonusFinale
	quit                                      *quitFlow
	tablePause                                *tablePause
	attract                                   *tableAttract
	startPhase                                tableStartPhase
	playerSelectionOpen                       bool
	Source                                    fs.FS
	Intro                                     *frontend.Intro
	Menu                                      *frontend.Menu
	Scores                                    *frontend.Scores
	ScoreBook                                 highscore.Book
	NameEntry                                 *highscore.Editor
	scoreStore                                *highscore.Store
	attractFrames                             int
	sceneChangeWait                           bool
	selection                                 *tableSelection
	entry                                     *tableEntry
	scoresPending                             bool
	footer                                    *frontend.Footer
	footerStarted                             bool
	gameOver                                  bool
	gameOverFrames                            int
	pendingTableReset                         bool
	cameraUp, spritesHidden                   bool
	scorePlayer                               int
	scoresEntered                             bool
	presentation                              *resources.Presentation
	Table                                     *resources.Table
	Engine                                    *physics.Engine
	Sound                                     *sound.Player
	Players                                   [8]Player
	bankPlayer                                int
	Player, PlayerCount, Ball, Charge, Camera int
	Playing, Paused, InChute, Muted           bool
	MusicDisabled                             bool
	playMusicSensor                           int
	awaitingPlayMusic                         bool
	previous                                  Input
	flippers                                  flipperInput
	nudgeMeter                                uint16
	nudgeDecay                                int
	AudioChanged                              bool
	preparation                               *ballPreparation
	restoration                               *rules.TargetRestoration
	ballStatus                                *ballStatus
	launching                                 bool
	keyboardPausePending                      bool
	cameraTransition                          bool
	pendingDrain, pendingReplacement          bool
	pendingNewBall                            bool
	pendingGameOver                           bool
	bonusClearPlayer                          int
	tiltEntered, nudgePrevious                bool
}

func New(source fs.FS) (*Session, error) {
	presentation, err := resources.PresentationFS(source)
	if err != nil {
		return nil, err
	}
	music, err := sound.DecodeFrontend(source, true)
	if err != nil {
		return nil, err
	}
	s := &Session{Source: source, presentation: presentation, Intro: frontend.NewIntro(&presentation.Opening), ScoreBook: presentation.Scores.Defaults, PlayerCount: 1, InChute: true}
	s.playMusic(music)
	return s, nil
}

func (s *Session) Select(id int) error {
	t, err := resources.LoadFS(s.Source, id)
	if err != nil {
		return err
	}
	m, err := sound.Decode(t)
	if err != nil {
		return err
	}
	r, err := rules.New(t)
	if err != nil {
		return err
	}
	s.Table = t
	s.flippers = flipperInput{blockedLeft: s.previous.Left, blockedRight: s.previous.Right}
	s.Charge = 0
	s.springLevel, s.springMotion = 0, 0
	s.selection = nil
	s.entry = nil
	// A cold table clears its sprite rows. Readiness can enable the ball
	// before the first spring frame prepares a mask, so retain that blank.
	s.ballSprite = ballSprite{}
	s.bankPlayer = 0
	s.bonusClearPlayer = -1
	s.nudgeDecay = t.Tilt.InitialDecay
	s.Scores = nil
	s.NameEntry = nil
	s.gameOver = false
	s.quit = nil
	s.tablePause = nil
	s.keyboardPausePending = false
	s.pendingTableReset, s.spritesHidden = false, false
	s.Intro = nil
	s.Engine = physics.New(t)
	s.playMusic(m)
	s.playMusicSensor = m.PlaySensor
	s.MusicDisabled = m.InitialMusicDisabled
	if s.MusicDisabled {
		s.Sound.BeginPlay(true)
	}
	s.Rules = r
	s.initializePanel()
	s.Bonus = nil
	s.finale = nil
	r.OnAward = func(points, bonus uint64) {
		p := &s.Players[s.bankPlayer]
		p.Score = (p.Score + points) % 1_000_000_000_000
		p.Bonus = (p.Bonus + bonus) % 1_000_000_000_000
	}
	r.OnScoreActivity = func() {
		// Contact/feature additions postpone idle announcements, even
		// when this contact or feature adds zero to both banks.
		s.panel.idle = s.Table.Rules.Panel.RepeatDelay
	}
	r.OnPanelNumber = s.panel.number
	r.OnRetriggerPhase = s.Sound.SetRetriggerPhase
	r.OnSound = s.Sound.Trigger
	r.OnTargetChange = func(id int, active bool) { s.Engine.SetTarget(id, active) }
	r.OnTargetLayerChange = func(level, id int, active bool) { s.Engine.SetTargetOnLayer(level, id, active) }
	r.OnCapture = func(x, y int, vx, vy int16) {
		s.Engine.State.X = int32(x << 10)
		s.Engine.State.Y = int32(y << 10)
		s.Engine.State.VX = vx
		s.Engine.State.VY = vy
		s.Engine.Locked = true
	}
	r.Banks = func() (uint64, uint64) { p := s.Players[s.bankPlayer]; return p.Score, p.Bonus }
	r.ActorPlayer = func() int { return s.bankPlayer }
	r.BanksFor = func(player int) (uint64, uint64) { p := s.Players[player]; return p.Score, p.Bonus }
	s.Playing = false
	s.preparation, s.ballStatus = nil, nil
	s.restoration = nil
	s.launching = false
	s.cameraTransition = false
	s.pendingDrain, s.pendingReplacement = false, false
	s.pendingNewBall = false
	s.pendingGameOver = false
	s.tiltEntered, s.nudgePrevious = false, false
	s.InChute = true
	s.Camera = 276
	s.Paused = false
	s.enterTableAttract()
	return nil
}

func (s *Session) Start(players int) {
	s.flippers = flipperInput{blockedLeft: s.previous.Left, blockedRight: s.previous.Right}
	s.entry = nil
	s.playerSelectionOpen = true
	s.finale = nil
	s.attract = nil
	s.startPhase = tableStartIdle
	s.pendingGameOver = false
	s.bonusClearPlayer = -1
	s.tiltEntered = false
	s.NameEntry = nil
	s.gameOver = false
	s.quit = nil
	s.pendingTableReset, s.spritesHidden = false, false
	s.Rules.Reset()
	s.Players = [8]Player{}
	s.Player = 0
	s.bankPlayer = 0
	s.PlayerCount = max(1, min(8, players))
	s.Rules.PlayerCount = s.PlayerCount
	s.Ball = 1
	s.Playing = true
	s.nextBall()
}

func (s *Session) nextBall() {
	s.keyboardPausePending = false
	s.Bonus = nil
	s.pendingNewBall = false
	s.tiltEntered = false
	s.pendingDrain, s.pendingReplacement = false, false
	s.launching = false
	s.preparation = nil
	s.InChute = true
	s.nudgeMeter = 0
	s.Rules.Layer = 0
	s.restoration = s.Rules.BeginTargetRestoration(s.Player)
	s.Rules.DrawSpring(0)
	if !s.restoration.Pending() {
		s.restoration.Finish()
		s.restoration = nil
		s.serveNormalBall()
		s.beginBallCountdown()
	}
}

// serveNormalBall follows target restoration, retaining spin, nudge/flipper
// pose and the outgoing bank until spring entry selects the next player's bank.
func (s *Session) serveNormalBall() {
	s.Rules.HoldBonus = false
	s.Engine.SetLayer(0)
	s.Engine.State.X, s.Engine.State.Y = 302<<10, 470<<10
	s.Engine.State.VX, s.Engine.State.VY = 0, 0
	s.Engine.Tilted = false
	s.updateCamera()
}

func (s *Session) beginBallCountdown() {
	if s.bonusClearPlayer >= 0 {
		s.Players[s.bonusClearPlayer].Bonus = 0
		s.bonusClearPlayer = -1
	}
	s.awaitingPlayMusic = true
	s.Sound.BeginBall(s.MusicDisabled)
	blank := false
	if s.ballStatus != nil {
		blank = s.ballStatus.blank
	}
	s.preparation = newBallPreparation(&s.Table.BallPreparation, blank)
	s.ballStatus = &s.preparation.status
	if s.Rules.ReservedBallsActive() {
		s.ballStatus.mode = reservedStatus
	} else if s.Rules.RelaunchPending() {
		s.ballStatus.mode = relaunchStatus
	}
	s.Sound.Trigger(s.Table.BallPreparation.Cues[0])
}

func (s *Session) Update(input Input) (err error) {
	s.retainTableView()
	input = s.applyPauseKey(input)
	if s.Paused && s.tablePause == nil && input.AnyKey {
		// A system/host pause resumes without dispatching the same command
		// to a menu, roster or initials editor.
		input = Input{Pause: input.Pause, AnyKey: true, Left: input.Left, Right: input.Right}
	}
	if s.frameAudio != nil {
		defer func() {
			if audioErr := s.finishAudioFrame(); err == nil {
				err = audioErr
			}
		}()
	}
	if s.tablePause != nil {
		if s.tablePause.suspended {
			if s.pausePressed(input) {
				s.tablePause.suspended = false
				s.Paused = !s.tablePause.exiting
				s.Sound.Pause(s.AudioOutputPaused())
			}
			s.previous = input
			return nil
		}
		if s.tablePause.exiting {
			s.tablePause = nil
			s.beginQuit()
			s.quit.phase = quitExitStart
		} else {
			s.advanceTablePause(input, false)
			s.previous = input
			return nil
		}
	}
	if s.entry != nil {
		if !s.entry.ready {
			if s.pausePressed(input) {
				s.Paused = !s.Paused
				s.Sound.Pause(s.Paused)
			}
			if !s.Paused {
				s.advanceTableEntry(input)
			}
			s.previous = input
			return nil
		}
		s.entry = nil
	}
	// The ordinary wait clears numeric separator rows independently of the
	// retained glyph cells. Numeric writers republish their marks this frame.
	if s.panel != nil && !s.Paused {
		s.panel.scoreDigits = 0
	}
	if s.quit != nil {
		if s.pausePressed(input) {
			s.Paused = !s.Paused
			s.Sound.Pause(s.Paused)
			s.previous = input
			return nil
		}
		if s.Paused {
			s.previous = input
			return nil
		}
		if s.quit.phase == quitComplete {
			if err := s.ShowMenu(); err != nil {
				return err
			}
			// The next source wait already belongs to selector replay and
			// its palette rise. Keep those services in the same update.
			err := s.updatePresentation(input)
			s.previous = input
			return err
		}
		result := s.advanceQuit(input)
		s.previous = input
		switch result {
		case quitCancelled:
			s.quit = nil
			s.gameOver = false
			s.pendingTableReset = true
		}
		return nil
	}
	if s.NameEntry != nil {
		if s.pausePressed(input) {
			s.Paused = !s.Paused
			s.Sound.Pause(s.Paused)
		}
		if s.Paused {
			s.previous = input
			return nil
		}
		err := s.updateNameEntry(input)
		s.previous = input
		return err
	}
	if s.Table != nil && s.pendingTableReset && !s.Paused {
		s.enterTableAttract()
	}
	if s.NameEntry == nil && (input.ExitTable || input.Back && !s.previous.Back && !(s.Playing && (s.PreparingBall() || s.launching))) {
		if s.Table != nil {
			if s.gameOver {
				s.presentScoreCycle(input)
			} else if s.attract != nil {
				s.updateTableAttract(input)
			}
			s.beginQuit()
			s.previous = input
			return nil
		}
	}
	if s.Table == nil {
		err := s.updatePresentation(input)
		s.previous = input
		return err
	}
	if !s.Paused && input.Pause && !s.previous.Pause && s.Playing && !s.PreparingBall() && !s.launching && s.Bonus == nil && s.finale == nil && !s.gameOver {
		s.tablePause = &tablePause{}
		s.Paused = true
		s.Sound.Pause(false)
		s.advanceTablePause(input, true)
		s.previous = input
		return nil
	}
	if s.pausePressed(input) {
		s.Paused = !s.Paused
		s.Sound.Pause(s.Paused)
	}
	if s.Paused {
		s.previous = input
		return nil
	}
	if s.startPressed(input) && !s.pendingGameOver {
		if s.gameOver {
			s.presentScoreCycle(input)
			s.selectTablePlayers(input.Players, true)
			s.previous = input
			return nil
		}
		if s.attract != nil {
			if s.startPhase == tableStartIdle {
				s.updateTableAttract(input)
			} else {
				s.updateTableStart(input)
			}
			s.selectTablePlayers(input.Players, false)
			s.previous = input
			return nil
		}
		// Selection during the first ball's readiness/spring wait is serviced
		// at that phase's original keyboard poll. Active play ignores it.
	}
	if s.pendingGameOver {
		s.pendingGameOver = false
		s.endGame()
		if s.NameEntry != nil {
			err := s.updateNameEntry(input)
			s.previous = input
			return err
		}
		s.updateScoreCycle(input)
		s.previous = input
		return nil
	}
	if s.pendingTableReset {
		s.enterTableAttract()
	}
	if !s.Playing {
		if s.gameOver {
			s.updateScoreCycle(input)
			s.previous = input
			return nil
		}
		if input.Launch && !s.previous.Launch && s.attract != nil && s.startPhase == tableStartIdle {
			s.updateTableAttract(input)
			s.selectTablePlayers(1, false)
			s.previous = input
			return nil
		}
		if s.attract != nil && !s.Playing {
			if s.startPhase == tableStartIdle {
				s.updateTableAttract(input)
			} else {
				s.updateTableStart(input)
			}
			s.previous = input
			return nil
		}
		s.Rules.DrawSpring(0)
		s.Rules.Tick(true, false, false)
		s.previous = input
		return nil
	}
	if s.pendingNewBall {
		s.pendingNewBall = false
		s.nextBall()
	}
	if s.pendingReplacement {
		s.pendingReplacement = false
		if s.Rules.TakeReplacement() {
			s.replaceBall()
		}
	}
	if s.pendingDrain {
		s.pendingDrain = false
		s.spritesHidden = true
		if s.Engine.Tilted {
			s.Bonus = s.Rules.BeginTiltedBonus(s.Players[s.bankPlayer].Bonus)
		} else {
			s.Bonus = s.Rules.BeginBonus(s.Players[s.bankPlayer].Bonus)
		}
		s.Sound.StopAtReturn()
	}
	if s.pendingNewBall || !s.Playing {
		s.previous = input
		return nil
	}
	if s.Bonus != nil {
		s.beginTableWait(input)
		if s.Bonus.MovingBackdrop() {
			s.Engine.Locked = true
			s.updateCamera()
			s.Engine.Step(physics.Controls{})
		}
		finished := s.Bonus.Step()
		s.writePanel(s.Bonus.Message())
		if finished {
			s.finishBall()
		}
		s.previous = input
		return nil
	}
	if s.finale != nil {
		s.beginTableWait(input)
		finished := s.finale.Step()
		s.writePanel(s.finale.Message())
		if finished {
			s.finale = nil
			s.completeBallEnd()
		}
		s.previous = input
		return nil
	}
	if s.PreparingBall() {
		s.beginTableWait(input)
		s.prepareBallFrame(input)
		s.previous = input
		return nil
	}
	s.beginObservedFrame()
	impulse := s.sampleTableInput(input)
	if s.launching {
		s.updateSpringFrame(input, impulse)
		s.previous = input
		return nil
	}
	if s.Engine.Tilted {
		s.updateTiltFrame(input)
		s.previous = input
		return nil
	}
	s.Rules.BeginLiveFrame()
	s.updateCamera()
	s.Engine.Step(s.motionControls(input))
	s.prepareBallSprite()
	s.advancePanelBeforeContacts()
	// The source samples the next force before target/sensor effects and
	// their final ramp-level switch, while geometry still belongs to this tick.
	s.Engine.SampleGravity()
	s.InChute = false
	px, py := s.Engine.State.PixelX()+8, s.Engine.State.PixelY()+8
	if !s.Engine.Tilted {
		rolloverContact := false
		if s.Rules.HasRollovers() {
			rolloverContact = s.Table.RolloverContact(s.Rules.ArtView(false, false), s.Engine.Layer,
				s.Engine.State.PixelX(), s.Engine.State.PixelY(), s.Engine.State.Nudge())
		}
		if s.Engine.ImpactObject != 0 {
			s.Rules.Hit(s.Engine.ImpactObject, false)
		}
		if s.Engine.HadContact {
			c := s.Engine.FrameContact
			s.Rules.Targets(c.PointX, c.PointY, c.Material, c.Powered)
		}
		s.Rules.Accelerator()
		s.Rules.SensorContacts(px, py, rolloverContact)
		s.Engine.SelectLayer()
		s.Rules.Layer = s.Engine.Layer
		s.Rules.Accelerator()
		if s.awaitingPlayMusic {
			for _, sensor := range s.Table.Layers[s.Engine.Layer].Sensors {
				if sensor.Address == s.playMusicSensor && px >= sensor.X0 && px <= sensor.X1 && py >= sensor.Y0 && py <= sensor.Y1 {
					s.Sound.BeginPlay(s.MusicDisabled)
					s.awaitingPlayMusic = false
					break
				}
			}
		}
	}
	s.Rules.FinishLiveFrame(s.flippers.left && !s.previous.Left || s.flippers.right && !s.previous.Right)
	if s.Rules.ReplacementPending() {
		s.cameraTransition = true
	}
	s.advancePanelAfterActions()
	s.advanceNudge(input.Nudge)
	s.Rules.FinishCapture(s.cameraTransition)
	s.pendingReplacement = s.Rules.ReplacementPending()
	s.Engine.Locked = s.Rules.Locked
	s.InChute = s.Rules.SensorActive(s.Table.Launch.SpringSensor)
	if s.InChute && impulse > 0 {
		s.Engine.State.VY -= int16(impulse)
	}
	if s.InChute && s.ballStatus != nil {
		s.ballStatus.advance(s.Rules.Frame, s.PlayerCount, s.Player, s.Ball, s.Table.ScoreText(s.Player, s.Players[s.Player].Score))
	}
	if s.Engine.State.PixelY() > 512 {
		s.pendingDrain = true
	}
	s.previous = input
	return nil
}

// replaceBall services a reserved ball while retaining the current table state.
func (s *Session) replaceBall() {
	s.keyboardPausePending = false
	s.launching = false
	s.restoration = nil
	s.Engine.SetLayer(0)
	s.Engine.State.X, s.Engine.State.Y = 302<<10, 470<<10
	s.Engine.State.VX, s.Engine.State.VY = 0, 0
	s.Engine.Locked = false
	s.InChute = true
	s.cameraTransition = true
	s.updateCamera()
	s.nudgeMeter = 0
	s.beginBallCountdown()
	// A reserved replacement remains hidden and held during readiness. Its
	// final countdown boundary enables the retained sprite and spring motion.
	s.spritesHidden = true
	s.Engine.Locked = true
}

// finishBall advances only after bonus counting, preserving earned extra balls.
func (s *Session) finishBall() {
	s.Bonus = nil
	if !s.Rules.HoldBonus {
		s.bonusClearPlayer = s.bankPlayer
	}
	s.selectNextTurn()
	if s.Ball > 3 && s.Rules.HoldBonus {
		// The last retained bonus is counted once more before score entry.
		// Constructing the announcement shares the ordinary count's last frame.
		s.finale = s.Rules.BeginHeldBonusFinale(s.Players[s.bankPlayer].Bonus)
		s.Sound.StopAtReturn()
		return
	}
	s.completeBallEnd()
}

// selectNextTurn changes the display/collection mask while keeping the actor
// bank selected until spring entry or score qualification. The final encore
// therefore runs with the already-advanced mask, including the eight-player
// no-active-bit boundary, and awards still reach the outgoing bank.
func (s *Session) selectNextTurn() {
	if !s.Rules.RelaunchPending() {
		s.Player++
	}
	if s.Player >= s.PlayerCount {
		s.Ball++
		if s.Ball <= 3 {
			s.Player = 0
		}
	}
	s.Rules.Player = s.Player
}

func (s *Session) completeBallEnd() {
	s.keyboardPausePending = false
	if s.Ball > 3 {
		// Score qualification and its music belong to the next outer update.
		s.pendingGameOver = true
	} else {
		s.pendingNewBall = true
	}
}
