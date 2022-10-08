package main

import (
	"github.com/ghetzel/go-stockutil/colorutil"
)

type ShaderFunc func(step *ShaderStep) (colorutil.Color, bool)

type ShaderStep struct {
	Index             int
	Frame             int
	CurrentColor      colorutil.Color
	TargetColor       colorutil.Color
	LastStepColor     colorutil.Color
	AnimationProgress float64
	Args              []string
}

func (self *ShaderStep) FromColor() (color colorutil.Color) {
	var ranges = self.Ranges()
	var i int = int(float64(len(ranges))*self.AnimationProgress) - 1

	if i >= 0 && i < len(ranges) {
		color, _ = ranges[i].Get(self.Index)
	} else {
		color = self.CurrentColor
	}

	return
}

func (self *ShaderStep) ToColor() (color colorutil.Color) {
	var ranges = self.Ranges()
	var i int = int(float64(len(ranges)) * self.AnimationProgress)

	if i >= 0 && i < len(ranges) {
		color, _ = ranges[i].Get(self.Index)
	} else {
		color = self.TargetColor
	}

	return
}

func (self *ShaderStep) Range() LEDSet {
	if ranges := self.Ranges(); len(ranges) > 0 {
		return ranges[0]
	} else {
		return make(LEDSet)
	}
}

func (self *ShaderStep) RangePair() (LEDSet, LEDSet, bool) {
	if ranges := self.Ranges(); len(ranges) > 1 {
		return ranges[0], ranges[1], true
	} else {
		return nil, nil, false
	}
}

func (self *ShaderStep) Ranges() (ranges []LEDSet) {
	for _, arg := range self.Args {
		ranges = append(ranges, ParseLEDRange(arg))
	}
	return
}

func Fill(step *ShaderStep) (colorutil.Color, bool) {
	return step.ToColor(), true
}

func Fade(step *ShaderStep) (colorutil.Color, bool) {
	if c, err := colorutil.MixN(step.ToColor(), step.FromColor(), step.AnimationProgress); err == nil {
		return c, true
	} else {
		return step.FromColor(), true
	}
}

// switch fx {
// case ``, `fill`:

// 	proto.WriteBytes(conn, timeout, payload...)
// case `colortrain`:
// 	var i int = 0
// 	// var oc colorutil.Color
// 	var cc colorutil.Color = colorutil.MustParse(`#ff0000`)

// 	for {
// 		if !ledset.Has(i) {
// 			continue
// 		}

// 		var r, g, b, _ uint8 = cc.RGBA255()
// 		proto.WriteTo(conn, timeout, i, r, g, b)
// 		time.Sleep(sleep)

// 		if i > 1 {
// 			// oc = cc
// 			cc, _ = colorutil.AdjustHue(cc, 1)
// 			// cc, _ = colorutil.Mix(cc, oc)
// 		}

// 		i = (i + 1) % num_leds
// 	}

// case `sequence`:
// 	for {
// 		for _, phase := range c.Args() {
// 			var ledset wledLedSet = parse_wledRange(phase)
// 			var payload = make([]byte, num_leds*4)

// 			for i := 0; i < num_leds; i++ {
// 				var r, g, b, _ uint8
// 				var c, ok = ledset.Get(i)

// 				if ok {
// 					r, g, b, _ = c.RGBA255()
// 				} else {
// 					continue
// 				}

// 				payload[0+(i*4)] = byte(i)
// 				payload[1+(i*4)] = r
// 				payload[2+(i*4)] = g
// 				payload[3+(i*4)] = b
// 			}

// 			proto.WriteBytes(conn, timeout, payload...)
// 			time.Sleep(sleep)
// 		}
// 	}
// }
