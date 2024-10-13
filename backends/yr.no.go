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
	"time"
)

type yrConfig struct {
	apiKey string
	lang   string
	debug  bool
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
				AirPressureAtSeaLevel float32 `json:"air_pressure_at_sea_level"`
				AirTemperature        float32 `json:"air_temperature"`
				CloudAreaFraction     float32 `json:"cloud_area_fraction"`
				RelativeHumidity      float32 `json:"relative_humidity"`
				WindFromDirection     float32 `json:"wind_from_direction"`
				WindSpeed             float32 `json:"wind_speed"`
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
				PrecipitationAmount float32 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_6_hours"`
	} `json:"data"`
}

type geonameResponse struct {
	TotalResultsCount int `json:"totalResultsCount"`
	Geonames          []struct {
		AdminCode1  string `json:"adminCode1"`
		Lng         string `json:"lng"`
		ToponymName string `json:"toponymName"`
		CountryId   string `json:"countryId"`
		Fcl         string `json:"fcl"`
		Population  int    `json:"population"`
		CountryCode string `json:"countryCode"`
		Name        string `json:"name"`
		FclName     string `json:"fclName"`
		AdminCodes1 struct {
			ISO3166_2 string `json:"ISO3166_2"`
		} `json:"adminCodes1"`
		CountryName string `json:"countryName"`
		FcodeName   string `json:"fcodeName"`
		AdminName1  string `json:"adminName1"`
		Lat         string `json:"lat"`
		Fcode       string `json:"fcode"`
	} `json:"geonames"`
}

type sunResponse struct {
	Copyright  string `json:"copyright"`
	LicenseURL string `json:"licenseURL"`
	Type       string `json:"type"`
	Geometry   struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	When struct {
		Interval []string `json:"interval"`
	} `json:"when"`
	Properties struct {
		Body    string `json:"body"`
		Sunrise struct {
			Time    string `json:"time"`
			Azimuth string `json:"azimuth"`
		} `json:"sunrise"`
		Sunset struct {
			Time    string  `json:"time"`
			Azimuth float64 `json:"azimuth"`
		} `json:"sunset"`
		Solarnoon struct {
			Time                string  `json:"time"`
			DiscCentreElevation float64 `json:"disc_centre_elevation"`
			Visible             bool    `json:"visible"`
		} `json:"solarnoon"`
		Solarmidnigth struct {
			Time                string  `json:"time"`
			DiscCentreElevation float64 `json:"disc_centre_elevation"`
			Visible             bool    `json:"visible"`
		} `json:"solararmidnigth"`
	} `json:"properties"`
}

type moonResponse struct {
	Copyright  string `json:"copyright"`
	LicenseURL string `json:"licenseURL"`
	Type       string `json:"type"`
	Geometry   struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	When struct {
		Interval []string `json:"interval"`
	} `json:"when"`
	Properties struct {
		Body     string `json:"body"`
		Moonrise struct {
			Time    string `json:"time"`
			Azimuth string `json:"azimuth"`
		} `json:"moonrise"`
		Moonset struct {
			Time    string `json:"time"`
			Azimuth string `json:"azimuth"`
		} `json:"moonset"`
		HighMoon struct {
			Time                string  `json:"time"`
			DiscCentreElevation float64 `json:"disc_centre_elevation"`
			Visible             bool    `json:"visible"`
		} `json:"high_moon"`
		LowMoon struct {
			Time                string  `json:"time"`
			DiscCentreElevation float64 `json:"disc_centre_elevation"`
			Visible             bool    `json:"visible"`
		} `json:"low_moon"`
		Moonphase float64 `json:"moonphase"`
	} `json:"properties"`
}

const (
	yrURI       = "https://api.met.no/weatherapi/locationforecast/2.0/compact?"
	geonamesURI = "http://api.geonames.org/searchJSON?"
	sunURI      = "https://api.met.no/weatherapi/sunrise/3.0/"
)

func (c *yrConfig) Setup() {
	//flag.StringVar(&c.apiKey, "wwo-api-key", "", "worldweatheronline backend: the api `KEY` to use")
	//flag.StringVar(&c.language, "wwo-lang", "en", "worldweatheronline backend: the `LANGUAGE` to request from worldweatheronline")
	//flag.BoolVar(&c.debug, "wwo-debug", false, "worldweatheronline backend: print raw requests and responses")
}

