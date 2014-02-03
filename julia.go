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

	http.HandleFunc("/palette.png", func(w http.ResponseWriter, r *http.Request) {
		const (
			W = 512
			H = 128
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
		args := make(map[rune]float64)
		letters := "abcdefghijkl"
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
	c                   complex128
	xmax, ymax, distMax float64
	width, height       int
	coloringFunc        func(*JuliaSet, complex128) float64
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

					pixel := s.paletteFunc(s, s.coloringFunc(s, complex(re, im)))

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
	//p := math.Exp(-dist / (s.distMax))
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
	return (1 / (1 + math.Exp(args['a']*x+args['b']))) - (1 / (1 + math.Exp(args['c']*x+args['d'])))
}

func G(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['e']*x+args['f']))) - (1 / (1 + math.Exp(args['g']*x+args['h'])))
}

func B(args map[rune]float64, x float64) float64 {
	return (1 / (1 + math.Exp(args['i']*x+args['j']))) - (1 / (1 + math.Exp(args['k']*x+args['l'])))
}

var index = `<!doctype html>
<head>
	<title>Julia Set</title>
	<style>
		input[type=number] {
			width: 5em;
			border: none;
			border-bottom: 1px dashed #888;
			font-size: 18px;
			font-family: serif;
		}
		input[type=number]:focus {
			outline: 0;
			border-color: #444;
			background: #fafafa;
		}
		table input[type=number] {
			width: 3em;
		}
		img { display: block }
		form, #images {
			display: inline-block;
			vertical-align: top;
			padding: 32px;
		}
		.eqn {
			text-align: center;
			font-size: 24px;
		}
	</style>
</head>
<body>
	<div id=images>
		<img id=fractal>
	</div>
	<form>
		<p class=eqn>ƒ<sub>c</sub>(<i>z</i>) = <i>z</i><sup>2</sup> + <i>c</i></p>
		<img id=palette>
		<table>
			<tr>
				<td>
					<label for=a><i>a =</i></label>
					<input name=a type=number step=1>
				</td>
				<td>
					<label for=b><i>b =</i></label>
					<input name=b type=number step=1>
				</td>
				<td>
					<label for=c><i>c =</i></label>
					<input name=c type=number step=1>
				</td>
				<td>
					<label for=d><i>d =</i></label>
					<input name=d type=number step=1>
				</td>
			</tr>
			<tr>
				<td>
					<label for=e><i>e =</i></label>
					<input name=e type=number step=1>
				</td>
				<td>
					<label for=f><i>f =</i></label>
					<input name=f type=number step=1>
				</td>
				<td>
					<label for=g><i>g =</i></label>
					<input name=g type=number step=1>
				</td>
				<td>
					<label for=h><i>h =</i></label>
					<input name=h type=number step=1>
				</td>
			</tr>
			<tr>
				<td>
					<label for=i><i>i =</i></label>
					<input name=i type=number value=0 step=1>
				</td>
				<td>
					<label for=j><i>j =</i></label>
					<input name=j type=number value=0 step=1>
				</td>
				<td>
					<label for=k><i>k =</i></label>
					<input name=k type=number value=0 step=1>
				</td>
				<td>
					<label for=l><i>l =</i></label>
					<input name=l type=number value=0 step=1>
				</td>
			</tr>
		</table>
		<hr>
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
		<button id=submit name=submit type=button>render</button>
		<button id=random type=button>pick a random c</button>
	</form>
	<script>
		var vars = {
			"a": -20, "b": 1, "c": -20, "d": 11,
			"e": -20, "f": 5, "g": -20, "h": 15,
			"i": -20, "j": 9, "k": -20, "l": 19,
			//"re": Math.random()*2-1,
			//"im": Math.random()*2-1,
			"re": -0.75,
			"im": 0.14,
			"scale": 0, "width": 512, "height": 512,
			"iterations": 255
		};

		var fractal, palette, inputs;
		var submit, random;

		function radioValue(s) {
			var inputs = document.getElementsByName(s);
			for (var i = 0, input; input = inputs[i]; i++) {
				if (input.checked) {
					return input.value;
				}
			}
		}

		function stick(stickButtons) {
			for (var i = 0, input; input = inputs[i]; i++) {
				input.setAttribute('disabled');
			}
			if (stickButtons) {
				submit.setAttribute('disabled');
				submit.innerText = "rendering...";
				random.setAttribute('disabled');
			}
		}

		function unstick() {
			for (var i = 0, input; input = inputs[i]; i++) {
				input.removeAttribute('disabled');
			}
			submit.removeAttribute('disabled');
			submit.innerText = "render";
			random.removeAttribute("disabled");
		}

		function updatePalette() {
			var q = [];
			"abcdefghijkl".split("").forEach(function(key) {
				q.push(key + "=" + vars[key]);
			})
			var args = q.join("&");

			stick(false);
			palette.addEventListener("load", function() {
				unstick();
				palette.removeEventListener("load");
			});
			palette.setAttribute("src", "/palette.png?" + args);
		}

		function updateFractal(e) {
			var q = [];
			for (var key in vars) {
				q.push(key + "=" + vars[key]);
			}
			q.push('palette=' + radioValue('palette'));
			q.push('coloring=' + radioValue('coloring'));
			var args = q.join("&");

			stick(true);
			var se;
			if (e != null && e.target.selectionEnd) {
				se = e.target.selectionEnd;
			}

			fractal.addEventListener("load", function() {
				unstick();
				if (e != null && e.target.setSelectionRange) {
					e.target.focus();
					e.target.setSelectionRange(se, se);
				}
				fractal.removeEventListener("load");
			});
			fractal.setAttribute("src", "/julia.png?" + args);
		}

		window.addEventListener("DOMContentLoaded", function() {
			for (var key in vars) {
				document.querySelector("[name=" + key + "]").value = vars[key];
			}

			fractal = document.querySelector('#fractal');
			palette = document.querySelector('#palette');
			inputs = document.querySelectorAll('input');
			submit = document.querySelector('#submit');
			random = document.querySelector("#random");

			for (var i = 0, input; input = inputs[i]; i++) {
				input.addEventListener("input", function(e) {
					var name = e.target.getAttribute("name");
					vars[name] = e.target.value;
					// the colors all have 1 length names
					if (name.length === 1) {
						updatePalette();
						if (radioValue("palette") === "gray") {
							return;
						}
					}
					// update immediately unless it's something potentially very expensive
					if (name !== "width" && name !== "height" && name !== "iterations") {
						updateFractal(e);
					}
				});
			}

			updateFractal();
			updatePalette();
			submit.addEventListener("click", updateFractal);
			random.addEventListener("click", function() {
				document.querySelector('[name=re]').value = vars['re'] = Math.random()*2-1;
				document.querySelector('[name=im]').value = vars['im'] = Math.random()*2-1;
				updateFractal();
			});
		});
	</script>
</body>`
