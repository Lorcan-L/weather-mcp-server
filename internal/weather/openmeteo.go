package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// OpenMeteo is the default, key-free provider.
type OpenMeteo struct {
	HTTP          *http.Client
	GeoURL        string
	ForecastURL   string
	AirQualityURL string
}

// NewOpenMeteo returns a provider pointed at the public Open-Meteo endpoints.
func NewOpenMeteo(timeout time.Duration) *OpenMeteo {
	return &OpenMeteo{
		HTTP:          &http.Client{Timeout: timeout},
		GeoURL:        "https://geocoding-api.open-meteo.com/v1/search",
		ForecastURL:   "https://api.open-meteo.com/v1/forecast",
		AirQualityURL: "https://air-quality-api.open-meteo.com/v1/air-quality",
	}
}

func (o *OpenMeteo) Name() string { return "Open-Meteo" }

const maxAttempts = 3

func transient(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}

// getJSON GETs and decodes JSON. It retries up to maxAttempts-1 times on
// network errors and transient HTTP statuses (429/502/503/504), with short
// exponential backoff (200ms, 400ms). Context cancellation aborts early.
func getJSON(ctx context.Context, c *http.Client, u string, hdr map[string]string, out any) error {
	var last error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "weather-mcp-server-cn")
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := c.Do(req)
		if err != nil {
			last = err
			if attempt == maxAttempts || ctx.Err() != nil {
				return last
			}
			sleepBackoff(ctx, attempt)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			last = fmt.Errorf("upstream %s returned HTTP %d", req.URL.Host, resp.StatusCode)
			if !transient(resp.StatusCode) || attempt == maxAttempts || ctx.Err() != nil {
				return last
			}
			sleepBackoff(ctx, attempt)
			continue
		}
		err = json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		return err
	}
	return last
}

func sleepBackoff(ctx context.Context, attempt int) {
	d := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond // 200ms, 400ms
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func (o *OpenMeteo) Geocode(ctx context.Context, query string) ([]Location, error) {
	q := url.Values{"name": {query}, "count": {"5"}, "language": {"zh"}, "format": {"json"}}
	var r struct {
		Results []struct {
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}
	if err := getJSON(ctx, o.HTTP, o.GeoURL+"?"+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	out := make([]Location, 0, len(r.Results))
	for _, x := range r.Results {
		out = append(out, Location{Name: x.Name, Admin1: x.Admin1, Country: x.Country, Latitude: x.Latitude, Longitude: x.Longitude, Timezone: x.Timezone})
	}
	return out, nil
}

func coords(loc Location) url.Values {
	tz := loc.Timezone
	if tz == "" {
		tz = "auto"
	}
	return url.Values{
		"latitude":  {strconv.FormatFloat(loc.Latitude, 'f', 4, 64)},
		"longitude": {strconv.FormatFloat(loc.Longitude, 'f', 4, 64)},
		"timezone":  {tz},
	}
}

func (o *OpenMeteo) Current(ctx context.Context, loc Location) (*Current, error) {
	q := coords(loc)
	q.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,precipitation,weather_code,wind_speed_10m,wind_direction_10m")
	var r struct {
		Current struct {
			Time string  `json:"time"`
			T    float64 `json:"temperature_2m"`
			AT   float64 `json:"apparent_temperature"`
			RH   float64 `json:"relative_humidity_2m"`
			P    float64 `json:"precipitation"`
			Code int     `json:"weather_code"`
			WS   float64 `json:"wind_speed_10m"`
			WD   float64 `json:"wind_direction_10m"`
		} `json:"current"`
	}
	if err := getJSON(ctx, o.HTTP, o.ForecastURL+"?"+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	c := r.Current
	zh, en := DescribeWMO(c.Code)
	return &Current{Time: c.Time, TempC: c.T, FeelsLikeC: c.AT, Humidity: c.RH, PrecipMM: c.P, WindKph: c.WS, WindDir: c.WD, Code: c.Code, TextZH: zh, TextEN: en}, nil
}

func (o *OpenMeteo) Forecast(ctx context.Context, loc Location, days int) ([]Day, error) {
	q := coords(loc)
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max")
	q.Set("forecast_days", strconv.Itoa(days))
	var r struct {
		Daily struct {
			Time []string  `json:"time"`
			Code []int     `json:"weather_code"`
			Max  []float64 `json:"temperature_2m_max"`
			Min  []float64 `json:"temperature_2m_min"`
			P    []float64 `json:"precipitation_sum"`
			PP   []float64 `json:"precipitation_probability_max"`
		} `json:"daily"`
	}
	if err := getJSON(ctx, o.HTTP, o.ForecastURL+"?"+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	d := r.Daily
	out := make([]Day, 0, len(d.Time))
	at := func(s []float64, i int) float64 {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	for i, t := range d.Time {
		code := 0
		if i < len(d.Code) {
			code = d.Code[i]
		}
		zh, en := DescribeWMO(code)
		out = append(out, Day{Date: t, MaxC: at(d.Max, i), MinC: at(d.Min, i), PrecipMM: at(d.P, i), PrecipProb: at(d.PP, i), Code: code, TextZH: zh, TextEN: en})
	}
	return out, nil
}

// AirQuality is a current air-quality reading.
type AirQuality struct {
	Time    string  `json:"time"`
	USAQI   float64 `json:"us_aqi"`
	EUAQI   float64 `json:"european_aqi"`
	CNAQI   int     `json:"cn_aqi_estimate"`
	Primary string  `json:"cn_primary_pollutant,omitempty"`
	PM25    float64 `json:"pm2_5"`
	PM10    float64 `json:"pm10"`
	O3      float64 `json:"ozone"`
	NO2     float64 `json:"nitrogen_dioxide"`
	SO2     float64 `json:"sulphur_dioxide"`
	CO      float64 `json:"carbon_monoxide"`
	LevelZH string  `json:"level_zh"`
	LevelEN string  `json:"level_en"`
}

// AirQuality fetches current air quality (always via Open-Meteo, key-free).
func (o *OpenMeteo) AirQuality(ctx context.Context, loc Location) (*AirQuality, error) {
	q := coords(loc)
	q.Set("current", "us_aqi,european_aqi,pm2_5,pm10,ozone,nitrogen_dioxide,sulphur_dioxide,carbon_monoxide")
	var r struct {
		Current struct {
			Time string  `json:"time"`
			US   float64 `json:"us_aqi"`
			EU   float64 `json:"european_aqi"`
			PM25 float64 `json:"pm2_5"`
			PM10 float64 `json:"pm10"`
			O3   float64 `json:"ozone"`
			NO2  float64 `json:"nitrogen_dioxide"`
			SO2  float64 `json:"sulphur_dioxide"`
			CO   float64 `json:"carbon_monoxide"`
		} `json:"current"`
	}
	if err := getJSON(ctx, o.HTTP, o.AirQualityURL+"?"+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	c := r.Current
	aqi, primary := ChinaAQI(Pollutants{PM25: c.PM25, PM10: c.PM10, O3: c.O3, NO2: c.NO2, SO2: c.SO2, COmg: c.CO / 1000})
	zh, en := AQILevel(aqi)
	return &AirQuality{Time: c.Time, USAQI: c.US, EUAQI: c.EU, CNAQI: aqi, Primary: primary, PM25: c.PM25, PM10: c.PM10, O3: c.O3, NO2: c.NO2, SO2: c.SO2, CO: c.CO, LevelZH: zh, LevelEN: en}, nil
}
