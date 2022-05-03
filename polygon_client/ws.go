package polygon

import (
	"context"
	"fmt"
	"strings"

	polygonws "github.com/polygon-io/client-go/websocket"
	polygonws_models "github.com/polygon-io/client-go/websocket/models"

	"github.com/gorilla/websocket"
	"github.com/polygon-io/go-app-ticker-wall/models"
)

//easyjson:json
type websocketTrades []websocketTrade

//easyjson:json
type websocketTrade struct {
	Event string  `json:"ev"`
	ID    string  `json:"i"`
	Price float64 `json:"p"`
	// This is a hack. We use the 'high' price if it's an aggregate because trades
	// also have a 'c' attribute, with a different type, which causes havoc.
	High   float64 `json:"h"`
	Ticker string  `json:"sym"`
}

// AddTickerToUpdates sends a subscribe message to the given tickers.
func (c *Client) AddTickerToUpdates(tickers []string) error {
	// Determine the prefix character.
	prefix := "A."
	if c.perTickUpdates {
		prefix = "T."
	}

	// Create the correct subscription parameters. ( Trades vs Aggregates ).
	subItems := make([]string, 0, len(tickers))
	for _, ticker := range tickers {
		subItems = append(subItems, prefix+ticker)
	}

	// Subscribe to updates for these items.
	subscribeString := fmt.Sprintf(`{"action":"subscribe","params":"%s"}`, strings.Join(subItems, ","))
	if err := c.wsClient.WriteMessage(websocket.TextMessage, []byte(subscribeString)); err != nil {
		return fmt.Errorf("unable to send subscription message to websockets: %w", err)
	}

	return nil
}

func (c *Client) ListenForTickerUpdates(ctx context.Context, tickers []string) error {
	if err := c.websocketClient.Connect(); err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}

	defer c.websocketClient.Close()

	topic := polygonws.StocksSecAggs
	if c.perTickUpdates {
		topic = polygonws.StocksTrades
	}

	if err := c.websocketClient.Subscribe(topic, tickers...); err != nil {
		return fmt.Errorf("subscribe websocket: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, more := <-c.websocketClient.Output():
			if !more {
				return nil
			}

			switch msg.(type) {
			case polygonws_models.EquityAgg:
				agg := msg.(polygonws_models.EquityAgg)
				c.PriceUpdates <- &models.PriceUpdate{
					Ticker: agg.Symbol,
					Price:  agg.Close,
				}
			case polygonws_models.EquityTrade:
				trade := msg.(polygonws_models.EquityTrade)
				c.PriceUpdates <- &models.PriceUpdate{
					Ticker: trade.Symbol,
					Price:  trade.Price,
				}
			}
		}
	}
}
