// Package domain holds data types shared across application features and adapters.
package domain

type Incident struct {
	VehicleType string
	Brand       string
	Location    string
	Severity    int
	LocalHour   *int
}

type Score struct {
	Username string  `json:"username"`
	Distance float64 `json:"distance"`
}

type Ranking struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type HourlyCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

type Statistics struct {
	Period        string        `json:"period"`
	Total         int           `json:"total"`
	Today         int           `json:"today"`
	FilteredCount int           `json:"filtered_count"`
	Brands        []Ranking     `json:"brands"`
	Vehicles      []Ranking     `json:"vehicles"`
	LocalHours    []HourlyCount `json:"local_hours"`
}
