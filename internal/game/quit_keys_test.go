package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"testing"
)

func TestOrdinaryQuitAnswersOnRelease(t *testing.T) {
	var keys quitKeys
	input := session.Input{}
	keys.apply(&input, true, false, []ebiten.Key{ebiten.KeyY}, nil)
	if input.AnyKey || input.QuitConfirm {
		t.Fatal("Y answered on press instead of release")
	}
	keys.apply(&input, true, false, nil, []ebiten.Key{ebiten.KeyEscape})
	if input.AnyKey || input.QuitConfirm {
		t.Fatal("Escape answered the question")
	}
	keys.apply(&input, true, false, nil, []ebiten.Key{ebiten.KeyY})
	if !input.AnyKey || !input.QuitConfirm {
		t.Fatal("Y release did not confirm")
	}
}

func TestQuestionResumeReleaseIsConsumedOnce(t *testing.T) {
	var keys quitKeys
	keys.consumeResume([]ebiten.Key{ebiten.KeyY})
	input := session.Input{}
	keys.apply(&input, true, false, nil, []ebiten.Key{ebiten.KeyY})
	if input.AnyKey || input.QuitConfirm {
		t.Fatal("host resume release also answered quit")
	}
	keys.apply(&input, true, false, []ebiten.Key{ebiten.KeyY}, nil)
	keys.apply(&input, true, false, nil, []ebiten.Key{ebiten.KeyY})
	if !input.QuitConfirm {
		t.Fatal("a new Y gesture remained consumed")
	}
}

func TestManualPausedQuestionKeepsPressSemantics(t *testing.T) {
	var keys quitKeys
	input := session.Input{}
	keys.apply(&input, true, true, []ebiten.Key{ebiten.KeyY}, nil)
	if !input.AnyKey || !input.QuitConfirm {
		t.Fatal("manual pause question lost its press-side response")
	}
}

func TestPPressCancelsOrdinaryQuestionWithoutPausing(t *testing.T) {
	g := readyTouchGame(t, 0)
	if err := g.Session.Update(session.Input{ExitTable: true}); err != nil {
		t.Fatal(err)
	}
	if !g.Session.AskingToQuit() {
		t.Fatal("question did not open")
	}
	input := session.Input{PauseKeyPressed: true}
	g.quitKeys.apply(&input, true, false, []ebiten.Key{ebiten.KeyP}, nil)
	if err := g.Session.Update(input); err != nil {
		t.Fatal(err)
	}
	if g.Session.Paused || g.Session.AskingToQuit() {
		t.Fatal("P froze or retained the original question")
	}
}

func TestSuspendClearsPreviousQuestionReleaseSuppression(t *testing.T) {
	g := readyTouchGame(t, 0)
	g.quitKeys.consumeResume([]ebiten.Key{ebiten.KeyY})
	g.Suspend()
	if len(g.quitKeys.consumed) != 0 {
		t.Fatal("a release missed while unfocused could suppress a later answer")
	}
}
