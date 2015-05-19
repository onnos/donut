// Go Donut!
//
// originals:
// http://www.a1k0n.net/2011/07/20/donut-math.html  (original by Andy Sloane)
// https://github.com/GaryBoone/GoDonut             (donut.go by Gary Boone)
//
// This termbox version by Onno Siemens.
//
// Hint: hit enter while running. ESCape to quit.
//

package main

import (
	"github.com/nsf/termbox-go"
	"math"
	"time"
)

const frame_delay = 20
const theta_spacing = 0.02
const phi_spacing = 0.02

const R1 = 0.9
const R2 = 1.8
const K2 = 8.0

type Screen struct {
	dim        int
	lum24      [][]int
	data       [][]byte
	zoom       float64
	transition int
}

func newZBuffer(d int) *[][]float64 {
	b := make([][]float64, d)
	for i := range b {
		b[i] = make([]float64, d)
	}
	return &b
}

func newScreen(d int) *Screen {
	b := make([][]byte, d)
	c := make([][]int, d)
	for i := range b {
		b[i] = make([]byte, d)
		c[i] = make([]int, d)
	}
	a := float64(0)
	z := int(0)
	return &Screen{d, c, b, a, z}
}

func (screen Screen) render(rendermode int) {
	for x, _ := range screen.data {
		for y, _ := range screen.data[x] {
			if screen.lum24[x][y] == ' ' {
				screen.draw(x, y, ' ', 0, 0)
			} else {
				switch rendermode {
				case 1:
					termbox.SetOutputMode(termbox.OutputGrayscale)
					screen.draw(x, y, ' ', screen.lum24[x][y], screen.lum24[x][y])
				case 2:
					termbox.SetOutputMode(termbox.OutputGrayscale)
					screen.draw(x, y, ' ', screen.lum24[x][y], screen.lum24[x][y]+250)
				case 3:
					termbox.SetOutputMode(termbox.Output216)
					screen.draw(x, y, rune(screen.data[x][y]), screen.lum24[x][y]/3+1, screen.lum24[x][y]/4)
				case 4:
					termbox.SetOutputMode(termbox.Output216)
					screen.draw(x, y, ' ', 0, screen.lum24[x][y])
				default:
					termbox.SetOutputMode(termbox.OutputGrayscale)
					screen.draw(x, y, rune(screen.data[x][y]), screen.lum24[x][y], 0)
				}
			}
		}
	}
	screen.clear()
}

func (screen *Screen) draw(x, y int, char rune, fg, bg int) {
	termbox.SetCell(x*2, y, char, termbox.Attribute(fg), termbox.Attribute(bg))
	termbox.SetCell(x*2-1, y, char, termbox.Attribute(fg), termbox.Attribute(bg))
}

func (screen *Screen) clear() {
	for i, _ := range screen.data {
		for j, _ := range screen.data[i] {
			screen.data[i][j] = ' '
			screen.lum24[i][j] = ' '
		}
	}
}

func (screen *Screen) computeFrame(A, B, K1 float64) {

	// precompute sines and cosines of A and B
	cosA := math.Cos(A)
	sinA := math.Sin(A)
	cosB := math.Cos(B)
	sinB := math.Sin(B)

	zbuffer := newZBuffer(screen.dim)

	// theta goes around the cross-sectional circle of a torus
	for theta := 0.0; theta < 2.0*math.Pi; theta += theta_spacing {
		// precompute sines and cosines of theta
		costheta := math.Cos(theta)
		sintheta := math.Sin(theta)

		// phi goes around the center of revolution of a torus
		for phi := 0.0; phi < 2.0*math.Pi; phi += phi_spacing {
			// precompute sines and cosines of phi
			cosphi := math.Cos(phi)
			sinphi := math.Sin(phi)

			// the x,y coordinate of the circle, before revolving (factored out of the above equations)
			circlex := R2 + R1*costheta
			circley := R1 * sintheta

			// final 3D (x,y,z) coordinate after rotations, directly from our math above
			x := circlex*(cosB*cosphi+sinA*sinB*sinphi) - circley*cosA*sinB
			y := circlex*(sinB*cosphi-sinA*cosB*sinphi) + circley*cosA*cosB
			z := K2 + cosA*circlex*sinphi + circley*sinA
			ooz := 1 / z // "one over z"

			// x and y projection.  note that y is negated here, because y goes up in
			// 3D space but down on 2D displays.
			xp := int(float64(screen.dim)/2.0 + K1*ooz*x)
			yp := int(float64(screen.dim)/2.0 - K1*ooz*y)

			// calculate luminance.  ugly, but correct.
			L := cosphi*costheta*sinB - cosA*costheta*sinphi - sinA*sintheta +
				cosB*(cosA*sintheta-costheta*sinA*sinphi)
			// L ranges from -sqrt(2) to +sqrt(2).  If it's < 0, the surface is
			// pointing away from us, so we won't bother trying to plot it.
			if L > 0 {
				// test against the z-buffer.  larger 1/z means the pixel is closer to
				// the viewer than what's already plotted.
				if ooz > (*zbuffer)[yp][xp] {
					(*zbuffer)[yp][xp] = ooz
					ascii_index := int(L * 8.0)  // this brings L into the range 0..11 (8*sqrt(2) = 11.3)
					lum24 := int((L * 16.0) + 1) // this brings L into the range 1..24 (16*sqrt(2) + 1 = 23.6)
					// now we lookup the character corresponding to the luminance and plot it in our output:
					screen.data[yp][xp] = ".,-~:;=!*#$@"[ascii_index]
					screen.lum24[yp][xp] = lum24
				}
			}
		}
	}
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	event_queue := make(chan termbox.Event)
	go func() {
		for {
			event_queue <- termbox.PollEvent()
		}
	}()
	w, h := termbox.Size()
	dim := int(math.Min(float64(w), float64(h)))
	screen := newScreen(dim)
	rendermode := int(0)
	termbox.SetOutputMode(termbox.OutputGrayscale)

	// Calculate K1 based on screen size: the maximum x-distance occurs roughly at
	// the edge of the torus, which is at x=R1+R2, z=0.  we want that to be
	// displaced 3/8ths of the width of the screen, which is 3/4th of the way from
	// the center to the side of the screen.
	//screen.dim*3/8 = K1*(R1+R2)/(K2+0)
	//screen.dim*K2*3/(8*(R1+R2)) = K1
	A, B, K1 := 1.0, 1.0, float64(screen.dim)*K2*3.0/(8.0*(R1+R2))

	// start zoomed out
	screen.zoom = K1
	K1 = 0
	screen.transition = 2

loop:
	for {
		select {
		case ev := <-event_queue:
			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyEsc {
				break loop
			}
			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyEnter {

				screen.transition = 1
			}

		default:
			// rotate the torus
			A += 0.07
			B += 0.03

			// zoom out if transitioning to new rendermode
			if screen.transition == 1 {
				K1--
			}
			// all the way zoomed out, switch rendermodes
			if K1 < 0 {
				screen.transition = 2
				if rendermode == 4 {
					rendermode = 0
				} else {
					rendermode++
				}
			}
			// zoom back in
			if screen.transition == 2 {
				if K1 < screen.zoom {
					K1++
				}
				if K1 >= screen.zoom {
					K1 = screen.zoom
					screen.transition = 0
				}
			}
			// draw all the things
			screen.computeFrame(A, B, K1)
			screen.render(rendermode)
			termbox.Flush()
			time.Sleep(frame_delay * time.Millisecond)
		}
	}
}
