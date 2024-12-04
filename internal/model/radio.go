package model

import "time"

// Radio represents a single radio station
type Radio struct {
	Name    string `json:"name"`
	PlayURL string `json:"playUrl"`
}

// Category represents a category of radio stations
type Category struct {
	Name      string  `json:"name"`
	RadioList []Radio `json:"radioList"`
}

// PlayHistory represents a play history record
type PlayHistory struct {
	ID        int64     `json:"id"`
	RadioName string    `json:"radio_name"`
	PlayURL   string    `json:"play_url"`
	PlayedAt  time.Time `json:"played_at"`
}
