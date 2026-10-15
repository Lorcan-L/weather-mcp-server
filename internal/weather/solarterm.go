package weather

import (
	"math"
	"time"
)

// SolarTerm is one of the 24 solar terms.
type SolarTerm struct {
	NameZH    string    `json:"name_zh"`
	NameEN    string    `json:"name_en"`
	Longitude int       `json:"sun_longitude_deg"`
	Time      time.Time `json:"time"`
}

// Ordered by solar longitude starting at 0° (春分).
var termNames = [24][2]string{
	{"春分", "Spring Equinox"}, {"清明", "Pure Brightness"}, {"谷雨", "Grain Rain"}, {"立夏", "Start of Summer"},
	{"小满", "Grain Buds"}, {"芒种", "Grain in Ear"}, {"夏至", "Summer Solstice"}, {"小暑", "Minor Heat"},
	{"大暑", "Major Heat"}, {"立秋", "Start of Autumn"}, {"处暑", "End of Heat"}, {"白露", "White Dew"},
	{"秋分", "Autumn Equinox"}, {"寒露", "Cold Dew"}, {"霜降", "Frost's Descent"}, {"立冬", "Start of Winter"},
	{"小雪", "Minor Snow"}, {"大雪", "Major Snow"}, {"冬至", "Winter Solstice"}, {"小寒", "Minor Cold"},
	{"大寒", "Major Cold"}, {"立春", "Start of Spring"}, {"雨水", "Rain Water"}, {"惊蛰", "Awakening of Insects"},
}

// China Standard Time.
var CST = time.FixedZone("CST", 8*3600)

func julianDay(t time.Time) float64 {
	return float64(t.UTC().UnixNano())/86400e9 + 2440587.5
}

func fromJulian(jd float64) time.Time {
	return time.Unix(0, int64((jd-2440587.5)*86400e9)).UTC()
}

// sunLongitude returns apparent geocentric solar longitude in degrees
// (Meeus, Astronomical Algorithms ch. 25, low accuracy ~0.01°).
func sunLongitude(jd float64) float64 {
	// Approximate TT-UT (ΔT ≈ 69s).
	T := (jd + 69.0/86400 - 2451545.0) / 36525
	L0 := 280.46646 + 36000.76983*T + 0.0003032*T*T
	M := (357.52911 + 35999.05029*T - 0.0001537*T*T) * math.Pi / 180
	C := (1.914602-0.004817*T-0.000014*T*T)*math.Sin(M) + (0.019993-0.000101*T)*math.Sin(2*M) + 0.000289*math.Sin(3*M)
	omega := (125.04 - 1934.136*T) * math.Pi / 180
	lon := L0 + C - 0.00569 - 0.00478*math.Sin(omega)
	return math.Mod(math.Mod(lon, 360)+360, 360)
}

func angleDiff(a, b float64) float64 {
	d := math.Mod(a-b+540, 360) - 180
	return d
}

// termTime finds the moment the sun reaches longitude target near jd guess.
func termTime(target float64, guess float64) float64 {
	jd := guess
	for i := 0; i < 50; i++ {
		d := angleDiff(target, sunLongitude(jd))
		jd += d / 360 * 365.2422
		if math.Abs(d) < 1e-7 {
			break
		}
	}
	return jd
}

// SolarTermsOfYear returns the 24 terms whose dates fall in the given Gregorian year (CST).
func SolarTermsOfYear(year int) []SolarTerm {
	out := make([]SolarTerm, 0, 24)
	start := julianDay(time.Date(year, 1, 1, 0, 0, 0, 0, CST))
	for k := 0; k < 24; k++ {
		// 小寒 (285°) is in early Jan; walk 15° steps from 285°.
		lon := math.Mod(285+float64(k)*15, 360)
		guess := start + 5 + float64(k)*365.2422/24
		jd := termTime(lon, guess)
		idx := int(lon/15) % 24
		out = append(out, SolarTerm{NameZH: termNames[idx][0], NameEN: termNames[idx][1], Longitude: int(lon), Time: fromJulian(jd).In(CST)})
	}
	return out
}

// SolarTermInfo describes the solar term context for a date.
type SolarTermInfo struct {
	Date     string    `json:"date"`
	IsTerm   bool      `json:"is_solar_term_day"`
	Current  SolarTerm `json:"current_term"`
	Next     SolarTerm `json:"next_term"`
	DaysNext int       `json:"days_until_next"`
}

// SolarTermFor returns which term period the date (CST) is in and the next term.
func SolarTermFor(d time.Time) SolarTermInfo {
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, CST)
	var all []SolarTerm
	for y := d.Year() - 1; y <= d.Year()+1; y++ {
		all = append(all, SolarTermsOfYear(y)...)
	}
	info := SolarTermInfo{Date: d.Format("2006-01-02")}
	for i, t := range all {
		td := time.Date(t.Time.Year(), t.Time.Month(), t.Time.Day(), 0, 0, 0, 0, CST)
		if td.After(d) {
			info.Current = all[i-1]
			info.Next = t
			info.DaysNext = int(td.Sub(d).Hours() / 24)
			cd := info.Current.Time
			info.IsTerm = cd.Year() == d.Year() && cd.YearDay() == d.YearDay()
			break
		}
	}
	return info
}
