package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeatherQuery contains latitude and longitude for weather.
type WeatherQuery struct {
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Units     string  `json:"units"` // celsius or fahrenheit
}

// WeatherResult contains parsed current weather and forecast.
type WeatherResult struct {
	City         string  `json:"city"`
	Temperature  float64 `json:"temperature"`
	Unit         string  `json:"unit"`
	Condition    string  `json:"condition"`
	WeatherCode  int     `json:"weatherCode"`
	WindSpeed    float64 `json:"windSpeed"`
	Humidity     int     `json:"humidity"`
	IsDay        bool    `json:"isDay"`
	Online       bool    `json:"online"`
	ErrorMessage string  `json:"error,omitempty"`
}

// QueryWeather fetches real-time meteorological conditions from Open-Meteo.
func QueryWeather(ctx context.Context, q WeatherQuery) WeatherResult {
	if q.Latitude == 0 && q.Longitude == 0 {
		q.Latitude = 40.7128
		q.Longitude = -74.0060
		if q.City == "" {
			q.City = "New York"
		}
	}

	tempUnit := "celsius"
	if q.Units == "fahrenheit" {
		tempUnit = "fahrenheit"
	}

	apiURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,is_day&temperature_unit=%s",
		q.Latitude, q.Longitude, tempUnit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return WeatherResult{Online: false, ErrorMessage: err.Error()}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return WeatherResult{Online: false, ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherResult{Online: false, ErrorMessage: fmt.Sprintf("Weather API returned %d", resp.StatusCode)}
	}

	var data struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			Humidity    int     `json:"relative_humidity_2m"`
			WeatherCode int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			IsDay       int     `json:"is_day"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return WeatherResult{Online: false, ErrorMessage: err.Error()}
	}

	unitSymbol := "°C"
	if tempUnit == "fahrenheit" {
		unitSymbol = "°F"
	}

	return WeatherResult{
		City:        q.City,
		Temperature: data.Current.Temperature,
		Unit:        unitSymbol,
		Condition:   weatherCodeToText(data.Current.WeatherCode),
		WeatherCode: data.Current.WeatherCode,
		WindSpeed:   data.Current.WindSpeed,
		Humidity:    data.Current.Humidity,
		IsDay:       data.Current.IsDay == 1,
		Online:      true,
	}
}

func weatherCodeToText(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 61, 63, 65:
		return "Rain"
	case 71, 73, 75:
		return "Snow fall"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Partly Cloudy"
	}
}
