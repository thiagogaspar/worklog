package types

import "time"

type Entry struct {
	Ticket    string    `json:"ticket"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Minutes   int       `json:"minutes"`
	Desc      string    `json:"desc,omitempty"`
}

type Day struct {
	Date    string  `json:"date"` // "2026-07-07"
	Entries []Entry `json:"entries"`
	Total   float64 `json:"total"` // horas
}

type Week struct {
	YearWeek string `json:"year_week"` // "2026-W28"
	Days     []Day  `json:"days"`
	Planned  float64 `json:"planned"` // horas planejadas na semana (ex: 40)
}

type ActiveSession struct {
	Ticket string    `json:"ticket"`
	Since  time.Time `json:"since"`
	Desc   string    `json:"desc,omitempty"`
}

type Store struct {
	Week        Week           `json:"week"`
	Active      *ActiveSession `json:"active,omitempty"`
}
