package leader

import (
	"context"
	"fmt"
	"time"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *Leader) refreshTickerAggs(ctx context.Context) error {
	for _, ticker := range t.Tickers {

		// Each call shouldn't take more than 10sec.
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Get the agg data.
		aggs, err := t.DataClient.GetTickerTodayAggs(timeoutCtx, ticker.Ticker, 10)
		if err != nil {
			return fmt.Errorf("unable to get todays aggs for ticker: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"count":  len(aggs),
			"ticker": ticker.Ticker,
		}).Debug("Got aggregates")

		// TODO: Normalize the aggregates for a time window.
		// We want gaps in the agg bars to be filled to convery an accurate
		// representation of time.

		// This ticker actually has changes
		if len(ticker.Aggs) != len(aggs) {
			// Lock and update ticker data.
			t.Lock()
			ticker.Aggs = aggs
			t.Unlock()
			t.Updates <- &models.Update{
				UpdateType: int32(models.UpdateTypeTickerUpdate),
				Ticker:     ticker,
			}
		}

	}

	return nil
}
