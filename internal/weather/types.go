// Package weather implements China-friendly weather, air quality and solar term
// lookups. Default provider is Open-Meteo (no API key); QWeather is optional.
package weather

import "context"

// Location is a geocoded place.
type Location struct {
	Name      string  `json:"name"`
	Admin1    string  `json:"admin1,omitempty"`
	Country   string  `json:"country,omitempty"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone,omitempty"`
	ID        string  `json:"id,omitempty"` // provider specific id (QWeather)
}

// Current is a current-conditions snapshot.
type Current struct {
	Time       string  `json:"time"`
	TempC      float64 `json:"temperature_c"`
	FeelsLikeC float64 `json:"feels_like_c"`
	Humidity   float64 `json:"humidity_pct"`
	PrecipMM   float64 `json:"precipitation_mm"`
	WindKph    float64 `json:"wind_kph"`
	WindDir    float64 `json:"wind_direction_deg"`
	Code       int     `json:"weather_code"`
	TextZH     string  `json:"text_zh"`
	TextEN     string  `json:"text_en"`
}

// Day is one forecast day.
type Day struct {
	Date       string  `json:"date"`
	MaxC       float64 `json:"max_c"`
	MinC       float64 `json:"min_c"`
	PrecipMM   float64 `json:"precipitation_mm"`
	PrecipProb float64 `json:"precipitation_probability_pct"`
	Code       int     `json:"weather_code"`
	TextZH     string  `json:"text_zh"`
	TextEN     string  `json:"text_en"`
}

// Provider is a weather data source.
type Provider interface {
	Name() string
	Geocode(ctx context.Context, query string) ([]Location, error)
	Current(ctx context.Context, loc Location) (*Current, error)
	Forecast(ctx context.Context, loc Location, days int) ([]Day, error)
}
