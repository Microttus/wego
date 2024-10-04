package backends

import (
	"encoding/json"
	"fmt"
	"github.com/schachmat/wego/iface"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type yrConfig struct {
	debug bool
}

type yrResponse struct {
	Type     string `json:"type"`
	Geometry struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Meta struct {
			UpdatedAt string `json:"updated_at"`
			Units     struct {
				AirTemperature string `json:"air_temperature"`
			} `json:"units"`
		} `json:"meta"`
		TimeSeries []timeSeriesBlock `json:"timeseries"`
	} `json:"properties"`
}

type timeSeriesBlock struct {
	Time string `json:"time"`
	Data struct {
		Instant struct {
			Details struct {
				AirPressureAtSeaLevel float64 `json:"air_pressure_at_sea_level"`
				AirTemperature        float64 `json:"air_temperature"`
				CloudAreaFraction     float64 `json:"cloud_area_fraction"`
				RelativeHumidity      float64 `json:"relative_humidity"`
				WindFromDirection     float64 `json:"wind_from_direction"`
				WindSpeed             float64 `json:"wind_speed"`
			} `json:"details"`
		} `json:"instant"`
		Next12Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
		} `json:"next_12_hours"`
		Next1Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				PrecipitationAmount float32 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_1_hours"`
		Next6Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				PrecipitationAmount float64 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_6_hours"`
	} `json:"data"`
}

const (
	yrURI = "https://api.met.no/weatherapi/locationforecast/2.0/compact?"
)

func (c *yrConfig) Setup() {
	//flag.StringVar(&c.apiKey, "wwo-api-key", "", "worldweatheronline backend: the api `KEY` to use")
	//flag.StringVar(&c.language, "wwo-lang", "en", "worldweatheronline backend: the `LANGUAGE` to request from worldweatheronline")
	//flag.BoolVar(&c.debug, "wwo-debug", false, "worldweatheronline backend: print raw requests and responses")
}

func (c *yrConfig) conditionParser(dayInfo timeSeriesBlock) (iface.Cond, error) {
	var ret iface.Cond
	yrWeatherMap := map[string]iface.WeatherCode{
		"clearsky_night":   iface.CodeSunny,
		"clearsky_day":     iface.CodeSunny,
		"cloudy":           iface.CodeCloudy,
		"partlycloudy_day": iface.CodePartlyCloudy,
	}

	if val, ok := yrWeatherMap[dayInfo.Data.Next1Hours.Summary.SymbolCode]; ok {
		ret.Code = val
	} else {
		ret.Code = iface.CodeUnknown
	}
	if &dayInfo.Data.Next1Hours.Details.PrecipitationAmount != nil {
		mmh := dayInfo.Data.Next1Hours.Details.PrecipitationAmount / 1000
		ret.PrecipM = &mmh
	}

	return ret, nil
}

func (c *yrConfig) dayParser(series []timeSeriesBlock, numDays int) []iface.Day {
	var forecast []iface.Day

	return forecast
}

func (c *yrConfig) fetch(url string) (*yrResponse, error) {
	//c.debug = true
	if c.debug {
		fmt.Printf("Fetching %s\n", url)
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	// Set the custom User-Agent header
	req.Header.Set("User-Agent", "WegoApp/1.0 (microttus@gmail.com)")

	// Execute the request
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Unable to get (%s): %v", url, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response body (%s): %v", url, err)
	}

	if c.debug {
		fmt.Printf("Response (%s):\n%s\n", url, string(body))
	}

	var resp yrResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal response (%s): %v\nThe json body is: %s", url, err, string(body))
	}
	if resp.Type != "Feature" {
		return nil, fmt.Errorf("Erroneous response body: %s", string(body))
	} else {
		log.Println("Successfully fetched yr " + resp.Type)
		log.Println("Weather now: " + resp.Properties.TimeSeries[0].Data.Next1Hours.Summary.SymbolCode)
	}
	return &resp, nil
}

func (c *yrConfig) Fetch(location string, numdays int) iface.Data {
	//var params []string
	//var resp yrResponse
	var ret iface.Data
	loc := ""

	if matched, err := regexp.MatchString(`^-?[0-9]*(\.[0-9]+)?,-?[0-9]*(\.[0-9]+)?$`, location); matched && err == nil {
		s := strings.Split(location, ",")
		loc = fmt.Sprintf("lat=%s&lon=%s", s[0], s[1])
	} else if matched, err = regexp.MatchString(`^[0-9].*`, location); matched && err == nil {
		loc = "zip=" + location
	} else {
		loc = "q=" + location
	}

	resp, err := c.fetch(yrURI + loc)
	if err != nil {
		log.Fatalf("Failed to fetch weather data: %v\n", err)
	}
	ret.Current, _ = c.conditionParser(resp.Properties.TimeSeries[0])
	ret.Location = fmt.Sprintf("%s", loc)

	//if err != nil {
	//	log.Fatalf("Failed to fetch weather data: %v\n", err)
	//}

	print("Fetch yr")

	return ret
}

func init() {
	iface.AllBackends["yr"] = &yrConfig{}
}
