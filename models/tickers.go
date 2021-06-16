package models

// TickerSlice is sortable by Ticker.
type TickerSlice []*Ticker

func (a TickerSlice) Len() int           { return len(a) }
func (a TickerSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TickerSlice) Less(i, j int) bool { return a[i].Ticker < a[j].Ticker }
