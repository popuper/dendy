package ppu

type Tile struct {
	Pixels    [8][8]uint8
	PaletteID uint8
}

func (p *PPU) tilePatternTableAddr() uint16 {
	if p.getFlag(CtrlPatternTableSelect) {
		return 0x1000
	}

	return 0
}

func (p *PPU) nameTableAddr() uint16 {
	return 0x2000 + uint16(p.Ctrl&CtrlNameTableSelect)*0x400
}

func (p *PPU) fetchTile(tileX, tileY int) (tile Tile) {
	offset := p.nameTableAddr()

	tileX += int(p.ScrollX / 8)
	if tileX >= 32 {
		offset ^= 0x0400
		tileX -= 32
	}

	tileY += int(p.ScrollY / 8)
	if tileY >= 30 {
		offset ^= 0x0800
		tileY -= 30
	}

	attr := p.readVRAM(offset + 0x03C0 + uint16(tileX)/4 + uint16(tileY)/4*8)
	id := p.readVRAM(offset + uint16(tileY)*32 + uint16(tileX))
	addr := p.tilePatternTableAddr() + uint16(id)*16

	for y := 0; y < 8; y++ {
		p1 := p.readVRAM(addr + uint16(y) + 0)
		p2 := p.readVRAM(addr + uint16(y) + 8)

		for x := 0; x < 8; x++ {
			pixel := p1 & (0x80 >> x) >> (7 - x) << 0
			pixel |= (p2 & (0x80 >> x) >> (7 - x)) << 1
			tile.Pixels[x][y] = pixel // two-bit pixel value (0-3)
		}
	}

	// two-bit palette ID (0-3)
	blockId := uint16(tileX%4/2) + uint16(tileY%4/2)*2
	tile.PaletteID = (attr >> (blockId * 2)) & 0x03

	return tile
}

func (p *PPU) renderTileScanline() {
	var (
		fineX = int(p.ScrollX % 8)
		fineY = int(p.ScrollY % 8)
	)

	var (
		frameY = (p.scanline + fineY) % 248
		tileY  = int(frameY) / 8
		pixelY = int(frameY) % 8
	)

	for tileX := 0; tileX < 32; tileX++ {
		tile := p.fetchTile(tileX, tileY)

		for pixelX := 0; pixelX < 8; pixelX++ {
			frameX := tileX*8 + pixelX

			pixel := tile.Pixels[pixelX][pixelY]
			if pixel == 0 {
				continue
			}

			// To simulate scrolling, we need to offset the tile's position by the fine
			// scroll values. This is not how the PPU does it, but it seems to work.
			x, y := frameX-fineX, frameY-fineY
			if x < 0 || y < 0 || x >= 256 || y >= 240 {
				continue
			}

			addr := 0x3F00 + uint16(tile.PaletteID)*4 + uint16(pixel)
			p.Frame[x][y] = Colors[p.readVRAM(addr)]
		}
	}

	// Simulate the smooth wrap-around effect by rendering the first 8 pixels of the
	// tiles from the next name table for the rightmost 8 pixels of the frame, if we
	// are scrolling horizontally. This is a hack, but it works for most games.
	if fineX > 0 {
		tile := p.fetchTile(32, tileY)

		for pixelX := 0; pixelX < fineX; pixelX++ {
			pixel := tile.Pixels[pixelX][pixelY]
			if pixel == 0 {
				continue
			}

			addr := 0x3F00 + uint16(tile.PaletteID)*4 + uint16(pixel)
			offsetX := 32*8 - fineX + pixelX

			x, y := offsetX, frameY-fineY
			if x < 0 || y < 0 || x >= 256 || y >= 240 {
				continue
			}

			p.Frame[x][y] = Colors[p.readVRAM(addr)]
		}
	}
}
