package main

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/colorutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultTransitionShaderDuration = 3000 * time.Millisecond
var DefaultColor = colorutil.MustParse(`#00000000`)
var ColorBlack = colorutil.MustParse(`#000000FF`)

type LEDSet map[int]colorutil.Color

func (self LEDSet) Has(i int) bool {
	if len(self) == 0 || i == math.MaxInt {
		return true
	}

	if _, ok := self[i]; ok {
		return true
	} else {
		return false
	}
}

func (self LEDSet) Get(i int) (colorutil.Color, bool) {
	if c, ok := self[i]; ok && !c.IsZero() {
		return c, true
	} else if c, ok := self[math.MaxInt]; ok && !c.IsZero() {
		return c, true
	} else {
		return colorutil.MustParse(`black`), false
	}
}

func ParseLEDRange(rangespec string, frames int) (leds LEDSet) {
	leds = make(LEDSet)
	rangespec = strings.TrimSpace(rangespec)
	var subranges = strings.Split(rangespec, `,`)

	for _, subrange := range subranges {
		var color colorutil.Color
		var idxspec, colorspec = stringutil.SplitPairTrimSpace(subrange, `@`)
		var index, rpt = stringutil.SplitPairTrimSpace(idxspec, `/`)

		if colorspec == `` {
			colorspec = subrange
			index = `*`
		}

		switch colorspec {
		case ``:
			continue
		case `-`:
			color = colorutil.MustParse(`rgba(0,0,0,1)`)
		default:
			color = colorutil.MustParse(colorspec)
		}

		if index == `*` {
			leds[math.MaxInt] = color
			return
		} else if a, b := stringutil.SplitPairTrimSpace(index, `:`); a != `` {
			var ai int = typeutil.NInt(a)

			if b != `` {
				var bi int = typeutil.NInt(b)
				var step = typeutil.NInt(rpt)

				switch rpt {
				case `*`:
					step = frames
				}

				if step == 0 {
					step = 1
				}

				for i := ai; i < bi; i += step {
					leds[i] = color
				}
			} else {
				leds[ai] = color
			}
		}
	}

	return
}

type Protocol byte

const (
	wled_WARLS  Protocol = 0x1
	wled_DRGB            = 0x2
	wled_DRGBW           = 0x3
	wled_DNRGB           = 0x4
	wled_NOTIFY          = 0x0
)

func (self LEDSet) Bytes(proto Protocol, timeout uint8, skipZero bool) []byte {
	var payload = make([]byte, len(self)*4)

	for i := 0; i < len(self); i++ {
		if c, ok := self[i]; ok && (!skipZero || !c.IsZero()) {
			var r, g, b, _ uint8 = c.RGBA255()
			payload[0+(i*4)] = byte(i)
			payload[1+(i*4)] = r
			payload[2+(i*4)] = g
			payload[3+(i*4)] = b
		}
	}

	return append([]byte{
		byte(proto),
		byte(timeout),
	}, payload...)
}

type Display struct {
	FrontBuffer              LEDSet
	BackBuffer               LEDSet
	FrameInterval            time.Duration
	AutoclearTimeout         uint8
	TransitionShader         ShaderFunc
	TransitionShaderDuration time.Duration
	TransitionArgs           []string
	DefaultColor             colorutil.Color
	Brightness               uint8
	ClearFirst               bool
	OffsetCounter            int
	ledcount                 int
	wledDest                 io.Writer
	proto                    Protocol
}

func NewDisplay(w io.Writer, ledcount int) *Display {
	var sset = &Display{
		FrontBuffer:              make(LEDSet),
		BackBuffer:               make(LEDSet),
		FrameInterval:            16 * time.Millisecond,
		ClearFirst:               true,
		AutoclearTimeout:         255,
		TransitionShader:         nil,
		TransitionShaderDuration: DefaultTransitionShaderDuration,
		TransitionArgs:           make([]string, 0),
		DefaultColor:             DefaultColor,
		Brightness:               255,
		ledcount:                 ledcount,
		wledDest:                 w,
		proto:                    wled_WARLS,
	}

	sset.init()
	return sset
}

func (self *Display) SetTransitionEffect(effect string, args ...string) error {
	effect = strings.ToLower(effect)
	self.TransitionArgs = args

	switch effect {
	case ``, `fill`:
		self.TransitionShader = Fill
		self.TransitionShaderDuration = 0
	case `fade`:
		self.TransitionShader = Fade
	default:
		return fmt.Errorf("unknown effect %q", effect)
	}

	return nil
}

func (self *Display) init() {
	for i := 0; i < self.ledcount; i++ {
		self.FrontBuffer[i] = self.DefaultColor
		self.BackBuffer[i] = self.DefaultColor
	}
}

func (self *Display) flushBuffer(buffer LEDSet) error {
	var buf = buffer.Bytes(
		self.proto,
		self.AutoclearTimeout,
		!self.ClearFirst,
	)

	// log.Debugf("flush: %x", buf)
	var _, err = self.wledDest.Write(buf)
	return err
}

func (self *Display) Flush() error {
	var fi time.Duration = self.FrameInterval
	var td time.Duration = self.TransitionShaderDuration

	log.Debugf("flush requested: fn=%p transition=%v", self.TransitionShader, td)
	self.init()

	if tfn := self.TransitionShader; tfn != nil {
		var progressStep float64 = float64(fi) / float64(td)
		var frame int = 1
		var progress float64 = 0
		var start = time.Now()

		log.Debugf("anim start: step=%f duration=%v", progressStep, td)

		for progress <= 1.0 && !math.IsInf(progress, 0) {
			var transitionBuffer = make(LEDSet)

			for i := 0; i < self.ledcount; i++ {
				transitionBuffer[i] = self.BackBuffer[i]
			}

			var continueAnimation bool

			for i := 0; i < self.ledcount; i++ {
				var oi int = (i + self.OffsetCounter) % len(self.FrontBuffer)
				var bc = self.BackBuffer[i]
				var fc = self.FrontBuffer[oi]
				var lc = bc
				var tc = lc

				if i > 0 {
					lc = transitionBuffer[i-1]
				}

				tc, continueAnimation = tfn(&ShaderStep{
					Index:             i,
					Frame:             frame,
					AnimationProgress: progress,
					CurrentColor:      fc,
					LastStepColor:     lc,
					Args:              self.TransitionArgs,
				})

				if self.ClearFirst || !tc.Equals(`black`) {
					transitionBuffer[i] = tc
					self.BackBuffer[i] = tc
				} else {
					transitionBuffer[i] = colorutil.Color{}
					self.BackBuffer[i] = colorutil.Color{}
				}
			}

			// self.flushBuffer(transitionBuffer)

			if !continueAnimation {
				break
			}

			if fi > 0 {
				time.Sleep(fi)
			}

			frame += 1
			progress += progressStep
		}

		log.Debugf("anim done: frames=%d progress=%v took=%v", frame, progress, time.Since(start))
	}

	for i := 0; i < len(self.FrontBuffer); i++ {
		var oi int = (i + self.OffsetCounter) % len(self.FrontBuffer)
		self.FrontBuffer[oi] = self.BackBuffer[i]
	}

	return self.flushBuffer(self.FrontBuffer)
}
