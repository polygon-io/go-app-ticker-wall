package leader

import (
	"context"

	"github.com/polygon-io/go-app-ticker-wall/models"
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
			t.Lock()

			// Put this update on the clients queue.
			for _, client := range t.Clients {
				client.Updates <- update
			}

			t.Unlock()
		}
	}
}
