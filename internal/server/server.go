// Package server wires the weather tools into an MCP server.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Lorcan-L/weather-mcp-server/internal/weather"
)

const Version = "2.0.0"

// Deps are the data sources used by the tools.
type Deps struct {
	Weather weather.Provider   // Open-Meteo by default, QWeather if configured
	Air     *weather.OpenMeteo // air quality + fallback geocoding
	Now     func() time.Time
}

// New builds the MCP server with all tools registered.
func New(d Deps) *server.MCPServer {
	if d.Now == nil {
		d.Now = time.Now
	}
	s := server.NewMCPServer("weather-mcp-server-cn", Version,
		server.WithToolCapabilities(false), server.WithLogging(),
		server.WithInstructions("中文天气 / Chinese-friendly weather: city names may be Chinese (厦门) or English (Xiamen). Output language via lang=zh|en|both."))

	langOpt := mcp.WithString("lang", mcp.Description("Output language: zh, en, or both (default both) / 输出语言"), mcp.Enum("zh", "en", "both"))
	cityOpt := mcp.WithString("city", mcp.Required(), mcp.Description("City name in Chinese or English, e.g. 厦门 / Xiamen. Prefer city level; district/county names work best with QWeather"))
	ro := mcp.WithReadOnlyHintAnnotation(true)

	s.AddTool(mcp.NewTool("search_city", mcp.WithDescription("Search places by Chinese or English name / 按中文或英文名称搜索城市"), cityOpt, ro),
		d.searchCity)
	s.AddTool(mcp.NewTool("get_current_weather", mcp.WithDescription("Current weather for a city / 查询城市实时天气"), cityOpt, langOpt, ro),
		d.current)
	s.AddTool(mcp.NewTool("get_forecast", mcp.WithDescription("Daily forecast (1-16 days) for a city / 查询未来天气预报"), cityOpt,
		mcp.WithNumber("days", mcp.Description("Number of days, 1-16 (default 3)")), langOpt, ro), d.forecast)
	s.AddTool(mcp.NewTool("get_air_quality", mcp.WithDescription("Current air quality incl. China AQI estimate, US/EU AQI, PM2.5 / 查询空气质量"), cityOpt, langOpt, ro),
		d.air)
	s.AddTool(mcp.NewTool("get_solar_term", mcp.WithDescription("24 solar terms (二十四节气) for a date: current term, next term, or full year list / 查询二十四节气"),
		mcp.WithString("date", mcp.Description("Date YYYY-MM-DD (default today, China time)")),
		mcp.WithNumber("year", mcp.Description("If set, list all 24 terms of that year")), langOpt, ro), d.solarTerm)
	return s
}

func lang(r mcp.CallToolRequest) string {
	l := r.GetString("lang", "both")
	if l != "zh" && l != "en" {
		l = "both"
	}
	return l
}

func pick(l, zh, en string) string {
	switch {
	case l == "zh" || en == "":
		return zh
	case l == "en":
		return en
	default:
		return zh + " / " + en
	}
}

// cityVariants returns lookup variants for Chinese place names: the geocoder
// indexes some cities as "厦门市" and others as "北京", so try both forms.
func cityVariants(city string) []string {
	out := []string{city}
	for _, suf := range []string{"市", "区", "县"} {
		if t := strings.TrimSuffix(city, suf); t != city && t != "" {
			return append(out, t)
		}
	}
	if hasHan(city) {
		out = append(out, city+"市", city+"区", city+"县")
	}
	return out
}

func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// resolve geocodes a city, trying Chinese name variants, the configured
// provider first and Open-Meteo as fallback. Results in China are preferred.
func (d Deps) resolve(ctx context.Context, city string) (weather.Location, error) {
	city = strings.TrimSpace(city)
	providers := []weather.Provider{d.Weather}
	if d.Air != nil && d.Weather != weather.Provider(d.Air) {
		providers = append(providers, d.Air)
	}
	var lastErr error
	for _, p := range providers {
		for _, v := range cityVariants(city) {
			locs, err := p.Geocode(ctx, v)
			if err != nil {
				lastErr = err
				break
			}
			if len(locs) == 0 {
				continue
			}
			for _, l := range locs {
				if l.Country == "中国" || l.Country == "China" {
					return l, nil
				}
			}
			return locs[0], nil
		}
	}
	if lastErr != nil {
		return weather.Location{}, lastErr
	}
	return weather.Location{}, fmt.Errorf("未找到城市 / city not found: %s", city)
}

func place(l weather.Location) string {
	parts := []string{l.Name}
	if l.Admin1 != "" && l.Admin1 != l.Name {
		parts = append(parts, l.Admin1)
	}
	if l.Country != "" {
		parts = append(parts, l.Country)
	}
	return strings.Join(parts, ", ")
}

func result(text string, data any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(data, "", "  ")
	return mcp.NewToolResultText(text + "\n\n```json\n" + string(b) + "\n```")
}

func (d Deps) searchCity(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, err := r.RequireString("city")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	locs, err := d.Weather.Geocode(ctx, city)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var sb strings.Builder
	for i, l := range locs {
		fmt.Fprintf(&sb, "%d. %s (%.4f, %.4f)\n", i+1, place(l), l.Latitude, l.Longitude)
	}
	if len(locs) == 0 {
		sb.WriteString("未找到 / not found")
	}
	return result(sb.String(), locs), nil
}

