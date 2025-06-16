package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"strings"
	"unsafe"

	"math"
	"sync/atomic"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const bufferSize = 64

const sinMinVal = 150
const sinMaxVal = 14000
const sinRatio = sinMaxVal / sinMinVal

var (
	whiteNoiseOn  int32 = 0
	whiteNoisePan       = 0.5
	whiteNoiseVol       = 0.5

	pinkNoiseOn  int32 = 0
	pinkNoisePan       = 0.5
	pinkNoiseVol       = 0.5

	sineToneOn         int32 = 0
	sineTonePan              = 0.5
	sineToneVol              = 0.5
	sineToneFreqCoarse       = 0.5
	sineToneFreqFine         = 0.5
	sineToneOff              = 0.0

	sineTime = 0.0
)

type AGC struct {
	targetLevel float64 // Desired output level, e.g. 0.3
	currentGain float64 // Gain applied to incoming signal
	peak        float64 // Tracks recent signal peak
	attack      float64 // How quickly to respond to louder signals
	decay       float64 // How quickly to return to normal after peaks
}

func NewAGC(target float64) *AGC {
	return &AGC{
		targetLevel: target,
		currentGain: 1.0,
		attack:      0.01,   // fast response to loudness
		decay:       0.0005, // slow recovery
	}
}

func (a *AGC) ProcessSample(sample float64) float64 {
	absSample := math.Abs(sample)

	// Update peak using attack/decay envelope follower
	if absSample > a.peak {
		a.peak = a.peak*(1-a.attack) + absSample*a.attack
	} else {
		a.peak = a.peak*(1-a.decay) + absSample*a.decay
	}

	// Avoid divide-by-zero
	if a.peak > 0.0001 {
		a.currentGain = a.targetLevel / a.peak
	}

	return sample * a.currentGain
}

var whiteAGC = NewAGC(0.5)
var pinkAGC = NewAGC(0.66)
var sineAGC = NewAGC(0.5)

