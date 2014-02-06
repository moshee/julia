package main

import (
	"image"
	"image/color"
	"math"
	"net/http"
	"net/url"
	"strconv"
)

type Fractal interface {
	At(o *Opts, x, y float64) color.Color
}

type Opts struct {
	xmax, ymax    float64
	distMax       float64
	width, height int
	coloringFunc  ColoringFunc
	paletteFunc   PaletteFunc
	args          map[rune]float64
	maxIters      int
	rePos, imPos  float64
}

func makeFractal(f Fractal, r *http.Request) image.Image {
	args := make(map[rune]float64)
	letters := "abcdefghijkl"
	for i, ch := range letters {
		val := r.FormValue(letters[i : i+1])
		args[ch], _ = strconv.ParseFloat(val, 64)
	}

	var (
		scale, _    = strconv.ParseFloat(r.FormValue("scale"), 64)
		width, _    = strconv.Atoi(r.FormValue("width"))
		height, _   = strconv.Atoi(r.FormValue("height"))
		maxIters, _ = strconv.Atoi(r.FormValue("iterations"))
		rePos, _    = strconv.ParseFloat(r.FormValue("rePos"), 64)
		imPos, _    = strconv.ParseFloat(r.FormValue("imPos"), 64)
	)
	if *itercap > 0 && maxIters > *itercap {
		maxIters = *itercap
	}
	if *rescap > 0 {
		if width > *rescap {
			width = *rescap
		}
		if height > *rescap {
			height = *rescap
		}
	}
	scale = math.Exp2(-scale / 2)

	ymax := scale
	xmax := scale
	if width >= height {
		xmax *= float64(width) / float64(height)
	} else {
		ymax *= float64(height) / float64(width)
	}

	distMax := (xmax * 2) / float64(width)

	coloringFunc := distance
	switch r.FormValue("coloring") {
	case "distance":
		coloringFunc = distance
	case "escape":
		coloringFunc = escapeTime
	}

	paletteFunc := gray
	switch r.FormValue("palette") {
	case "gray":
		paletteFunc = gray
	case "color":
		paletteFunc = palette
	}

	o := &Opts{
		xmax, ymax, distMax,
		width, height,
		coloringFunc,
		paletteFunc,
		args,
		maxIters,
		rePos, imPos,
	}

	img := image.NewRGBA(image.Rect(0, 0, o.width, o.height))

	done := make(chan struct{})

	for cpu := 0; cpu < cpus; cpu++ {
		go func(n int) {
			// render in staggered 1 pixel rows interleaved so each goroutine
			// gets approx the same amount of work
			for py := n; py < o.height; py += cpus {
				y := o.ymax*2.0*float64(py)/float64(o.height) - o.ymax - o.imPos

				for px := 0; px < o.width; px++ {
					x := o.xmax*2.0*float64(px)/float64(o.width) - o.xmax + o.rePos

					pix := f.At(o, x, y)
					img.Set(px, py, pix)
				}
			}
			done <- struct{}{}
		}(cpu)
	}

	for i := 0; i < cpus; i++ {
		<-done
	}

	return img
}

func newJulia(v url.Values) Fractal {
	var (
		cre, _ = strconv.ParseFloat(v.Get("re"), 64)
		cim, _ = strconv.ParseFloat(v.Get("im"), 64)
	)

	return &JuliaSet{complex(cre, cim)}
}

type JuliaSet struct {
	c complex128
}

func (s *JuliaSet) At(o *Opts, x, y float64) color.Color {
	f := o.coloringFunc(o, complex(x, y), s.c)
	return o.paletteFunc(o, f)
}

func escape(z complex128) bool {
	x := real(z)
	y := imag(z)
	return (x*x + y*y) > 4.0
}

func newMandelbrot() Fractal {
	return MandelbrotSet{}
}

type MandelbrotSet struct{}

func (s MandelbrotSet) At(o *Opts, x, y float64) color.Color {
	f := o.coloringFunc(o, 0, complex(x, y))
	return o.paletteFunc(o, f)
}
