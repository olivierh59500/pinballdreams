package session

import (
	"fmt"
	"image"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
)

func BenchmarkTableFrame(b *testing.B) {
	for table := range 4 {
		for _, height := range []int{262, 542} {
			for _, reuse := range []bool{false, true} {
				b.Run(fmt.Sprintf("table%d/height%d/reuse%t", table+1, height, reuse), func(b *testing.B) {
					s, err := New(assets.Files())
					if err != nil {
						b.Fatal(err)
					}
					if err := s.Select(table); err != nil {
						b.Fatal(err)
					}
					s.Start(1)
					s.Frame(height)
					canvas := image.NewRGBA(image.Rect(0, 0, 320, height))
					b.ReportAllocs()
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						if reuse {
							s.DrawTable(canvas)
						} else {
							s.Frame(height)
						}
					}
				})
			}
		}
	}
}

// BenchmarkAttractDraw measures the production update/draw handover, including
// changing lamp artwork rather than repeatedly drawing an unchanged image.
func BenchmarkAttractDraw(b *testing.B) {
	for table := range 4 {
		b.Run(fmt.Sprintf("table%d", table+1), func(b *testing.B) {
			s, err := New(assets.Files())
			if err != nil {
				b.Fatal(err)
			}
			if err = s.Select(table); err != nil {
				b.Fatal(err)
			}
			canvas := image.NewRGBA(image.Rect(0, 0, 320, 542))
			s.DrawTable(canvas)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if err = s.Update(Input{}); err != nil {
					b.Fatal(err)
				}
				s.DrawTable(canvas)
			}
		})
	}
}

// BenchmarkPresentedWorld measures the reusable lighting copy and application
// renderer after a natural launch, without per-frame pixel allocation.
func BenchmarkPresentedWorld(b *testing.B) {
	for table := range 4 {
		b.Run(fmt.Sprintf("table%d", table+1), func(b *testing.B) {
			s, err := New(assets.Files())
			if err != nil {
				b.Fatal(err)
			}
			if err = s.Select(table); err != nil {
				b.Fatal(err)
			}
			s.EnableFrameAudio(nil)
			if err = s.Update(Input{Start: true, Players: 1, Launch: true}); err != nil {
				b.Fatal(err)
			}
			for range 400 {
				if err = s.Update(Input{Launch: true}); err != nil {
					b.Fatal(err)
				}
			}
			for n := 0; n < 512 && !s.ordinaryTablePlay(); n++ {
				if err = s.Update(Input{}); err != nil {
					b.Fatal(err)
				}
			}
			if !s.ordinaryTablePlay() {
				b.Fatal("natural launch did not enter ordinary play")
			}
			canvas := image.NewRGBA(image.Rect(0, 0, 320, s.Table.Viewport.Height))
			s.retainTableView()
			s.DrawPresentedTable(canvas)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.retainTableView()
				s.DrawPresentedTable(canvas)
			}
		})
	}
}
