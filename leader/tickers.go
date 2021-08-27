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
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Each call shouldn't take more than 10sec.
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Get the agg data.
		today := getCurrentOrPreviousWeekday(time.Now())
		aggs, err := t.DataClient.GetTickerTodayAggs(timeoutCtx, today, ticker.Ticker, 10)
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

func getCurrentOrPreviousWeekday(today time.Time) time.Time {
	weekday := today.Weekday()
	if weekday == time.Sunday || weekday == time.Saturday {
		// Go back a day
		today = today.AddDate(0, 0, -1)
		return getCurrentOrPreviousWeekday(today)
	}
	return today
}

func (t *Leader) refreshTickerDetails(ctx context.Context, firstRun bool) error {
	for _, ticker := range t.Tickers {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Each call shouldn't take more than 10sec.
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Get details.
		tickerDetails, err := t.DataClient.LoadTickerData(timeoutCtx, ticker.Ticker)
		if err != nil {
			return err
		}

		// Update with details
		t.Lock()
		ticker.CompanyName = tickerDetails.CompanyName
		ticker.PreviousClosePrice = tickerDetails.PreviousClosePrice
		ticker.OutstandingShares = tickerDetails.OutstandingShares
		if firstRun {
			ticker.Price = tickerDetails.Price
		}
		t.Unlock()
		t.Updates <- &models.Update{
			UpdateType: int32(models.UpdateTypeTickerUpdate),
			Ticker:     ticker,
		}
	}
	return nil
}
