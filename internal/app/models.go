package app

type Service struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageURL    string   `json:"imageUrl"`
	Props       []string `json:"props"`
}

type CartItem struct {
	ServiceID int `json:"serviceId"`
	Quantity  int `json:"quantity"`
}

// Calculation result (used for API and SSR)
type Result struct {
	ServiceID   int     `json:"serviceId"`
	ServiceName string  `json:"serviceName"`
	NaturalHz   float64 `json:"naturalHz"`
	Isolation   float64 `json:"isolationPercent"`
}
