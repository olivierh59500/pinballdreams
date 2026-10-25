package game

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

func TestInitializingAudioRetainsInputAndPCMWithoutAdvancingGame(t *testing.T) {
	newGame := func() *Game {
		t.Helper()
		g, err := New(assets.Files(), false)
		if err != nil {
			t.Fatal(err)
		}
		if err = g.Session.Select(0); err != nil {
			t.Fatal(err)
		}
		g.Session.Start(1)
		g.SetAudioDiagnostics(true)
		g.audio.attach(g.Session)
		return g
	}
	g, control := newGame(), newGame()
	step := func(game *Game) {
		t.Helper()
		input, due := game.nextSessionInput()
		if !due {
			t.Fatal("expected a logical session step")
		}
		if err := game.Session.Update(input); err != nil {
			t.Fatal(err)
		}
		game.pending.finishFrame()
	}
	// The first logical update may initialize the output context. Its real
	// produced note/PCM block must remain intact during subsequent ready waits.
	step(g)
	step(control)
	ready := make(chan struct{})
	g.context = &audioDevice{ready: ready}
	clock, pose, declarations := g.Session.Rules.Frame, g.Session.Engine.State, g.Session.Sound.OutputDeclarations()
	queued, phase := g.audio.queue.BufferedBytes(), g.clock
	command := session.Input{Launch: true, SpringMotion: 5, MusicPressed: true}
	g.pending.add(command, nil)
	for range 1000 {
		g.pending.add(session.Input{Launch: true}, nil)
		if _, due := g.nextSessionInput(); due {
			t.Fatal("output initialization advanced the logical game")
		}
	}
	metrics := g.audio.queue.Metrics()
	if g.Session.Rules.Frame != clock || g.Session.Engine.State != pose || g.Session.Sound.OutputDeclarations() != declarations || g.audio.queue.BufferedBytes() != queued || g.clock != phase || metrics.ResetCalls != 0 || g.audio.diagnostics.readyWaitPolls != 1000 {
		t.Fatal("ready waits changed game/music state, clock, PCM or measurement")
	}
	if g.pending.input.SpringMotion != 5 || !g.pending.input.MusicPressed || !g.pending.input.Launch {
		t.Fatal("ready waits lost accumulated commands or pointer travel")
	}
	close(ready)
	control.pending.add(command, nil)
	for polls := 0; ; polls++ {
		left, a := g.nextSessionInput()
		right, b := control.nextSessionInput()
		if a != b || left != right || polls > 8 {
			t.Fatal("readiness changed logical timing or retained input order")
		}
		if a {
			if err := g.Session.Update(left); err != nil {
				t.Fatal(err)
			}
			if err := control.Session.Update(right); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	drain := func(game *Game) []byte {
		t.Helper()
		var result []byte
		for game.audio.queue.BufferedBytes() != 0 {
			block := make([]byte, 4096)
			n, err := game.audio.queue.Read(block)
			if err != nil || n == 0 {
				t.Fatal("queued output did not remain readable")
			}
			result = append(result, block[:n]...)
		}
		return result
	}
	if !bytes.Equal(drain(g), drain(control)) || g.Session.Engine.State != control.Session.Engine.State || g.Session.Sound.OutputDeclarations() != control.Session.Sound.OutputDeclarations() {
		t.Fatal("initialization waits dropped/reordered PCM, notes or gameplay")
	}
}
