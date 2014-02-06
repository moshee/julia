package main

import (
	"flag"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"runtime"
	"strconv"
)

var (
	cpus    = runtime.NumCPU()
	listen  = flag.String("listen", ":8080", "Port to listen on")
	itercap = flag.Int("cap.iters", 0, "Cap max iterations to value (<= 0 for uncapped)")
	rescap  = flag.Int("cap.res", 0, "Cap max width or height to value (<=0 for uncapped)")
)

func init() {
	flag.Parse()
	runtime.GOMAXPROCS(cpus)
}

func main() {
	index, err := ioutil.ReadFile("index.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(index)
	})

	var (
		red   = color.RGBA{255, 0, 0, 255}
		green = color.RGBA{0, 255, 0, 255}
		blue  = color.RGBA{0, 0, 255, 255}
	)

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
			ry := int(H * (1 - R(args, p)))
			img.Set(x, ry, red)
			img.Set(x, ry+1, red)
			gy := int(H * (1 - G(args, p)))
			img.Set(x, gy, green)
			img.Set(x, gy+1, green)
			by := int(H * (1 - B(args, p)))
			img.Set(x, by, blue)
			img.Set(x, by+1, blue)
		}

		png.Encode(w, img)
	})

	http.HandleFunc("/julia.png", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		img := makeFractal(newJulia(r.Form), r)
		if err := png.Encode(w, img); err != nil {
			log.Printf("Error encoding png: %v", err)
		}
	})

	http.HandleFunc("/julia.jpg", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		img := makeFractal(newJulia(r.Form), r)
		if err := jpeg.Encode(w, img, &jpeg.Options{Quality: 90}); err != nil {
			log.Printf("Error encoding jpeg: %v", err)
		}
	})

	http.HandleFunc("/mandelbrot.png", func(w http.ResponseWriter, r *http.Request) {
		img := makeFractal(newMandelbrot(), r)
		if err := png.Encode(w, img); err != nil {
			log.Printf("Error encoding png: %v", err)
		}
	})

	http.HandleFunc("/mandelbrot.jpg", func(w http.ResponseWriter, r *http.Request) {
		img := makeFractal(newMandelbrot(), r)
		if err := jpeg.Encode(w, img, &jpeg.Options{Quality: 90}); err != nil {
			log.Printf("Error encoding jpeg: %v", err)
		}
	})

	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Base(r.URL.Path))
	})

	log.Println("Listening on", *listen)
	log.Println(http.ListenAndServe(*listen, nil))
}
