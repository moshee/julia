var vars = {
	"a": -20, "b": 1, "c": -20, "d": 11,
	"e": -20, "f": 5, "g": -20, "h": 15,
	"i": -20, "j": 9, "k": -20, "l": 19,
	"re": -0.505, "im": 0.523,
	"rePos": 0, "imPos": 0,
	"scale": 0, "width": 512, "height": 512,
	"iterations": 255
};

var fractal, palette, inputs;
var submit, random, randomColors;
var center;
var stuck = false;

function $(sel, base) {
	if (typeof base === 'undefined') {
		base = document;
	}
	return base.querySelector(sel);
}

function setVar(key, val) {
	$('[name=' + key + ']').value = vars[key] = val;
}

function radioValue(s) {
	var inputs = document.getElementsByName(s);
	for (var i = 0, input; input = inputs[i]; i++) {
		if (input.checked) {
			return input.value;
		}
	}
}

function stick() {
	stuck = true;
	for (var i = 0, input; input = inputs[i]; i++) {
		input.disabled = true;
	}
	for (var i = 0, button; button = document.querySelectorAll("button")[i]; i++) {
		button.disabled = true;
		if ('disabledValue' in button.dataset) {
			var old = button.innerText;
			button.innerText = button.dataset.disabledValue;
			button.dataset.disabledValue = old;
		}
	}
}

function unstick() {
	for (var i = 0, input; input = inputs[i]; i++) {
		if (input.dataset.leaveDisabled) continue;
		input.disabled = false;
	}
	for (var i = 0, button; button = document.querySelectorAll("button")[i]; i++) {
		if (button.dataset.leaveDisabled) continue;
		button.disabled = false;
		if ('disabledValue' in button.dataset) {
			var old = button.innerText;
			button.innerText = button.dataset.disabledValue;
			button.dataset.disabledValue = old;
		}
	}
	stuck = false;
}

function updatePalette() {
	var q = [];
	"abcdefghijkl".split("").forEach(function(key) {
		q.push(key + "=" + vars[key]);
	})
	var args = q.join("&");

	palette.addEventListener("load", function() {
		palette.removeEventListener("load");
	});
	palette.setAttribute("src", "/palette.png?" + args);
}

function updateFractal(e) {
	var q = [];
	for (var key in vars) {
		q.push(key + "=" + vars[key]);
	}
	var radios = [];
	for (var i = 0, radio; radio = document.querySelectorAll("input[type=radio]")[i]; i++) {
		var name = radio.getAttribute("name");
		if (radios.indexOf(name) == -1) {
			radios.push(name);
		}
	}
	q = q.concat(radios.map(function(name) {
		return name + '=' + radioValue(name);
	}));
	q.push('center=' + center.checked);
	var args = q.join("&");

	stick();
	var se;
	try {
		se = e.target.selectionEnd;
	} catch (e) {
		se = null;
	}

	fractal.addEventListener("load", function() {
		unstick();
		if (se != null && e.target.setSelectionRange) {
			e.target.focus();
			e.target.setSelectionRange(se, se);
		}
		fractal.removeEventListener("load");
	});

	var type = radioValue("fractal-type");
	var format = radioValue("format");
	var endpoint = "/" + type + "." + format;

	fractal.setAttribute("src", endpoint + "?" + args);
}

function radioListener(sel, func, shouldUpdate) {
	for (var i = 0, radio; radio = document.querySelectorAll(sel)[i]; i++) {
		if (shouldUpdate) {
			radio.addEventListener("change", function(e) {
				func(e);
				updateFractal(e);
			});
		} else {
			radio.addEventListener("change", func);
		}
	}
	func();
}