func main() {

	portaudio.Initialize()
	defer portaudio.Terminate()

	myApp := app.New()
	myWindow := myApp.NewWindow("Tinnitus Explorer")

	toggleSpacer := canvas.NewText(strings.Repeat("W", 10), color.Transparent)
	labelSpacer := canvas.NewText(strings.Repeat("W", 10), color.Transparent)

	quitButton := widget.NewButton("Quit", func() {
		myApp.Quit()
	})

	whiteNoiseToggle := container.NewStack(toggleSpacer,
		widget.NewCheck("White Noise", func(checked bool) {
			atomic.StoreInt32(&whiteNoiseOn, 1-atomic.LoadInt32(&whiteNoiseOn))
		}))

	pinkNoiseToggle := container.NewStack(toggleSpacer,
		widget.NewCheck("Pink Noise", func(checked bool) {
			atomic.StoreInt32(&pinkNoiseOn, 1-atomic.LoadInt32(&pinkNoiseOn))
		}))

	sineToneToggle := container.NewStack(toggleSpacer,
		widget.NewCheck("Sine Tone", func(checked bool) {
			atomic.StoreInt32(&sineToneOn, 1-atomic.LoadInt32(&sineToneOn))
		}))

	makeSlider := func(onChanged func(float64)) *widget.Slider {
		newSlider := widget.NewSlider(0, 100)
		newSlider.Step = 1
		newSlider.OnChanged = onChanged
		newSlider.SetValue(50)
		return newSlider
	}

	whiteNoisePanLabel := widget.NewLabel("")
	whiteNoisePanSlider := makeSlider(func(value float64) {
		update := (float64(value) / 50.0) - 1.0
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&whiteNoisePan)), math.Float64bits(update))
		whiteNoisePanLabel.Text = fmt.Sprintf("Pan: %+01.2f", update)
		whiteNoisePanLabel.Refresh()
	})

	whiteNoiseVolLabel := widget.NewLabel("")
	whiteNoiseVolSlider := makeSlider(func(value float64) {
		update := float64(value / 100)
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&whiteNoiseVol)), math.Float64bits(update))
		whiteNoiseVolLabel.Text = fmt.Sprintf("Vol: %+01.2f", update)
		whiteNoiseVolLabel.Refresh()
	})

	pinkNoisePanLabel := widget.NewLabel("")
	pinkNoisePanSlider := makeSlider(func(value float64) {
		update := (float64(value) / 50.0) - 1.0
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&pinkNoisePan)), math.Float64bits(update))
		pinkNoisePanLabel.Text = fmt.Sprintf("Pan: %+01.2f", update)
		pinkNoisePanLabel.Refresh()
	})

	pinkNoiseVolLabel := widget.NewLabel("")
	pinkNoiseVolSlider := makeSlider(func(value float64) {
		update := float64(value / 100)
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&pinkNoiseVol)), math.Float64bits(update))
		pinkNoiseVolLabel.Text = fmt.Sprintf("Vol: %+01.2f", update)
		pinkNoiseVolLabel.Refresh()
	})

	sineTonePanLabel := widget.NewLabel("")
	sineTonePanSlider := makeSlider(func(value float64) {
		update := (float64(value) / 50.0) - 1.0
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&sineTonePan)), math.Float64bits(update))
		sineTonePanLabel.Text = fmt.Sprintf("Pan: %+01.2f", update)
		sineTonePanLabel.Refresh()
	})

	sineToneVolLabel := widget.NewLabel("")
	sineToneVolSlider := makeSlider(func(value float64) {
		update := float64(value / 100)
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&sineToneVol)), math.Float64bits(update))
		sineToneVolLabel.Text = fmt.Sprintf("Vol: %+01.2f", update)
		sineToneVolLabel.Refresh()
	})

	sineToneOffLabel := widget.NewLabel("")
	sineToneOffSlider := makeSlider(func(value float64) {
		update := value - 50
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&sineToneOff)), math.Float64bits(update))
		sineToneOffLabel.Text = fmt.Sprintf("Off: %%%+01.2f", update)
		sineToneOffLabel.Refresh()
	})

	sineToneFreqCoarseLabel := widget.NewLabel("")
	sineToneFreqSliderCoarse := makeSlider(func(value float64) {
		update := sinMinVal * math.Pow(sinRatio, value/100.0)
		atomic.StoreUint64((*uint64)(unsafe.Pointer(&sineToneFreqCoarse)), math.Float64bits(update))
		sineToneFreqCoarseLabel.Text = fmt.Sprintf("Hertz: %0.0f", update)
		sineToneFreqCoarseLabel.Refresh()
	})

	sineToneFreqFineLabel := widget.NewLabel("")
	sineToneFreqSliderFine := makeSlider(func(value float64) {

		s := sineToneFreqSliderCoarse.Value

		var a, b, c float64
		if s == 0 {
			a = sinMinVal * math.Pow(sinRatio, 0/100.0)
			b = sinMinVal * math.Pow(sinRatio, 1/100.0)
			c = sinMinVal * math.Pow(sinRatio, 2/100.0)
		} else if s == 100 {
			a = sinMinVal * math.Pow(sinRatio, 98/100.0)
			b = sinMinVal * math.Pow(sinRatio, 99/100.0)
			c = sinMinVal * math.Pow(sinRatio, 100/100.0)
		} else {
			a = sinMinVal * math.Pow(sinRatio, (s-1)/100.0)
			b = sinMinVal * math.Pow(sinRatio, s/100.0)
			c = sinMinVal * math.Pow(sinRatio, (s+1)/100.0)
		}

		var update float64

		// remember, it is as if 0-49 is a 0-100 slider, and 51-100 is also a 0-100 slider
		if value < 50 {
			update -= ((100 - (value * 2)) / 100.0) * (b - a)
		} else if value > 50 {
			update += (((value - 50) * 2) / 100.0) * (c - b)
		} else {
			update = 0
		}

		atomic.StoreUint64((*uint64)(unsafe.Pointer(&sineToneFreqFine)), math.Float64bits(update))
		sineToneFreqFineLabel.Text = fmt.Sprintf("Hertz: %0.0f", sineToneFreqCoarse+update)
		sineToneFreqFineLabel.Refresh()
	})

	whiteNoiseContent := widget.NewCard("White Noise Controls", "",
		container.NewBorder(nil, nil, whiteNoiseToggle, nil,
			container.New(layout.NewFormLayout(),
				container.NewStack(labelSpacer, whiteNoisePanLabel), whiteNoisePanSlider,
				whiteNoiseVolLabel, whiteNoiseVolSlider)))

	pinkNoiseContent := widget.NewCard("Pink Noise Controls", "",
		container.NewBorder(nil, nil, pinkNoiseToggle, nil,
			container.New(layout.NewFormLayout(),
				container.NewStack(labelSpacer, pinkNoisePanLabel), pinkNoisePanSlider,
				pinkNoiseVolLabel, pinkNoiseVolSlider)))

	sineToneContent := widget.NewCard("Sine Tone Controls", "", container.NewBorder(nil, nil, sineToneToggle, nil,
		container.New(layout.NewFormLayout(),
			container.NewStack(labelSpacer, sineTonePanLabel), sineTonePanSlider,
			sineToneVolLabel, sineToneVolSlider,
			sineToneOffLabel, sineToneOffSlider,
			sineToneFreqCoarseLabel, sineToneFreqSliderCoarse,
			sineToneFreqFineLabel, sineToneFreqSliderFine,
		)))

	myWindow.SetContent(container.NewVBox(whiteNoiseContent, pinkNoiseContent, sineToneContent,
		quitButton))

	stream, err := portaudio.OpenDefaultStream(0, 2, sampleRate, bufferSize, audioCallbackStereo)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	stream.Start()
	defer stream.Stop()

	myWindow.ShowAndRun()

}

