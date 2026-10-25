package resources

import (
	"fmt"
	"image"
)

// ScoreSeparators describes the supplied marks below each group of digits.
// These pixels belong to numeric writes, independently of retained glyph text.
type ScoreSeparators struct {
	LastCell, Digits, Y int
	Pixels              []image.Point
}

// TableViewport contains rectangles decoded from the original presentation.
// No display registers or scanout instructions enter the native renderer.
type TableViewport struct {
	Width, Height, PanelHeight, TableTop int
	SeparatorY                           int
	SeparatorColor                       Ink
}

const (
	DefaultTableWidth  = 320
	DefaultTableHeight = 262
	FullTableHeight    = 542
)

func (v TableViewport) FullHeight(tableRows int) int { return v.TableTop + tableRows }

func (v TableViewport) Validate() error {
	if v.Width != DefaultTableWidth || v.Height != DefaultTableHeight || v.PanelHeight != 23 || v.TableTop != 26 {
		return fmt.Errorf("unsupported native table viewport")
	}
	if v.SeparatorY < v.PanelHeight || v.SeparatorY >= v.TableTop || v.SeparatorColor.R > 15 || v.SeparatorColor.G > 15 || v.SeparatorColor.B > 15 {
		return fmt.Errorf("invalid native table separator")
	}
	return nil
}
