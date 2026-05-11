// Command make-icons renders PWA install icons (192px and 512px) as PNGs from
// a procedural castle silhouette. Run once when the design changes:
//
//	go run ./cmd/make-icons --out ../frontend/public
//
// The output is committed to the repo. No native dependencies.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

var (
	night   = color.RGBA{0x1a, 0x1f, 0x2e, 0xff}
	candle  = color.RGBA{0xe9, 0xd8, 0xa6, 0xff}
	emberGl = color.RGBA{0xca, 0x6f, 0x1e, 0xff}
)

func main() {
	out := flag.String("out", ".", "directory to write icon-192.png and icon-512.png")
	flag.Parse()

	for _, size := range []int{192, 512} {
		img := drawCastle(size)
		path := filepath.Join(*out, fmt.Sprintf("icon-%d.png", size))
		f, err := os.Create(path)
		if err != nil {
			die(err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			die(err)
		}
		if err := f.Close(); err != nil {
			die(err)
		}
		fmt.Println("wrote", path)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "make-icons:", err)
	os.Exit(1)
}

// drawCastle renders a square icon: rounded-corner night background, a beige
// castle silhouette with three crenellated towers, and a small ember light
// behind the open gate to hint at warmth.
func drawCastle(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// rounded background — corner radius ~18% of size
	r := size * 18 / 100
	fillRounded(img, image.Rect(0, 0, size, size), r, night)

	// castle silhouette occupies the middle 75% of the canvas
	pad := size / 8
	cb := image.Rect(pad, size*40/100, size-pad, size-pad)

	// main wall
	wallTop := cb.Min.Y + (cb.Dy() * 30 / 100)
	fillRect(img, image.Rect(cb.Min.X, wallTop, cb.Max.X, cb.Max.Y), candle)

	// crenellations on top of the main wall (5 teeth)
	tooth := cb.Dx() / 11
	gap := tooth
	x := cb.Min.X
	toothTop := cb.Min.Y + (cb.Dy() * 15 / 100)
	for i := 0; i < 5; i++ {
		fillRect(img, image.Rect(x, toothTop, x+tooth, wallTop), candle)
		x += tooth + gap
	}

	// three towers: left, center (taller), right
	towerW := cb.Dx() * 14 / 100
	// left
	lt := image.Rect(cb.Min.X, cb.Min.Y+cb.Dy()*5/100, cb.Min.X+towerW, cb.Max.Y)
	fillRect(img, lt, candle)
	// right
	rt := image.Rect(cb.Max.X-towerW, cb.Min.Y+cb.Dy()*5/100, cb.Max.X, cb.Max.Y)
	fillRect(img, rt, candle)
	// center tower (tallest)
	cx := (cb.Min.X + cb.Max.X) / 2
	ct := image.Rect(cx-towerW/2, cb.Min.Y, cx+towerW/2, cb.Max.Y)
	fillRect(img, ct, candle)

	// gate (cutout — paint background color over the wall)
	gateW := towerW
	gateH := cb.Dy() * 40 / 100
	gate := image.Rect(cx-gateW/2, cb.Max.Y-gateH, cx+gateW/2, cb.Max.Y)
	// round the top of the gate by skipping pixels above a quarter-circle
	for y := gate.Min.Y; y < gate.Max.Y; y++ {
		for x := gate.Min.X; x < gate.Max.X; x++ {
			if y < gate.Min.Y+gateW/2 {
				dx := x - cx
				dy := y - (gate.Min.Y + gateW/2)
				if dx*dx+dy*dy > (gateW/2)*(gateW/2) {
					continue
				}
			}
			img.Set(x, y, night)
		}
	}

	// ember dot inside the gate, near the bottom
	emberR := gateW / 5
	emberCY := gate.Max.Y - emberR*3/2
	for y := emberCY - emberR; y <= emberCY+emberR; y++ {
		for x := cx - emberR; x <= cx+emberR; x++ {
			dx := x - cx
			dy := y - emberCY
			if dx*dx+dy*dy <= emberR*emberR {
				img.Set(x, y, emberGl)
			}
		}
	}

	return img
}

func fillRect(img *image.RGBA, r image.Rectangle, c color.Color) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

// fillRounded draws a filled rectangle with rounded corners of radius rad.
func fillRounded(img *image.RGBA, r image.Rectangle, rad int, c color.Color) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if inRoundedRect(x, y, r, rad) {
				img.Set(x, y, c)
			}
		}
	}
}

func inRoundedRect(x, y int, r image.Rectangle, rad int) bool {
	// the four corner squares need a circle test; the rest is inside.
	switch {
	case x < r.Min.X+rad && y < r.Min.Y+rad:
		dx := r.Min.X + rad - x
		dy := r.Min.Y + rad - y
		return dx*dx+dy*dy <= rad*rad
	case x >= r.Max.X-rad && y < r.Min.Y+rad:
		dx := x - (r.Max.X - rad - 1)
		dy := r.Min.Y + rad - y
		return dx*dx+dy*dy <= rad*rad
	case x < r.Min.X+rad && y >= r.Max.Y-rad:
		dx := r.Min.X + rad - x
		dy := y - (r.Max.Y - rad - 1)
		return dx*dx+dy*dy <= rad*rad
	case x >= r.Max.X-rad && y >= r.Max.Y-rad:
		dx := x - (r.Max.X - rad - 1)
		dy := y - (r.Max.Y - rad - 1)
		return dx*dx+dy*dy <= rad*rad
	}
	return true
}