func audioCallbackStereo(out []float32) {
	for i := 0; i < len(out); i += 2 {

		type oneOutput struct{ left, right float64 }
		var outputSlice []oneOutput

		if atomic.LoadInt32(&whiteNoiseOn) == 1 {
			noise := (rand.Float64()*2 - 1)
			noise = whiteAGC.ProcessSample(noise)

			l, r := circularPan(noise, math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&whiteNoisePan)))))
			l = l * whiteNoiseVol
			r = r * whiteNoiseVol

			outputSlice = append(outputSlice, oneOutput{l, r})
		}

		if atomic.LoadInt32(&pinkNoiseOn) == 1 {
			noise := pinkNoise()
			noise = pinkAGC.ProcessSample(noise)

			l, r := circularPan(noise, math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&pinkNoisePan)))))
			l = l * pinkNoiseVol
			r = r * pinkNoiseVol

			outputSlice = append(outputSlice, oneOutput{l, r})
		}

		if atomic.LoadInt32(&sineToneOn) == 1 {
			sample := math.Sin(sineTime + (2 * math.Pi * (sineToneOff / 100)))
			sample = sineAGC.ProcessSample(sample)

			sineTime += 2 * math.Pi * (sineToneFreqCoarse + sineToneFreqFine) / sampleRate
			if sineTime > 2*math.Pi {
				sineTime -= 2 * math.Pi
			}

			l, r := circularPan(sample, math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&sineTonePan)))))
			l = l * sineToneVol
			r = r * sineToneVol

			outputSlice = append(outputSlice, oneOutput{l, r})
		}

		out[i] = 0
		out[i+1] = 0

		portion := 1 / float64(len(outputSlice))
		for _, j := range outputSlice {
			out[i] += float32(j.left * portion)
			out[i+1] += float32(j.right * portion)
		}

	}
}

/*
func linearPan(value, position float64) (left, right float64) {
	left = value * (1 - position)
	right = value * position
	return
}
*/

func circularPan(value, position float64) (left, right float64) {
	angle := (position + 1) * (math.Pi / 4)
	left = value * math.Cos(angle)
	right = value * math.Sin(angle)
	return
}

// from https://www.firstpr.com.au/dsp/pink-noise/
var pinkNoiseGen = pinkNoiseGenerator{}

type pinkNoiseGenerator struct {
	b0, b1, b2, b3, b4, b5, b6 float64
}

func pinkNoise() float64 {
	white := rand.Float64()*2 - 1
	p := &pinkNoiseGen
	p.b0 = 0.99886*p.b0 + white*0.0555179
	p.b1 = 0.99332*p.b1 + white*0.0750759
	p.b2 = 0.96900*p.b2 + white*0.1538520
	p.b3 = 0.86650*p.b3 + white*0.3104856
	p.b4 = 0.55000*p.b4 + white*0.5329522
	p.b5 = -0.7616*p.b5 - white*0.0168980
	p.b6 = white * 0.115926
	return (p.b0 + p.b1 + p.b2 + p.b3 + p.b4 + p.b5 + p.b6 + white*0.5362)
}
