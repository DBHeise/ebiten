// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ebiten

import (
	"fmt"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/buffered"
	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/mipmap"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

func init() {
	mipmap.SetGraphicsDriver(uiDriver().Graphics())
	shareable.SetGraphicsDriver(uiDriver().Graphics())
	graphicscommand.SetGraphicsDriver(uiDriver().Graphics())
}

type defaultGame struct {
	update  func(screen *Image) error
	width   int
	height  int
	context *uiContext
}

func (d *defaultGame) Update(screen *Image) error {
	return d.update(screen)
}

func (d *defaultGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Ignore the outside size.
	d.context.m.Lock()
	w, h := d.width, d.height
	d.context.m.Unlock()
	return w, h
}

type uiContext struct {
	game      Game
	offscreen *Image
	screen    *Image

	// scaleForWindow is the scale of a window. This doesn't represent the scale on fullscreen. This value works
	// only on desktops.
	//
	// scaleForWindow is for backward compatibility and is used to calculate the window size when SetScreenSize
	// is called.
	scaleForWindow float64

	outsideSizeUpdated bool
	outsideWidth       float64
	outsideHeight      float64

	m sync.Mutex
}

var theUIContext = &uiContext{}

func (c *uiContext) set(game Game, scaleForWindow float64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.game = game

	if g, ok := game.(*defaultGame); ok {
		c.scaleForWindow = scaleForWindow
		g.context = c
	}
}

func (c *uiContext) setScaleForWindow(scale float64) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.game == nil {
		panic("ebiten: setScaleForWindow can be called only after the main loop starts")
	}

	g, ok := c.game.(*defaultGame)
	if !ok {
		panic("ebiten: setScaleForWindow can be called only when Run is used")
	}

	if w := uiDriver().Window(); w != nil {
		ww, wh := g.width, g.height
		c.scaleForWindow = scale
		w.SetSize(int(float64(ww)*scale), int(float64(wh)*scale))
	}
}

func (c *uiContext) getScaleForWindow() float64 {
	c.m.Lock()
	defer c.m.Unlock()

	if c.game == nil {
		panic("ebiten: getScaleForWindow can be called only after the main loop starts")
	}

	if _, ok := c.game.(*defaultGame); !ok {
		panic("ebiten: getScaleForWindow can be called only when Run is used")
	}
	s := c.scaleForWindow
	return s
}

// SetScreenSize sets the (logical) screen size and adjusts the window size.
//
// SetScreenSize is for backward compatibility. This is called from ebiten.SetScreenSize and
// uidriver/mobile.UserInterface.
func (c *uiContext) SetScreenSize(width, height int) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.game == nil {
		panic("ebiten: SetScreenSize can be called only after the main loop starts")
	}

	g, ok := c.game.(*defaultGame)
	if !ok {
		panic("ebiten: SetScreenSize can be called only when Run is used")
	}

	g.width = width
	g.height = height
	if w := uiDriver().Window(); w != nil {
		s := c.scaleForWindow
		w.SetSize(int(float64(width)*s), int(float64(height)*s))
	}
}

func (c *uiContext) Layout(outsideWidth, outsideHeight float64) {
	c.outsideSizeUpdated = true
	c.outsideWidth = outsideWidth
	c.outsideHeight = outsideHeight
}

func (c *uiContext) updateOffscreen() {
	sw, sh := c.game.Layout(int(c.outsideWidth), int(c.outsideHeight))
	if sw <= 0 || sh <= 0 {
		panic("ebiten: Layout must return positive numbers")
	}

	if c.offscreen != nil && !c.outsideSizeUpdated {
		if w, h := c.offscreen.Size(); w == sw && h == sh {
			return
		}
	}
	c.outsideSizeUpdated = false

	if c.screen != nil {
		_ = c.screen.Dispose()
		c.screen = nil
	}

	if c.offscreen != nil {
		if w, h := c.offscreen.Size(); w != sw || h != sh {
			_ = c.offscreen.Dispose()
			c.offscreen = nil
		}
	}
	if c.offscreen == nil {
		c.offscreen = newImage(sw, sh, FilterDefault, true)
	}

	// The window size is automatically adjusted when Run is used.
	if _, ok := c.game.(*defaultGame); ok {
		c.SetScreenSize(sw, sh)
	}

	// TODO: This is duplicated with mobile/ebitenmobileview/funcs.go. Refactor this.
	d := uiDriver().DeviceScaleFactor()
	c.screen = newScreenFramebufferImage(int(c.outsideWidth*d), int(c.outsideHeight*d))

	// Do not have to update scaleForWindow since this is used only for backward compatibility.
	// Then, if a window is resizable, scaleForWindow (= ebiten.ScreenScale) might not match with the actual
	// scale. This is fine since ebiten.ScreenScale will be deprecated.
}

