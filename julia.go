package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, index)
	})

	http.HandleFunc("/palette", func(w http.ResponseWriter, r *http.Request) {
		const (
			W = 512
			H = 128
		)
		img := image.NewRGBA(image.Rect(0, 0, W, H))
		args := make(map[rune]float64)
		letters := "abcdefghi"
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

	http.HandleFunc("/julia", func(w http.ResponseWriter, r *http.Request) {
		args := make(map[rune]float64)
		letters := "abcdefghi"
		for i, ch := range letters {
			val := r.FormValue(letters[i : i+1])
			args[ch], _ = strconv.ParseFloat(val, 64)
		}

		cre := cheat(r, "re")
		cim := cheat(r, "im")
		scale := math.Exp2(-cheat(r, "scale") / 2)
		width, _ := strconv.Atoi(r.FormValue("width"))
		height, _ := strconv.Atoi(r.FormValue("height"))
		maxIters, _ := strconv.Atoi(r.FormValue("iterations"))

		c := complex(cre, cim)
		ymax := scale
		xmax := scale * float64(width) / float64(height)
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
		}
		s.run(w)
	})
	fmt.Println(http.ListenAndServe(*listen, nil))
}

func cheat(r *http.Request, s string) float64 {
	n, _ := strconv.ParseFloat(r.FormValue(s), 64)
	return n
}

func f(z, c complex128) complex128 {
	return z*z + c
}
func fp(z complex128) complex128 {
	return 2 * z
}

type JuliaSet struct {
	c                   complex128
	xmax, ymax, distMax float64
	width, height       int
	coloringFunc        func(*JuliaSet, complex128) color.Color
	paletteFunc         func(*JuliaSet, float64) color.Color
	args                map[rune]float64
	maxIters            int
}

func (s *JuliaSet) run(w io.Writer) {
	img := image.NewRGBA(image.Rect(0, 0, s.width, s.height))

	wg := new(sync.WaitGroup)
	wg.Add(cpus)
	slice := s.width / cpus

	for cpu := 0; cpu < cpus; cpu++ {
		go func(n int) {
			for py := 0; py < s.width; py++ {
				im := s.ymax*2.0*float64(py)/float64(s.height) - s.ymax
				for px := n * slice; px < (n+1)*slice; px++ {
					re := s.xmax*2.0*float64(px)/float64(s.width) - s.xmax

					pixel := s.coloringFunc(s, complex(re, im))

					img.Set(px, py, pixel)
				}
			}
			wg.Done()
		}(cpu)
	}

	wg.Wait()

	if err := png.Encode(w, img); err != nil {
		panic(err)
	}
}

func escape(z complex128) bool {
	x := real(z)
	y := imag(z)
	return (x*x + y*y) > 4.0
}

func (s *JuliaSet) distance(z complex128) color.Color {
	zp := complex(1.0, 0.0) // z' = f'(z_n)
	zn := z                 // next z_n value

derivative:
	for j := 0; j < s.maxIters; j++ {
		zn = f(z, s.c)
		zp *= fp(z)
		z = zn
		if cmplx.Abs(zp) > 1.0e60 {
			break derivative
		}
	}

	za := cmplx.Abs(z)
	dist := za * (math.Log(za) / cmplx.Abs(zp))
	p := 1 - math.Exp(-4*dist/(s.distMax))
	return s.paletteFunc(s, p)
}

func (s *JuliaSet) escapeTime(z complex128) color.Color {
	i := 0
	for i < s.maxIters {
		z = f(z, s.c)
		if escape(z) {
			break
		}
		i++
	}

	p := 1 - float64(i)/float64(s.maxIters)

	return s.paletteFunc(s, p)
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
	return 1 - (1 / (1 + math.Exp(args['a']*x+args['b'])))
}

func G(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['c']*x+args['d']))) - (1 / (1 + math.Exp(args['e']*x+args['f'])))
}

func B(args map[rune]float64, x float64) float64 {
	return 1 / (args['g'] + math.Exp(args['h']*x+args['i']))
}

