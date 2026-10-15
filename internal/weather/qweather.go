package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// QWeather (和风天气) optional provider. Requires QWEATHER_API_KEY and
// QWEATHER_API_HOST (the per-account host shown in the QWeather console).
type QWeather struct {
	HTTP *http.Client
	Key  string
	Host string // e.g. https://abc123.re.qweatherapi.com
}

func NewQWeather(key, host string, timeout time.Duration) *QWeather {
	if host != "" && !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}
	return &QWeather{HTTP: &http.Client{Timeout: timeout}, Key: key, Host: strings.TrimRight(host, "/")}
}

func (q *QWeather) Name() string { return "QWeather 和风天气" }

func (q *QWeather) get(ctx context.Context, path string, v url.Values, out any) error {
	v.Set("lang", "zh")
	return getJSON(ctx, q.HTTP, q.Host+path+"?"+v.Encode(), map[string]string{"X-QW-Api-Key": q.Key}, out)
}

func check(code string) error {
	if code != "200" {
		return fmt.Errorf("QWeather returned code %s", code)
	}
	return nil
}

func f(s string) float64 { x, _ := strconv.ParseFloat(s, 64); return x }

func (q *QWeather) Geocode(ctx context.Context, query string) ([]Location, error) {
	var r struct {
		Code     string `json:"code"`
		Location []struct {
			Name, ID, Lat, Lon, Adm1, Country, Tz string
		} `json:"location"`
	}
	if err := q.get(ctx, "/geo/v2/city/lookup", url.Values{"location": {query}, "number": {"5"}}, &r); err != nil {
		return nil, err
	}
	if err := check(r.Code); err != nil {
		return nil, err
	}
	var out []Location
	for _, l := range r.Location {
		out = append(out, Location{Name: l.Name, ID: l.ID, Admin1: l.Adm1, Country: l.Country, Latitude: f(l.Lat), Longitude: f(l.Lon), Timezone: l.Tz})
	}
	return out, nil
}

func (q *QWeather) loc(l Location) string {
	if l.ID != "" {
		return l.ID
	}
	return fmt.Sprintf("%.2f,%.2f", l.Longitude, l.Latitude)
}

func (q *QWeather) Current(ctx context.Context, l Location) (*Current, error) {
	var r struct {
		Code string `json:"code"`
		Now  struct {
			ObsTime, Temp, FeelsLike, Text, Wind360, WindSpeed, Humidity, Precip, Icon string
		} `json:"now"`
	}
	if err := q.get(ctx, "/v7/weather/now", url.Values{"location": {q.loc(l)}}, &r); err != nil {
		return nil, err
	}
	if err := check(r.Code); err != nil {
		return nil, err
	}
	n := r.Now
	return &Current{Time: n.ObsTime, TempC: f(n.Temp), FeelsLikeC: f(n.FeelsLike), Humidity: f(n.Humidity), PrecipMM: f(n.Precip), WindKph: f(n.WindSpeed), WindDir: f(n.Wind360), Code: int(f(n.Icon)), TextZH: n.Text, TextEN: ""}, nil
}

func (q *QWeather) Forecast(ctx context.Context, l Location, days int) ([]Day, error) {
	path := "/v7/weather/3d"
	switch {
	case days > 10:
		path = "/v7/weather/15d"
	case days > 7:
		path = "/v7/weather/10d"
	case days > 3:
		path = "/v7/weather/7d"
	}
	var r struct {
		Code  string `json:"code"`
		Daily []struct {
			FxDate, TempMax, TempMin, TextDay, Precip, IconDay string
		} `json:"daily"`
	}
	if err := q.get(ctx, path, url.Values{"location": {q.loc(l)}}, &r); err != nil {
		return nil, err
	}
	if err := check(r.Code); err != nil {
		return nil, err
	}
	var out []Day
	for i, d := range r.Daily {
		if i >= days {
			break
		}
		out = append(out, Day{Date: d.FxDate, MaxC: f(d.TempMax), MinC: f(d.TempMin), PrecipMM: f(d.Precip), Code: int(f(d.IconDay)), TextZH: d.TextDay})
	}
	return out, nil
}
