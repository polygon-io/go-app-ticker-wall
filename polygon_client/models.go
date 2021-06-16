package polygon

type APIResponse struct {
	Ticker    string `json:"ticker"`
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
}

// Aggregate is a rollup of trades for a specified time window.
type Aggregate struct {
	Ticker    string  `json:"T"`
	Timestamp int64   `json:"t"`
	Volume    float64 `json:"v"`
	Close     float64 `json:"c"`
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
}

// Trade is an executed trade reported to exchanges or TRFs.
type Trade struct {
	Price float64 `json:"p"`
}

// PreviousClose is the previous days close price. Takes into account weekends, holidays.
// This should always return a price for a ticker if it has ever traded previously.
type PreviousClose struct {
	APIResponse
	Results []*Aggregate `json:"results"`
}

// LastTrade is the last trade that has occured for this ticker.
type LastTrade struct {
	APIResponse
	Results Trade `json:"results"`
}

// CompanyDetails returns the meta data about this company.
type CompanyDetails struct {
	APIResponse
	Results Company `json:"results"`
}

// Company is the meta data about a company.
type Company struct {
	CompanyName       string `json:"name"`
	OutstandingShares int64  `json:"outstanding_shares"`
}