var index = `<!doctype html>
<head>
	<title>lol</title>
	<style>
		input[type=number] { width: 5em }
		img { display: block }
		form, #images {
			display: inline-block;
			vertical-align: top;
			padding: 32px;
		}
	</style>
</head>
<body>
	<div id=images>
		<img id=fractal>
		<img id=palette>
	</div>
	<form>
		<p>R = 1 - (1 / (1 + <i>e</i><sup><i><b>a</b>x</i> + <i><b>b</b></i></sup>))</p>
		<p>G = (1 / (1 + <i>e</i><sup><i><b>c</b>x</i> + <i><b>d</b></i></sup>)) - (1 / (1 + <i>e</i><sup><i><b>e</b>x</i> + <i><b>f</b></i></sup>))</p>
		<p>B = 1 - (1 / (<i><b>g</b></i> + <i>e</i><sup><i><b>h</b>x</i> + <i><b>i</b></i></sup>))</p>
		<p>ƒ<sub>c</sub>(<i>z</i>) = <i>z</i><sup>2</sup> + <i>c</i></p>
		<div>
			<label for=a><i>a =</i></label>
			<input name=a type=number value=0 step=1>
			<label for=b><i>b =</i></label>
			<input name=b type=number value=8 step=1>
		</div>
		<div>
			<label for=c><i>c =</i></label>
			<input name=c type=number value=0 step=1>
			<label for=d><i>d =</i></label>
			<input name=d type=number value=0 step=1>
			<label for=e><i>e =</i></label>
			<input name=e type=number value=0 step=1>
			<label for=f><i>f =</i></label>
			<input name=f type=number value=0 step=1>
		</div>
		<div>
			<label for=g><i>g =</i></label>
			<input name=g type=number value=0 step=1>
			<label for=h><i>h =</i></label>
			<input name=h type=number value=0 step=1>
			<label for=i><i>i =</i></label>
			<input name=i type=number value=0 step=1>
		</div>
		<div>
			<span><i>c</i> = <input name=re type=number value=0 step=0.01> +
			<input name=im type=number value=0 step=0.01><i>i</i>
			(scale = <input name=scale type=number step=1>)
		</div>
		<hr>
		<div>
			<label for=width>width =</label>
			<input name=width type=number>
			<label for=height>height =</label>
			<input name=height type=number>
			<label for=iterations>iterations =</label>
			<input name=iterations type=number>
		</div>
		<div class=radio>
			<label for=coloring>Coloring function:</label>
			<input name=coloring type=radio value=escape checked>escape time
			<input name=coloring type=radio value=distance>distance
		</div>
		<div class=radio>
			<label for=palette>Palette:</label>
			<input name=palette type=radio value=color checked>color
			<input name=palette type=radio value=gray>gray
		</div>
		<button id=submit type=button>render</button>
		<button id=random type=button>pick a random c</button>
	</form>
	<script>
		var vars = {
			"a": -30, "b":  10,
			"c": -30, "d":   3, "e": -20, "f": 12,
			"g":   2, "h": -20, "i":  13,
			"re": Math.random()*2-1,
			"im": Math.random()*2-1,
			"scale": 0, "width": 512, "height": 512,
			"iterations": 255
		};

		var colorsDirty = true;
		var fractal, palette, inputs;
		var submit;

		function radioValue(s) {
			var inputs = document.getElementsByName(s);
			for (var i = 0, input; input = inputs[i]; i++) {
				if (input.checked) {
					return input.value;
				}
			}
		}

		function unstick() {
			for (var i = 0, input; input = inputs[i]; i++) {
				input.removeAttribute('disabled');
			}
			submit.removeAttribute('disabled');
			submit.innerText = "render";
		}

		function updateImage() {
			var q = [];
			for (var key in vars) {
				q.push(key + "=" + vars[key]);
			}
			q.push('palette=' + radioValue('palette'));
			q.push('coloring=' + radioValue('coloring'));
			var args = q.join("&");

			submit.setAttribute('disabled');
			submit.innerText = "rendering...";

			fractal.setAttribute("src", "/julia?" + args);
			fractal.removeEventListener("load");
			fractal.addEventListener("load", unstick);
			if (colorsDirty) {
				colorsDirty = false;
				palette.setAttribute("src", "/palette?" + args);
			}
		}

		window.addEventListener("DOMContentLoaded", function() {
			for (var key in vars) {
				document.querySelector("[name=" + key + "]").value = vars[key];
			}

			fractal = document.querySelector('#fractal');
			palette = document.querySelector('#palette');
			inputs = document.querySelectorAll('input');
			submit = document.querySelector('#submit');

			for (var i = 0, input; input = inputs[i]; i++) {
				input.addEventListener("input", function(e) {
					var name = e.target.getAttribute("name");
					vars[name] = e.target.value;
					// the colors all have 1 length names
					if (name.length == 1) {
						colorsDirty = true;
					}
					// update immediately unless it's something potentially very expensive
					if (name != "width" && name != "height" && name != "iterations") {
						e.target.setAttribute('disabled');
						updateImage();
					}
				});
			}

			updateImage();
			submit.addEventListener("click", updateImage);
			document.querySelector("#random").addEventListener("click", function() {
				document.querySelector('[name=re]').value = vars['re'] = Math.random()*2-1;
				document.querySelector('[name=im]').value = vars['im'] = Math.random()*2-1;
				updateImage();
			});
		});
	</script>
</body>`
