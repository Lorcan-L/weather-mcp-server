package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Lorcan-L/weather-mcp-server/internal/weather"
)

func fake(t *testing.T) *weather.OpenMeteo {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geo":
			if r.URL.Query().Get("name") == "厦门市" {
				w.Write([]byte(`{"results":[{"name":"厦门市","admin1":"福建","country":"中国","latitude":24.48,"longitude":118.08,"timezone":"Asia/Shanghai"}]}`))
			} else {
				w.Write([]byte(`{}`))
			}
		case "/forecast":
			if r.URL.Query().Get("current") != "" {
				w.Write([]byte(`{"current":{"time":"2026-10-15T14:00","temperature_2m":27.3,"apparent_temperature":29,"relative_humidity_2m":70,"weather_code":0}}`))
			} else {
				w.Write([]byte(`{"daily":{"time":["2026-10-15"],"weather_code":[63],"temperature_2m_max":[28],"temperature_2m_min":[22],"precipitation_sum":[5],"precipitation_probability_max":[80]}}`))
			}
		case "/air":
			w.Write([]byte(`{"current":{"time":"2026-10-15T14:00","us_aqi":80,"european_aqi":40,"pm2_5":50,"pm10":60}}`))
		}
	}))
	t.Cleanup(srv.Close)
	o := weather.NewOpenMeteo(5 * time.Second)
	o.GeoURL, o.ForecastURL, o.AirQualityURL = srv.URL+"/geo", srv.URL+"/forecast", srv.URL+"/air"
	return o
}

func call(t *testing.T, c *client.Client, name string, args map[string]any) (string, bool) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	return res.Content[0].(mcp.TextContent).Text, res.IsError
}

func TestTools(t *testing.T) {
	o := fake(t)
	s := New(Deps{Weather: o, Air: o, Now: func() time.Time { return time.Date(2026, 10, 15, 6, 0, 0, 0, time.UTC) }})
	c, err := client.NewInProcessClient(s)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.Start(ctx))
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	_, err = c.Initialize(ctx, init)
	require.NoError(t, err)

	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	assert.Len(t, tools.Tools, 5)

	txt, isErr := call(t, c, "get_current_weather", map[string]any{"city": "厦门"})
	assert.False(t, isErr)
	assert.Contains(t, txt, "厦门市")
	assert.Contains(t, txt, "晴 / Clear sky")
	assert.Contains(t, txt, "27.3°C")

	txt, _ = call(t, c, "get_current_weather", map[string]any{"city": "厦门", "lang": "en"}) // 市 suffix added
	assert.Contains(t, txt, "Clear sky")
	assert.Contains(t, txt, "Condition: Clear sky\n")

	txt, _ = call(t, c, "get_forecast", map[string]any{"city": "厦门", "days": 1, "lang": "zh"})
	assert.Contains(t, txt, "中雨")

	txt, _ = call(t, c, "get_air_quality", map[string]any{"city": "厦门"})
	assert.Contains(t, txt, "中国AQI")
	assert.Contains(t, txt, "PM2.5")

	txt, _ = call(t, c, "get_solar_term", map[string]any{"lang": "zh"})
	assert.Contains(t, txt, "寒露")
	assert.Contains(t, txt, "霜降")

	txt, _ = call(t, c, "get_solar_term", map[string]any{"year": 2026})
	assert.Contains(t, txt, "2026-02-04")

	_, isErr = call(t, c, "get_current_weather", map[string]any{"city": "不存在的地方xyz"})
	assert.True(t, isErr)
	_, isErr = call(t, c, "get_solar_term", map[string]any{"date": "bad"})
	assert.True(t, isErr)
}

func TestCityVariants(t *testing.T) {
	assert.Equal(t, []string{"厦门", "厦门市", "厦门区", "厦门县"}, cityVariants("厦门"))
	assert.Equal(t, []string{"厦门市", "厦门"}, cityVariants("厦门市"))
	assert.Equal(t, []string{"Xiamen"}, cityVariants("Xiamen"))
}
