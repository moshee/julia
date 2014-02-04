package main

import (
	"flag"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"runtime"
	"strconv"
	"sync"
)

var cpus = runtime.NumCPU()

func init() {
	runtime.GOMAXPROCS(cpus)
}

func main() {
	listen := flag.String("listen", ":8080", "Port to listen on")
	flag.Parse()

	index, err := ioutil.ReadFile("index.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(index)
	})

	http.HandleFunc("/palette.png", func(w http.ResponseWriter, r *http.Request) {
		const (
			W = 512
			H = 64
		)
		img := image.NewRGBA(image.Rect(0, 0, W, H))
		args := make(map[rune]float64)
		letters := "abcdefghijkl"
		for i, ch := range letters {
			val := r.FormValue(letters[i : i+1])
			args[ch], _ = strconv.ParseFloat(val, 64)
		}

		for x := 0; x < W; x++ {
			p := float64(x) / W
			for y := 0; y < H; y++ {
				img.Set(x, y, color.RGBA{
					uint8(255 * R(args, p)),
					uint8(255 * G(args, p)),
					uint8(255 * B(args, p)),
					255,
				})
			}
			img.Set(x, int(H*(1-R(args, p))), color.RGBA{255, 0, 0, 255})
			img.Set(x, int(H*(1-R(args, p)))+1, color.RGBA{255, 255, 255, 255})
			img.Set(x, int(H*(1-R(args, p)))-1, color.RGBA{255, 255, 255, 255})

			img.Set(x, int(H*(1-G(args, p))), color.RGBA{0, 255, 0, 255})
			img.Set(x, int(H*(1-G(args, p)))+1, color.RGBA{255, 255, 255, 255})
			img.Set(x, int(H*(1-G(args, p)))-1, color.RGBA{255, 255, 255, 255})

			img.Set(x, int(H*(1-B(args, p))), color.RGBA{0, 0, 255, 255})
			img.Set(x, int(H*(1-B(args, p)))+1, color.RGBA{255, 255, 255, 255})
			img.Set(x, int(H*(1-B(args, p)))-1, color.RGBA{255, 255, 255, 255})
		}

		png.Encode(w, img)
	})

	http.HandleFunc("/julia.png", func(w http.ResponseWriter, r *http.Request) {
		img := makeJulia(r)
		if err := png.Encode(w, img); err != nil {
			log.Printf("Error encoding png: %v", err)
		}
	})

	http.HandleFunc("/julia.jpg", func(w http.ResponseWriter, r *http.Request) {
		img := makeJulia(r)
		if err := jpeg.Encode(w, img, &jpeg.Options{Quality: 90}); err != nil {
			log.Printf("Error encoding jpeg: %v", err)
		}
	})

	log.Println("Listening on", *listen)
	log.Println(http.ListenAndServe(*listen, nil))
}

func makeJulia(r *http.Request) image.Image {
	args := make(map[rune]float64)
	letters := "abcdefghijkl"
	for i, ch := range letters {
		val := r.FormValue(letters[i : i+1])
		args[ch], _ = strconv.ParseFloat(val, 64)
	}

	var (
		cre, _      = strconv.ParseFloat(r.FormValue("re"), 64)
		cim, _      = strconv.ParseFloat(r.FormValue("im"), 64)
		scale, _    = strconv.ParseFloat(r.FormValue("scale"), 64)
		width, _    = strconv.Atoi(r.FormValue("width"))
		height, _   = strconv.Atoi(r.FormValue("height"))
		maxIters, _ = strconv.Atoi(r.FormValue("iterations"))
		center, _   = strconv.ParseBool(r.FormValue("center"))
		rePos, _    = strconv.ParseFloat(r.FormValue("rePos"), 64)
		imPos, _    = strconv.ParseFloat(r.FormValue("imPos"), 64)
	)
	scale = math.Exp2(-scale / 2)
	if center {
		rePos = cre
		imPos = cim
	}

	c := complex(cre, cim)
	ymax := scale
	xmax := scale
	if width >= height {
		xmax *= float64(width) / float64(height)
	} else {
		ymax *= float64(height) / float64(width)
	}

	distMax := (xmax * 2) / float64(width)

	coloring := r.FormValue("coloring")
	coloringFunc := (*JuliaSet).distance
	switch coloring {
	case "distance":
		coloringFunc = (*JuliaSet).distance
	case "escape":
		coloringFunc = (*JuliaSet).escapeTime
	}

	paletteType := r.FormValue("palette")
	paletteFunc := (*JuliaSet).gray
	switch paletteType {
	case "gray":
		paletteFunc = (*JuliaSet).gray
	case "color":
		paletteFunc = (*JuliaSet).palette
	}

	s := &JuliaSet{
		c,
		xmax, ymax, distMax,
		width, height,
		coloringFunc,
		paletteFunc,
		args,
		maxIters,
		rePos, imPos,
	}
	return s.run()
}