func (c *yrConfig) conditionParser(dayInfo timeSeriesBlock) (iface.Cond, error) {
	var ret iface.Cond
	yrWeatherMap := map[string]iface.WeatherCode{
		"clearsky_night":                    iface.CodeSunny,
		"clearsky_day":                      iface.CodeSunny,
		"cloudy":                            iface.CodeCloudy,
		"fair_day":                          iface.CodePartlyCloudy,
		"fair_night":                        iface.CodePartlyCloudy,
		"fog":                               iface.CodeFog,
		"heavyrain":                         iface.CodeHeavyRain,
		"heavyrainthunder":                  iface.CodeThunderyHeavyRain,
		"heavyrainshowers_day":              iface.CodeHeavyShowers,
		"heavyrainshowers_night":            iface.CodeHeavyShowers,
		"heavyrainshowersandthunder_day":    iface.CodeThunderyHeavyRain,
		"heavyrainshowersandthunder_night":  iface.CodeThunderyHeavyRain,
		"heavysleet":                        iface.CodeHeavySnowShowers,
		"heavysleetandthunder":              iface.CodeThunderySnowShowers,
		"heavysleetshowers_day":             iface.CodeHeavySnowShowers,
		"heavysleetshowers_night":           iface.CodeHeavySnowShowers,
		"heavysleetshowersandthunder_day":   iface.CodeThunderySnowShowers,
		"heavysleetshowersandthunder_night": iface.CodeThunderySnowShowers,
		"heavysnow":                         iface.CodeHeavySnow,
		"heavysnowandthunder":               iface.CodeThunderySnowShowers,
		"heavysnowshowers_day":              iface.CodeHeavySnowShowers,
		"heavysnowshowers_night":            iface.CodeHeavySnowShowers,
		"heavysnowshowersandthunder_day":    iface.CodeThunderySnowShowers,
		"heavysnowshowersandthunder_night":  iface.CodeThunderySnowShowers,
		"lightrain":                         iface.CodeLightRain,
		"lightrainandthunder":               iface.CodeThunderyShowers,
		"lightrainshowers_day":              iface.CodeLightShowers,
		"lightrainshowers_night":            iface.CodeLightShowers,
		"lightrainshowersandthunder_day":    iface.CodeThunderyShowers,
		"lightrainshowersandthunder_night":  iface.CodeThunderyShowers,
		"lightsleet":                        iface.CodeLightSleet,
		"lightsleetandthunder":              iface.CodeThunderySnowShowers,
		"lightsleetshowers_day":             iface.CodeLightSleetShowers,
		"lightsleetshowers_night":           iface.CodeLightSleetShowers,
		"lightsnow":                         iface.CodeLightSnow,
		"lightsnowandthunder":               iface.CodeThunderySnowShowers,
		"lightsnowshowers_day":              iface.CodeThunderySnowShowers,
		"lightsnowshowers_night":            iface.CodeThunderySnowShowers,
		"lightsleetshowersandthunder_day":   iface.CodeThunderySnowShowers,
		"lightsleetshowersandthunder_night": iface.CodeThunderySnowShowers,
		"partlycloudy_day":                  iface.CodePartlyCloudy,
		"partlycloudy_night":                iface.CodePartlyCloudy,
		"rain":                              iface.CodeLightRain,
		"rainandthunder":                    iface.CodeThunderyShowers,
		"rainshowers_day":                   iface.CodeLightShowers,
		"rainshowers_night":                 iface.CodeLightShowers,
		"rainshowersandthunder_day":         iface.CodeThunderyShowers,
		"rainshowersandthunder_night":       iface.CodeThunderyShowers,
		"sleet":                             iface.CodeLightSleet,
		"sleetandthunder":                   iface.CodeThunderySnowShowers,
		"sleetshowers_day":                  iface.CodeLightSleetShowers,
		"sleetshowers_night":                iface.CodeLightSleetShowers,
		"sleetshowersandthunder_day":        iface.CodeThunderyShowers,
		"sleetshowersandthunder_night":      iface.CodeThunderyShowers,
		"snow":                              iface.CodeHeavySnow,
		"snowandthunder":                    iface.CodeThunderySnowShowers,
		"snowshowers_day":                   iface.CodeHeavySnowShowers,
		"snowshowers_night":                 iface.CodeHeavySnowShowers,
		"snowshowersandthunder_day":         iface.CodeThunderyShowers,
		"snowshowersandthunder_night":       iface.CodeThunderyShowers,
	}

	if val, ok := yrWeatherMap[dayInfo.Data.Next6Hours.Summary.SymbolCode]; ok {
		ret.Code = val
	} else if val, ok := yrWeatherMap[dayInfo.Data.Next1Hours.Summary.SymbolCode]; ok {
		ret.Code = val
	} else {
		ret.Code = iface.CodeUnknown
	}

	temp := dayInfo.Data.Instant.Details.AirTemperature
	ret.TempC = &temp

	if &dayInfo.Data.Next6Hours.Details.PrecipitationAmount != nil {
		mmh := dayInfo.Data.Next6Hours.Details.PrecipitationAmount / 1000
		ret.PrecipM = &mmh
	}

	WindKmph := dayInfo.Data.Instant.Details.WindSpeed / 3.6
	ret.WindspeedKmph = &WindKmph

	WindDeg := int(dayInfo.Data.Instant.Details.WindFromDirection)
	ret.WinddirDegree = &WindDeg

	Humid := int(dayInfo.Data.Instant.Details.RelativeHumidity)
	ret.Humidity = &Humid

	ret.Time, _ = time.Parse(time.RFC3339, dayInfo.Time)

	return ret, nil
}