func (c *uiContext) setWindowResizable(resizable bool) {
	c.m.Lock()
	defer c.m.Unlock()

	if resizable && c.game != nil {
		if _, ok := c.game.(*defaultGame); ok {
			panic("ebiten: a resizable window works with RunGame, not Run")
		}
	}
	if w := uiDriver().Window(); w != nil {
		w.SetResizable(resizable)
	}
}

func (c *uiContext) screenScale() float64 {
	if c.offscreen == nil {
		return 0
	}
	sw, sh := c.offscreen.Size()
	d := uiDriver().DeviceScaleFactor()
	scaleX := c.outsideWidth / float64(sw) * d
	scaleY := c.outsideHeight / float64(sh) * d
	return math.Min(scaleX, scaleY)
}

func (c *uiContext) offsets() (float64, float64) {
	if c.offscreen == nil {
		return 0, 0
	}
	sw, sh := c.offscreen.Size()
	d := uiDriver().DeviceScaleFactor()
	s := c.screenScale()
	width := float64(sw) * s
	height := float64(sh) * s
	return (c.outsideWidth*d - width) / 2, (c.outsideHeight*d - height) / 2
}

func (c *uiContext) Update(afterFrameUpdate func()) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.

	if err := buffered.BeginFrame(); err != nil {
		return err
	}
	if err := c.update(afterFrameUpdate); err != nil {
		return err
	}
	if err := buffered.EndFrame(); err != nil {
		return err
	}

	return nil
}

func (c *uiContext) update(afterFrameUpdate func()) error {
	updateCount := clock.Update(MaxTPS())
	for i := 0; i < updateCount; i++ {
		c.updateOffscreen()

		// Mipmap images should be disposed by Clear.
		c.offscreen.Clear()

		setDrawingSkipped(i < updateCount-1)

		if err := hooks.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		if err := c.game.Update(c.offscreen); err != nil {
			return err
		}
		uiDriver().Input().ResetForFrame()
		afterFrameUpdate()
	}

	// c.screen might be nil when updateCount is 0 in the initial state (#1039).
	if c.screen == nil {
		return nil
	}

	// This clear is needed for fullscreen mode or some mobile platforms (#622).
	c.screen.Clear()

	op := &DrawImageOptions{}

	s := c.screenScale()
	switch vd := uiDriver().Graphics().VDirection(); vd {
	case driver.VDownward:
		// c.screen is special: its Y axis is down to up,
		// and the origin point is lower left.
		op.GeoM.Scale(s, -s)
		_, h := c.offscreen.Size()
		op.GeoM.Translate(0, float64(h)*s)
	case driver.VUpward:
		op.GeoM.Scale(s, s)
	default:
		panic(fmt.Sprintf("ebiten: invalid v-direction: %d", vd))
	}

	op.GeoM.Translate(c.offsets())
	op.CompositeMode = CompositeModeCopy

	// filterScreen works with >=1 scale, but does not well with <1 scale.
	// Use regular FilterLinear instead so far (#669).
	if s >= 1 {
		op.Filter = filterScreen
	} else {
		op.Filter = FilterLinear
	}
	_ = c.screen.DrawImage(c.offscreen, op)
	return nil
}

func (c *uiContext) AdjustPosition(x, y float64) (float64, float64) {
	d := uiDriver().DeviceScaleFactor()
	ox, oy := c.offsets()
	s := c.screenScale()
	return (x*d - ox) / s, (y*d - oy) / s
}