func f(z, c complex128) complex128 {
	return z*z + c
	//return cmplx.Sinh(z*z) + cmplx.Exp(z) + c
	//return (z*z-z)/(2*cmplx.Log(z)) + c
	//return z*z*z + c
	//return 1/(z*z) + c
}
func fp(z, c complex128) complex128 {
	return 2 * z
	//return 2*z*cmplx.Cosh(z*z) + cmplx.Exp(z)
	//return ((2*z-1)*(2*cmplx.Log(z)-1) - 2*z + 2) / cmplx.Pow(2*cmplx.Log(z), 2+0i)
	//return 3 * z * z
	//return 2 / (z * z * z)
}

type JuliaSet struct {
	c             complex128
	xmax, ymax    float64
	distMax       float64
	width, height int
	coloringFunc  func(*JuliaSet, complex128) float64
	paletteFunc   func(*JuliaSet, float64) color.Color
	args          map[rune]float64
	maxIters      int
	rePos, imPos  float64
}

func (s *JuliaSet) run() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, s.width, s.height))

	wg := new(sync.WaitGroup)
	wg.Add(cpus)
	slice := s.width / cpus

	for cpu := 0; cpu < cpus; cpu++ {
		go func(n int) {
			for py := 0; py < s.height; py++ {
				im := s.ymax*2.0*float64(py)/float64(s.height) - s.ymax - s.imPos

				for px := n * slice; px < (n+1)*slice; px++ {
					re := s.xmax*2.0*float64(px)/float64(s.width) - s.xmax + s.rePos
					pixel := s.paletteFunc(s, s.coloringFunc(s, complex(re, im)))

					img.Set(px, py, pixel)
				}
			}
			wg.Done()
		}(cpu)
	}

	wg.Wait()

	return img
}

func escape(z complex128) bool {
	x := real(z)
	y := imag(z)
	return (x*x + y*y) > 4.0
}

func (s *JuliaSet) distance(z complex128) float64 {
	zp := 1 + 0i // z' = f'(z_n)
	zn := z      // next z_n value

derivative:
	for j := 0; j < s.maxIters; j++ {
		zn = f(z, s.c)
		zp *= fp(z, s.c)
		z = zn
		if cmplx.Abs(zp) > 1.0e60 {
			break derivative
		}
	}

	za := cmplx.Abs(z)
	dist := za * (math.Log(za) / cmplx.Abs(zp))
	return -1/(3*(dist/s.distMax)+1) + 1
}

func (s *JuliaSet) escapeTime(z complex128) float64 {
	i := 0
	for i < s.maxIters {
		z = f(z, s.c)
		if escape(z) {
			break
		}
		i++
	}

	p := 1 - float64(i)/float64(s.maxIters)

	return p
}

func (s *JuliaSet) gray(x float64) color.Color {
	return color.Gray{uint8(255 * x)}
}

func (s *JuliaSet) palette(x float64) color.Color {
	return color.RGBA{
		uint8(255 * R(s.args, x)),
		uint8(255 * G(s.args, x)),
		uint8(255 * B(s.args, x)),
		255,
	}
}

func R(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['a']*x+args['b']))) -
		(1 / (1 + math.Exp(args['c']*x+args['d'])))
}

func G(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['e']*x+args['f']))) -
		(1 / (1 + math.Exp(args['g']*x+args['h'])))
}

func B(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['i']*x+args['j']))) -
		(1 / (1 + math.Exp(args['k']*x+args['l'])))
}
