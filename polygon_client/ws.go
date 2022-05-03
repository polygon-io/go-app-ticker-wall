package polygon

import (
	"context"
	"fmt"

	polygonws "github.com/polygon-io/client-go/websocket"
	polygonws_models "github.com/polygon-io/client-go/websocket/models"

	"github.com/polygon-io/go-app-ticker-wall/models"
)

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
