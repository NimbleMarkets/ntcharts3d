// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"math"
	"strconv"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

func numericTicks(minimum, maximum float32, count int, format func(float64) string) []AxisTick {
	lo, hi := float64(minimum), float64(maximum)
	if !math3d.IsFinite(lo) || !math3d.IsFinite(hi) || lo > hi {
		return nil
	}
	if count == 0 {
		count = 5
	}
	count = max(2, min(64, count))
	step := float64(0)
	if hi > lo {
		raw := (hi - lo) / float64(count-1)
		power := math.Pow(10, math.Floor(math.Log10(raw)))
		fraction := raw / power
		unit := float64(10)
		for _, candidate := range []float64{1, 2, 5, 10} {
			if fraction <= candidate*(1+1e-6) {
				unit = candidate
				break
			}
		}
		step = unit * power
	}
	label := func(v float64) string {
		if format != nil {
			return format(v)
		}
		if v == 0 {
			return "0"
		}
		// Enough precision to distinguish ticks even far from zero.
		if math.Abs(v) >= 1e6 || math.Abs(v) < .0001 {
			return strconv.FormatFloat(v, 'g', -1, 32)
		}
		precision := 0
		if step > 0 {
			precision = max(0, int(-math.Floor(math.Log10(step))))
		} else {
			return strconv.FormatFloat(v, 'g', -1, 32)
		}
		return strconv.FormatFloat(v, 'f', precision, 64)
	}
	if lo == hi {
		return []AxisTick{{minimum, label(lo)}}
	}
	start := math.Ceil(lo/step-1e-10) * step
	var ticks []AxisTick
	for i := 0; i < 64; i++ {
		v := start + float64(i)*step
		if v > hi+step*1e-8 {
			break
		}
		if math.Abs(v) < step*1e-8 {
			v = 0
		}
		value := float32(v)
		if value < minimum || value > maximum {
			continue
		}
		if len(ticks) > 0 && ticks[len(ticks)-1].Value == value {
			continue
		}
		ticks = append(ticks, AxisTick{value, label(v)})
	}
	if len(ticks) == 0 {
		return []AxisTick{{minimum, label(lo)}}
	}
	return ticks
}