func (c *yrConfig) dayParser(series []timeSeriesBlock, numDays int) []iface.Day {
	var forecast []iface.Day
	var day *iface.Day

	for _, data := range series {
		slot, err := c.conditionParser(data)
		if err != nil {
			log.Println("Error parsing hourly weather condition:", err)
			continue
		}
		if day == nil {
			day = new(iface.Day)
			day.Date = slot.Time
		}
		if day.Date.Day() == slot.Time.Day() {
			day.Slots = append(day.Slots, slot)
		}
		if day.Date.Day() != slot.Time.Day() {
			forecast = append(forecast, *day)
			if len(forecast) >= numDays {
				break
			}
			day = new(iface.Day)
			day.Date = slot.Time
			day.Slots = append(day.Slots, slot)

			log.Println("New day")
		}
	}

	return forecast
}

func (c *yrConfig) geonameParser(url string) (geoName string, geoCoordinates string, err error) {
	//Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("Failed to create request: %v", err)
	}

	// Set the custom User-Agent header
	//req.Header.Set("User-Agent", "YrForWegoApp/1.0")

	// Execute the request
	res, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Unable to get (%s): %v", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", "", fmt.Errorf("Unable to read response body (%s): %v", url, err)
	}
	if c.debug {
		fmt.Printf("Response (%s):\n%s\n", url, string(body))
	}

	var resp geonameResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return "", "", fmt.Errorf("Unable to unmarshal response (%s): %v\nThe json body is: %s", url, err, string(body))
	}

	retGeoName := resp.Geonames[0].Name + ", " + resp.Geonames[0].AdminName1 + ", " + resp.Geonames[0].CountryName
	retGeoCoordinates := "lat=" + resp.Geonames[0].Lat + "&lon=" + resp.Geonames[0].Lng

	return retGeoName, retGeoCoordinates, nil

}

func (c *yrConfig) sunParser(url string, coord string, date string) (parsedAstroDay iface.Astro, err error) {
	//Create a new HTTP client
	client := &http.Client{}

	var AstroDay iface.Astro

	fmt.Println("In dato: " + date)

	sunParsingURL := url + "sun?" + coord + "&" + date
	//moonParsingURL := url + "moon?" + coord + "&" + date

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", sunParsingURL, nil)
	if err != nil {
		return AstroDay, fmt.Errorf("Failed to create request: %v", err)
	}

	// Set the custom User-Agent header
	//req.Header.Set("User-Agent", "YrForWegoApp/1.0")

	// Execute the request
	res, err := client.Do(req)
	if err != nil {
		return AstroDay, fmt.Errorf("Unable to get (%s): %v", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	//body, err := io.ReadAll(res.Body)
	//if err != nil {
	//	return "", "", fmt.Errorf("Unable to read response body (%s): %v", url, err)
	//}
	//if c.debug {
	//	fmt.Printf("Response (%s):\n%s\n", url, string(body))
	//}

	return AstroDay, nil
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
	req.Header.Set("User-Agent", "WegoApp/1.0 (https://github.com/Microttus/wego)")

	// Execute the request
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Unable to get (%s): %v", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

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
	}
	return &resp, nil
}

func (c *yrConfig) Fetch(location string, numdays int) iface.Data {
	//var params []string
	//var resp yrResponse
	var ret iface.Data
	loc := ""
	var name string

	if matched, err := regexp.MatchString(`^-?[0-9]*(\.[0-9]+)?,-?[0-9]*(\.[0-9]+)?$`, location); matched && err == nil {
		s := strings.Split(location, ",")
		loc = fmt.Sprintf("lat=%s&lon=%s", s[0], s[1])
		name = loc
	} else if matched, err = regexp.MatchString(`^[0-9].*`, location); matched && err == nil {
		loc = "zip=" + location
	} else {
		qLocation := fmt.Sprintf("%sq=%s&maxRows=1&username=yrforwego", geonamesURI, location)
		retName, coord, err := c.geonameParser(qLocation)
		if err != nil {
			log.Fatalf("Failed to find location: %s\n", err)
		} else {
			loc = coord
			name = retName
		}
	}

	resp, err := c.fetch(yrURI + loc)
	if err != nil {
		log.Fatalf("Failed to fetch weather data: %v\n", err)
	}
	ret.Current, _ = c.conditionParser(resp.Properties.TimeSeries[0])
	ret.Location = fmt.Sprintf("%s", name)

	if numdays == 0 {
		return ret
	}
	ret.Forecast = c.dayParser(resp.Properties.TimeSeries, numdays)

	return ret
}

func init() {
	iface.AllBackends["yr"] = &yrConfig{}
}
