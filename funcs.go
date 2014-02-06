package main

import (
	"image/color"
	"math"
	"math/cmplx"
)

var (
	coloringFuncs = map[string]ColoringFunc{
		"distance": distance,
		"escape":   escapeTime,
	}
	paletteFuncs = map[string]PaletteFunc{
		"color": palette,
		"gray":  gray,
	}
	iterFuncs = map[string]IterFunc{
		"quadratic": IterFunc{
			f: func(z, c complex128) complex128 {
				return z*z + c
			},
			fp: func(z, c complex128) complex128 {
				return 2 * z
			},
		},
		"sierpinski": IterFunc{
			f: func(z, c complex128) complex128 {
				return z*z + c/z
			},
			fp: func(z, c complex128) complex128 {
				return 2*z + c/(z*z)
			},
		},
		"quadratic-abs": IterFunc{
			f: func(z, c complex128) complex128 {
				z = complex(math.Abs(real(z)), math.Abs(imag(z)))
				return z*z + c
			},
			fp: func(z, c complex128) complex128 {
				return (2 * complex(math.Abs(real(z)), math.Abs(imag(z))) * complex(real(z)*math.Abs(imag(z)), imag(z)*math.Abs(real(z)))) / complex(math.Abs(imag(z))*math.Abs(real(z)), 0)
			},
		},
	}
)

type IterFunc struct {
	f, fp func(z, c complex128) complex128
}

type ColoringFunc func(o *Opts, z, c complex128) float64

func distance(o *Opts, z, c complex128) float64 {
	zp := 1 + 0i // z' = f'(z_n)
	zn := z      // next z_n value

derivative:
	for j := 0; j < o.maxIters; j++ {
		zn = o.iterFunc.f(z, c)
		zp *= o.iterFunc.fp(z, c)
		z = zn
		if cmplx.Abs(zp) > 1.0e60 {
			break derivative
		}
	}

	za := cmplx.Abs(z)
	dist := za * (math.Log(za) / cmplx.Abs(zp))
	return -1/(3*(dist/o.distMax)+1) + 1
}

func escapeTime(o *Opts, z, c complex128) float64 {
	i := 0
	for i < o.maxIters {
		z = o.iterFunc.f(z, c)
		if escape(z) {
			break
		}
		i++
	}

	p := 1 - float64(i)/float64(o.maxIters)

	return p
}

type PaletteFunc func(o *Opts, x float64) color.Color

func gray(o *Opts, x float64) color.Color {
	return color.Gray{uint8(255 * x)}
}

func palette(o *Opts, x float64) color.Color {
	return color.RGBA{
		uint8(255 * R(o.args, x)),
		uint8(255 * G(o.args, x)),
		uint8(255 * B(o.args, x)),
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
