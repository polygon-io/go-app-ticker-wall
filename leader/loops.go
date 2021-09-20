package leader

import (
	"context"
	"time"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// broadcastPriceUpdatesLoop listens to updates from the DataClient and sends that to all gRPC clients.
func (t *Leader) broadcastPriceUpdatesLoop(ctx context.Context) error {
	// Read from DataClient price updates channel onto our update channel.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case priceUpdate := <-t.DataClient.PriceUpdates:
			t.Updates <- &models.Update{
				UpdateType:  int32(models.UpdateTypePrice),
				PriceUpdate: priceUpdate,
			}
		}
	}
}

// clientUpdateLoop spins until we have an update, which is then queued up for all existing clients.
func (t *Leader) clientUpdateLoop(ctx context.Context) error {
	defer close(t.Updates)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-t.Updates:
			t.RLock()

			// Put this update on the clients queue.
			for _, client := range t.Clients {
				client.Updates <- update
			}

			t.RUnlock()
		}
	}
}

// tickerAggsUpdateLoop continually updates each tickers aggregates.
func (t *Leader) tickerAggsUpdateLoop(ctx context.Context) error {
	timer1 := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer1.C:
			if err := t.refreshTickerAggs(ctx); err != nil {
				logrus.WithError(err).Error("Unable to update ticker aggs.")
				// We probably don't want to completely exit if ever 1 API call fails.
				// return err
			}
		}
	}
}

// tickerDetailsUpdateLoop continually updates each tickers details.
func (t *Leader) tickerDetailsUpdateLoop(ctx context.Context) error {
	timer1 := time.NewTicker(500 * time.Second) // every 5min
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer1.C:
			if err := t.refreshTickerDetails(ctx, false); err != nil {
				logrus.WithError(err).Error("Unable to update ticker details.")
				// We probably don't want to completely exit if ever 1 API call fails.
				// return err
			}
		}
	}
}
