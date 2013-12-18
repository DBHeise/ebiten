package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"image"
)

type Device struct {
	ids *ids
}

func NewDevice() *Device {
	device := &Device{
		ids: newIds(),
	}
	return device
}

func (d *Device) CreateContext(screenWidth, screenHeight, screenScale int) *Context {
	return newContext(d.ids, screenWidth, screenHeight, screenScale)
}

func (d *Device) Update(context *Context, draw func(graphics.Context)) {
	context.update(draw)
}

func (d *Device) CreateRenderTarget(width, height int) (graphics.RenderTargetId, error) {
	renderTargetId, err := d.ids.CreateRenderTarget(width, height, graphics.FilterLinear)
	if err != nil {
		return 0, err
	}
	return renderTargetId, nil
}

func (d *Device) CreateTexture(img image.Image, filter graphics.Filter) (graphics.TextureId, error) {
	return d.ids.CreateTexture(img, filter)
}
