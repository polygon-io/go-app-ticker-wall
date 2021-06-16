package client

import "github.com/polygon-io/go-app-ticker-wall/models"

// GetTickers returns all the tickers we have.
func (t *Client) GetTickers() []*models.Ticker {
	t.RLock()
	defer t.RUnlock()

	return t.Tickers
}
