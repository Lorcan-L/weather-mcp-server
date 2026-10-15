package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolarTermDates(t *testing.T) {
	want := map[int]map[string]string{
		2024: {"立春": "2024-02-04", "清明": "2024-04-04", "夏至": "2024-06-21", "冬至": "2024-12-21"},
		2025: {"小寒": "2025-01-05", "春分": "2025-03-20", "秋分": "2025-09-23", "冬至": "2025-12-21"},
		2026: {"立春": "2026-02-04", "清明": "2026-04-05", "夏至": "2026-06-21", "寒露": "2026-10-08", "冬至": "2026-12-22"},
	}
	for y, m := range want {
		terms := SolarTermsOfYear(y)
		require.Len(t, terms, 24)
		got := map[string]string{}
		for _, x := range terms {
			got[x.NameZH] = x.Time.Format("2006-01-02")
			assert.Equal(t, y, x.Time.Year(), x.NameZH)
		}
		for name, d := range m {
			assert.Equal(t, d, got[name], "%d %s", y, name)
		}
	}
}

func TestSolarTermFor(t *testing.T) {
	info := SolarTermFor(time.Date(2026, 10, 15, 0, 0, 0, 0, CST))
	assert.Equal(t, "寒露", info.Current.NameZH)
	assert.Equal(t, "霜降", info.Next.NameZH)
	assert.False(t, info.IsTerm)
	assert.Equal(t, 8, info.DaysNext) // 霜降 2026-10-23

	info = SolarTermFor(time.Date(2026, 4, 5, 0, 0, 0, 0, CST))
	assert.True(t, info.IsTerm)
	assert.Equal(t, "清明", info.Current.NameZH)
	assert.Equal(t, "Pure Brightness", info.Current.NameEN)
}

func TestChinaAQI(t *testing.T) {
	aqi, p := ChinaAQI(Pollutants{PM25: 75, PM10: 40})
	assert.Equal(t, 100, aqi)
	assert.Equal(t, "PM2.5", p)
	aqi, p = ChinaAQI(Pollutants{PM25: 10, PM10: 20, O3: 50})
	assert.LessOrEqual(t, aqi, 50)
	assert.Equal(t, "", p)
	zh, en := AQILevel(120)
	assert.Equal(t, "轻度污染", zh)
	assert.Equal(t, "Lightly polluted", en)
	aqi, _ = ChinaAQI(Pollutants{PM25: 9999})
	assert.Equal(t, 500, aqi)
}

func TestDescribeWMO(t *testing.T) {
	zh, en := DescribeWMO(95)
	assert.Equal(t, "雷阵雨", zh)
	assert.Equal(t, "Thunderstorm", en)
	zh, _ = DescribeWMO(12345)
	assert.Equal(t, "未知", zh)
}

// FakeOpenMeteo serves canned Open-Meteo responses.
func FakeOpenMeteo(t *testing.T) (*OpenMeteo, func()) {
	mux := http.NewServeMux()
	mux.HandleFunc("/geo", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "zh", r.URL.Query().Get("language"))
		if strings.Contains(r.URL.Query().Get("name"), "厦门") || r.URL.Query().Get("name") == "Xiamen" {
			w.Write([]byte(`{"results":[{"name":"厦门市","admin1":"福建","country":"中国","latitude":24.4798,"longitude":118.0819,"timezone":"Asia/Shanghai"}]}`))
			return
		}
		w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/forecast", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("current") != "" {
			w.Write([]byte(`{"current":{"time":"2026-10-15T14:00","temperature_2m":27.3,"apparent_temperature":29.1,"relative_humidity_2m":70,"precipitation":0,"weather_code":2,"wind_speed_10m":12.5,"wind_direction_10m":90}}`))
			return
		}
		assert.Equal(t, "2", r.URL.Query().Get("forecast_days"))
		w.Write([]byte(`{"daily":{"time":["2026-10-15","2026-10-16"],"weather_code":[61,0],"temperature_2m_max":[28,29],"temperature_2m_min":[22,23],"precipitation_sum":[3.2,0],"precipitation_probability_max":[60,5]}}`))
	})
	mux.HandleFunc("/air", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"current":{"time":"2026-10-15T14:00","us_aqi":61,"european_aqi":35,"pm2_5":20,"pm10":35,"ozone":90,"nitrogen_dioxide":15,"sulphur_dioxide":3,"carbon_monoxide":250}}`))
	})
	srv := httptest.NewServer(mux)
	o := NewOpenMeteo(5 * time.Second)
	o.GeoURL, o.ForecastURL, o.AirQualityURL = srv.URL+"/geo", srv.URL+"/forecast", srv.URL+"/air"
	return o, srv.Close
}

func TestOpenMeteo(t *testing.T) {
	o, done := FakeOpenMeteo(t)
	defer done()
	ctx := context.Background()
	locs, err := o.Geocode(ctx, "厦门")
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "厦门市", locs[0].Name)
	c, err := o.Current(ctx, locs[0])
	require.NoError(t, err)
	assert.Equal(t, 27.3, c.TempC)
	assert.Equal(t, "多云", c.TextZH)
	fc, err := o.Forecast(ctx, locs[0], 2)
	require.NoError(t, err)
	require.Len(t, fc, 2)
	assert.Equal(t, "小雨", fc[0].TextZH)
	a, err := o.AirQuality(ctx, locs[0])
	require.NoError(t, err)
	assert.Equal(t, 35, a.CNAQI) // PM10 35 -> IAQI 35, PM2.5 20 -> 28.6
	assert.Equal(t, "优", a.LevelZH)
}

func TestQWeather(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-QW-Api-Key"))
		switch r.URL.Path {
		case "/geo/v2/city/lookup":
			w.Write([]byte(`{"code":"200","location":[{"name":"厦门","id":"101230201","lat":"24.49","lon":"118.11","adm1":"福建省","country":"中国","tz":"Asia/Shanghai"}]}`))
		case "/v7/weather/now":
			assert.Equal(t, "101230201", r.URL.Query().Get("location"))
			w.Write([]byte(`{"code":"200","now":{"obsTime":"2026-10-15T14:00+08:00","temp":"26","feelsLike":"28","text":"多云","wind360":"90","windSpeed":"10","humidity":"72","precip":"0.0","icon":"101"}}`))
		case "/v7/weather/7d":
			w.Write([]byte(`{"code":"200","daily":[{"fxDate":"2026-10-15","tempMax":"28","tempMin":"22","textDay":"阵雨","precip":"2.0","iconDay":"300"}]}`))
		default:
			w.Write([]byte(`{"code":"404"}`))
		}
	}))
	defer srv.Close()
	q := NewQWeather("k", srv.URL, 5*time.Second)
	ctx := context.Background()
	locs, err := q.Geocode(ctx, "厦门")
	require.NoError(t, err)
	assert.Equal(t, "101230201", locs[0].ID)
	c, err := q.Current(ctx, locs[0])
	require.NoError(t, err)
	assert.Equal(t, 26.0, c.TempC)
	assert.Equal(t, "多云", c.TextZH)
	fc, err := q.Forecast(ctx, locs[0], 5)
	require.NoError(t, err)
	assert.Equal(t, "阵雨", fc[0].TextZH)
	_, err = q.Forecast(ctx, locs[0], 16)
	assert.Error(t, err)
}