window.addEventListener("DOMContentLoaded", function() {
	for (var key in vars) {
		$("[name=" + key + "]").value = vars[key];
	}

	fractal      = $('#fractal');
	palette      = $('#palette');
	inputs       = document.querySelectorAll('input');
	submit       = $('#submit');
	random       = $("#random");
	randomColors = $('#random-colors');
	center       = $("#center");

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
		input.addEventListener("keydown", function(e) {
			if (e.target.disabled) {
				return;
			}
			var delta = +e.target.getAttribute("step") || 0.01;

			switch (e.keyCode) {
				case 38: // up
				case 40: // down
				if (e.shiftKey) {
					delta *= 10;
				} else if (e.metaKey || e.altKey) {
					delta /= 10;
				}
				break;
				default:
				return true;
			}
			e.preventDefault();

			var deltaParts = delta.toString().split('.');
			var deltaPrec = deltaParts.length >= 2 ? deltaParts[1].length : 0;

			if (e.keyCode == 40) {
				delta = -delta;
			}

			// round bad float values
			/*
			var oldParts = e.target.value.split('.');
			var oldPrec = oldParts.length >= 2 ? oldParts[1].length : 0;

			var parts = newValue.toString().split('.');
			var prec = parts.length >= 2 ? parts[1].length : 0;

			// check if the floating point precision added too many decimals
			if (prec - oldPrec > deltaPrec) {
				prec--;
			}

			e.target.value = +newValue.toFixed(prec);
			*/
			var newValue = +e.target.value + delta;
			e.target.value = +newValue.toFixed(deltaPrec);
			e.target.dispatchEvent(new Event('input'));
		}, true);
	}

	updateFractal();
	updatePalette();
	submit.addEventListener("click", updateFractal);
	center.addEventListener("click", updateFractal);

	radioListener("input[name=fractal-type]", function() {
		var inputs = document.querySelectorAll("#parameter input, #parameter button");
		if (radioValue("fractal-type") === "mandelbrot") {
			for (var j = 0, input; input = inputs[j]; j++) {
				input.disabled = true;
				input.dataset.leaveDisabled = true;
			}
		} else {
			for (var j = 0, input; input = inputs[j]; j++) {
				input.disabled = false;
				delete input.dataset.leaveDisabled;
			}
		}
	}, true);

	radioListener("input[name=func]", function() { }, true);

	var rePos = $("#rePos");
	var imPos = $("#imPos");
	center.addEventListener("click", function(e) {
		if (e.target.checked) {
			rePos.dataset.leaveDisabled = true;
			rePos.dataset.old = rePos.value;
			rePos.value = vars.re;
			imPos.dataset.leaveDisabled = true;
			imPos.dataset.old = imPos.value;
			imPos.value = vars.im;
		} else {
			rePos.disabled = false;
			delete rePos.dataset.leaveDisabled;
			rePos.value = rePos.dataset.old;
			imPos.disabled = false;
			delete imPos.dataset.leaveDisabled;
			imPos.value = imPos.dataset.old;
		}
	});
	random.addEventListener("click", function() {
		setVar('re', Math.random()*2 - 1);
		setVar('im', Math.random()*2 - 1);
		updateFractal();
	});
	randomColors.addEventListener('click', function() {
		var b = Math.floor(Math.random()*15);
		var f = Math.floor(Math.random()*15);
		var j = Math.floor(Math.random()*15);
		setVar('a', Math.floor(-Math.random()*10 - 15));
		setVar('b', b);
		setVar('c', Math.floor(-Math.random()*10 - 15));
		setVar('d', b + Math.ceil(Math.random()*15));
		setVar('e', Math.floor(-Math.random()*10 - 15));
		setVar('f', f);
		setVar('g', Math.floor(-Math.random()*10 - 15));
		setVar('h', f + Math.ceil(Math.random()*15));
		setVar('i', Math.floor(-Math.random()*10 - 15));
		setVar('j', j);
		setVar('k', Math.floor(-Math.random()*10 - 15));
		setVar('l', j + Math.ceil(Math.random()*15));
		updatePalette();
		updateFractal();
	});
	$("#knob-panel").addEventListener("keydown", function(e) {
		if (!stuck && e.keyCode == 13) { // enter
			updatePalette();
			updateFractal();
		}
	});
	$("#fractal").addEventListener("click", function(e) {
		// find the relative % location of the click, where the exact middle is 0, 0
		var xp =     2*e.offsetX / e.target.offsetWidth - 1;
		var yp = 1 - 2*e.offsetY / e.target.offsetHeight;
		var scale = Math.pow(2, -vars.scale / 2);
		// find the width of the viewport in units
		var xDelta = yDelta = scale;
		if (vars.width >= vars.height) {
			xDelta *= vars.width/vars.height;
		} else {
			yDelta *= vars.height/vars.width;
		}
		setVar("rePos", +vars.rePos + xp * xDelta);
		setVar("imPos", +vars.imPos + yp * yDelta);
		center.checked = false;
		delete rePos.dataset.leaveDisabled;
		delete imPos.dataset.leaveDisabled;
		updateFractal();
	});
	$("#reset-to-origin").addEventListener("click", function() {
		if (vars.rePos == 0 && vars.imPos == 0) return;
		setVar("rePos", 0);
		setVar("imPos", 0);
		updateFractal();
	});
	$("#zoom0").addEventListener("click", function() {
		if (vars.scale == 0) return;
		setVar("scale", 0);
		updateFractal();
	});
});
