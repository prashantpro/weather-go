package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "sort"
    _ "github.com/lib/pq"
)

const (
    host     = "weather-db"
    port     = 5432
    user     = "weatherman"
    password = "secr3t"
    dbname   = "weatherdb"
)

type Weather struct {
    City        string  `json:"city"`
    Temperature float64 `json:"temperature"`
}

type WeatherStats struct {
	City string  `json:"city"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Mean float64 `json:"mean"`
}

var db *sql.DB

func initDB() {
    psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)

    var err error
    db, err = sql.Open("postgres", psqlInfo)
    if err != nil {
        log.Fatal(err)
    }

    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Successfully connected to the database!")
}

// Get all weather data
func getAllWeathers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT city, temperature FROM weathers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var weathers []Weather
	for rows.Next() {
		var weather Weather
		if err := rows.Scan(&weather.City, &weather.Temperature); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		weathers = append(weathers, weather)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weathers)
}

// Get weather data by city and optional stat (min, mean, max)
func getWeatherByCity(w http.ResponseWriter, r *http.Request) {
    city := r.URL.Query().Get("name")
    stat := r.URL.Query().Get("stat")

    if city == "" {
        http.Error(w, "City parameter is required", http.StatusBadRequest)
        return
    }

    rows, err := db.Query("SELECT temperature FROM weathers WHERE city = $1", city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

    var temperatures []float64
	for rows.Next() {
		var temp float64
		if err := rows.Scan(&temp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		temperatures = append(temperatures, temp)
	}

	if len(temperatures) == 0 {
		http.Error(w, "No data found for the city", http.StatusNotFound)
		return
	}

    if stat == "" {
		// No stat provided, return all temperatures for the city
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(temperatures)
    } else {
        // Calculate the requested stat and return it
        result := calculateStat(temperatures, stat)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]float64{stat: result})
    }
}

// Calculate min, mean, max
func calculateStat(temperatures []float64, stat string) float64 {
	var min, max, sum float64
	min = temperatures[0]
	max = temperatures[0]

	for _, temp := range temperatures {
		if temp < min {
			min = temp
		}
		if temp > max {
			max = temp
		}
		sum += temp
	}

	var result float64
	switch stat {
	case "min":
		result = min
	case "max":
		result = max
	case "mean":
		result = sum / float64(len(temperatures))
	default:
		result = temperatures[0] // Return the first temperature if no valid stat is provided
	}

    // Round to one decimal place
	return math.Round(result*10) / 10
}

func getWeatherStats(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT city, temperature FROM weathers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cityTemps := make(map[string][]float64)
	for rows.Next() {
		var city string
		var temperature float64
		if err := rows.Scan(&city, &temperature); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cityTemps[city] = append(cityTemps[city], temperature)
	}

	// Prepare the stats
	stats := make([]WeatherStats, 0, len(cityTemps))
	for city, temps := range cityTemps {
		min := calculateStat(temps, "min")
		max := calculateStat(temps, "max")
		mean := calculateStat(temps, "mean")
		stats = append(stats, WeatherStats{
			City: city, Min: min, Max: max, Mean: mean,
		})
	}

    // Sort by city name
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].City < stats[j].City
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func main() {
    initDB()
    defer db.Close()

    http.HandleFunc("/weathers", getAllWeathers)
    http.HandleFunc("/weathers/city", getWeatherByCity)
    http.HandleFunc("/weathers/stats", getWeatherStats)

    fmt.Println("Starting server on :8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
