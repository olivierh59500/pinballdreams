// Pinball Dreams is a native Go conversion using reconstructed original data.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/game"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
)

func main() {
	table := flag.Int("table", 0, "open a table, 1 through 4; 0 plays the original opening")
	menu := flag.Bool("menu", false, "open table selection directly")
	saveDirectory := flag.String("save-dir", "", "score directory; default uses the system user configuration directory")
	touch := flag.Bool("touch", false, "enable the portrait touch layout")
	capture := flag.String("capture", "", "write a headless PNG capture and exit")
	ticks := flag.Int("ticks", 0, "native simulation frames before a capture")
	flag.Parse()
	if *capture != "" {
		s, err := session.New(assets.Files())
		if err != nil {
			panic(err)
		}
		if *menu {
			if err = s.ShowMenu(); err != nil {
				panic(err)
			}
		}
		if *table > 0 {
			if err = s.Select(*table - 1); err != nil {
				panic(err)
			}
			s.Start(1)
		}
		// Headless simulation shares the live replay clock while discarding
		// its PCM. Song-dependent bonus and game-over waits still advance.
		s.EnableAppAudio(nil)
		playFrames := 0
		for n := range *ticks {
			preparing := s.PreparingBall()
			if err = s.Update(session.Input{Launch: playFrames < 30, Left: n%80 < 40, Right: n%95 < 50}); err != nil {
				panic(err)
			}
			if !preparing {
				playFrames++
			}
		}
		if err = os.MkdirAll(filepath.Dir(*capture), 0755); err != nil {
			panic(err)
		}
		f, err := os.Create(*capture)
		if err != nil {
			panic(err)
		}
		frameHeight := 256
		if s.Table != nil {
			frameHeight = s.Table.Viewport.Height
		}
		if err = png.Encode(f, s.PresentedFrame(frameHeight)); err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		fmt.Println("Native frame:", *capture)
		return
	}
	if *saveDirectory == "" {
		directory, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}
		*saveDirectory = filepath.Join(directory, "PinballDreams")
	}
	g, err := game.NewWithStorage(nil, *touch, *saveDirectory)
	if err != nil {
		panic(err)
	}
	if *menu {
		if err = g.Session.ShowMenu(); err != nil {
			panic(err)
		}
	}
	if *table > 0 {
		if err = g.Session.Select(*table - 1); err != nil {
			panic(err)
		}
	}
	ebiten.SetWindowTitle("Pinball Dreams — Native Go")
	ebiten.SetWindowSize(960, 768)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenFilterEnabled(false)
	ebiten.SetTPS(timing.HostTicksPerSecond)
	if err = ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
