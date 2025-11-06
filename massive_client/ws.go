package massive

import (
	"context"
	"fmt"

	massivews "github.com/massive-com/client-go/v2/websocket"
	massivews_models "github.com/massive-com/client-go/v2/websocket/models"

	"github.com/massive-com/go-app-ticker-wall/v2/models"
)

func (c *Client) ListenForTickerUpdates(ctx context.Context, tickers []string) error {
	if err := c.websocketClient.Connect(); err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}

	defer c.websocketClient.Close()

	topic := massivews.StocksSecAggs
	if c.perTickUpdates {
		topic = massivews.StocksTrades
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
			case massivews_models.EquityAgg:
				agg := msg.(massivews_models.EquityAgg)
				c.PriceUpdates <- &models.PriceUpdate{
					Ticker: agg.Symbol,
					Price:  agg.Close,
				}
			case massivews_models.EquityTrade:
				trade := msg.(massivews_models.EquityTrade)
				c.PriceUpdates <- &models.PriceUpdate{
					Ticker: trade.Symbol,
					Price:  trade.Price,
				}
			}
		}
	}
}
