package weather

import "math"

// Pollutants are concentrations in µg/m³ (CO in mg/m³).
type Pollutants struct{ PM25, PM10, O3, NO2, SO2, COmg float64 }

var iaqiSteps = []float64{0, 50, 100, 150, 200, 300, 400, 500}

// Breakpoints from China HJ 633-2012 (PM: 24h; O3/NO2/SO2/CO: 1h values).
var bp = map[string][]float64{
	"PM2.5": {0, 35, 75, 115, 150, 250, 350, 500},
	"PM10":  {0, 50, 150, 250, 350, 420, 500, 600},
	"O3":    {0, 160, 200, 300, 400, 800, 1000, 1200},
	"NO2":   {0, 100, 200, 700, 1200, 2340, 3090, 3840},
	"SO2":   {0, 150, 500, 650, 800, 1600, 2100, 2620},
	"CO":    {0, 5, 10, 35, 60, 90, 120, 150},
}

func iaqi(c float64, b []float64) float64 {
	if c <= 0 {
		return 0
	}
	for i := 1; i < len(b); i++ {
		if c <= b[i] {
			return (iaqiSteps[i]-iaqiSteps[i-1])/(b[i]-b[i-1])*(c-b[i-1]) + iaqiSteps[i-1]
		}
	}
	return 500
}

// ChinaAQI estimates the Chinese AQI (HJ 633-2012) and primary pollutant.
func ChinaAQI(p Pollutants) (int, string) {
	vals := map[string]float64{"PM2.5": p.PM25, "PM10": p.PM10, "O3": p.O3, "NO2": p.NO2, "SO2": p.SO2, "CO": p.COmg}
	best, name := 0.0, ""
	for _, k := range []string{"PM2.5", "PM10", "O3", "NO2", "SO2", "CO"} {
		if v := iaqi(vals[k], bp[k]); v > best {
			best, name = v, k
		}
	}
	aqi := int(math.Ceil(best))
	if aqi <= 50 {
		name = "" // no primary pollutant when AQI <= 50
	}
	return aqi, name
}

// AQILevel returns the Chinese AQI category.
func AQILevel(aqi int) (string, string) {
	switch {
	case aqi <= 50:
		return "优", "Excellent"
	case aqi <= 100:
		return "良", "Good"
	case aqi <= 150:
		return "轻度污染", "Lightly polluted"
	case aqi <= 200:
		return "中度污染", "Moderately polluted"
	case aqi <= 300:
		return "重度污染", "Heavily polluted"
	default:
		return "严重污染", "Severely polluted"
	}
}