func (d Deps) current(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, err := r.RequireString("city")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	loc, err := d.resolve(ctx, city)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	c, err := d.Weather.Current(ctx, loc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	l := lang(r)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s — %s (%s)\n", place(loc), pick(l, "实时天气", "Current weather"), c.Time)
	fmt.Fprintf(&sb, "%s: %s\n", pick(l, "天气", "Condition"), pick(l, c.TextZH, c.TextEN))
	fmt.Fprintf(&sb, "%s: %.1f°C (%s %.1f°C)\n", pick(l, "气温", "Temperature"), c.TempC, pick(l, "体感", "feels like"), c.FeelsLikeC)
	fmt.Fprintf(&sb, "%s: %.0f%%  %s: %.1f mm\n", pick(l, "湿度", "Humidity"), c.Humidity, pick(l, "降水", "Precipitation"), c.PrecipMM)
	fmt.Fprintf(&sb, "%s: %.1f km/h, %.0f°\n", pick(l, "风速", "Wind"), c.WindKph, c.WindDir)
	fmt.Fprintf(&sb, "%s: %s", pick(l, "数据源", "Source"), d.Weather.Name())
	return result(sb.String(), map[string]any{"location": loc, "current": c, "source": d.Weather.Name()}), nil
}

func (d Deps) forecast(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, err := r.RequireString("city")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	days := r.GetInt("days", 3)
	if days < 1 {
		days = 1
	}
	if days > 16 {
		days = 16
	}
	loc, err := d.resolve(ctx, city)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fc, err := d.Weather.Forecast(ctx, loc, days)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	l := lang(r)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s — %s (%d)\n", place(loc), pick(l, "天气预报", "Forecast"), len(fc))
	for _, x := range fc {
		fmt.Fprintf(&sb, "%s  %s  %.0f~%.0f°C  %s %.1f mm", x.Date, pick(l, x.TextZH, x.TextEN), x.MinC, x.MaxC, pick(l, "降水", "precip"), x.PrecipMM)
		if x.PrecipProb > 0 {
			fmt.Fprintf(&sb, " (%.0f%%)", x.PrecipProb)
		}
		sb.WriteString("\n")
	}
	fmt.Fprintf(&sb, "%s: %s", pick(l, "数据源", "Source"), d.Weather.Name())
	return result(sb.String(), map[string]any{"location": loc, "daily": fc, "source": d.Weather.Name()}), nil
}

func (d Deps) air(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, err := r.RequireString("city")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	loc, err := d.resolve(ctx, city)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	a, err := d.Air.AirQuality(ctx, loc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	l := lang(r)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s — %s (%s)\n", place(loc), pick(l, "空气质量", "Air quality"), a.Time)
	fmt.Fprintf(&sb, "%s: %d %s", pick(l, "中国AQI(估算)", "China AQI (est.)"), a.CNAQI, pick(l, a.LevelZH, a.LevelEN))
	if a.Primary != "" {
		fmt.Fprintf(&sb, ", %s: %s", pick(l, "首要污染物", "primary"), a.Primary)
	}
	fmt.Fprintf(&sb, "\nUS AQI: %.0f  EU AQI: %.0f\n", a.USAQI, a.EUAQI)
	fmt.Fprintf(&sb, "PM2.5 %.1f  PM10 %.1f  O3 %.1f  NO2 %.1f  SO2 %.1f µg/m³  CO %.0f µg/m³\n", a.PM25, a.PM10, a.O3, a.NO2, a.SO2, a.CO)
	sb.WriteString(pick(l, "数据源: Open-Meteo 空气质量模型（CAMS），非国控站点实测", "Source: Open-Meteo air-quality model (CAMS), not official monitoring stations"))
	return result(sb.String(), map[string]any{"location": loc, "air_quality": a}), nil
}

func (d Deps) solarTerm(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	l := lang(r)
	if y := r.GetInt("year", 0); y != 0 {
		if y < 1900 || y > 2100 {
			return mcp.NewToolResultError("year must be 1900-2100"), nil
		}
		terms := weather.SolarTermsOfYear(y)
		var sb strings.Builder
		fmt.Fprintf(&sb, "%d %s\n", y, pick(l, "年二十四节气（北京时间）", "solar terms (China Standard Time)"))
		for _, t := range terms {
			fmt.Fprintf(&sb, "%s  %s\n", t.Time.Format("2006-01-02 15:04"), pick(l, t.NameZH, t.NameEN))
		}
		return result(sb.String(), terms), nil
	}
	date := d.Now().In(weather.CST)
	if s := r.GetString("date", ""); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, weather.CST)
		if err != nil {
			return mcp.NewToolResultError("date must be YYYY-MM-DD / 日期格式应为 YYYY-MM-DD"), nil
		}
		date = t
	}
	info := weather.SolarTermFor(date)
	var sb strings.Builder
	if info.IsTerm {
		fmt.Fprintf(&sb, "%s %s %s (%s)\n", info.Date, pick(l, "今天是", "is"), pick(l, info.Current.NameZH, info.Current.NameEN), info.Current.Time.Format("15:04 CST"))
	} else {
		fmt.Fprintf(&sb, "%s %s: %s (%s %s)\n", info.Date, pick(l, "所处节气", "current term"), pick(l, info.Current.NameZH, info.Current.NameEN), pick(l, "始于", "since"), info.Current.Time.Format("2006-01-02"))
	}
	fmt.Fprintf(&sb, "%s: %s — %s (%d %s)", pick(l, "下一个节气", "Next term"), pick(l, info.Next.NameZH, info.Next.NameEN), info.Next.Time.Format("2006-01-02 15:04 CST"), info.DaysNext, pick(l, "天后", "days"))
	return result(sb.String(), info), nil
}
