package leader

import (
	"context"
	"strings"
	"sync"

	"github.com/polygon-io/go-app-ticker-wall/models"
	polygon "github.com/polygon-io/go-app-ticker-wall/polygon_client"
	"github.com/sirupsen/logrus"
	tombv2 "gopkg.in/tomb.v2"
)

// Leader manages the state.
type Leader struct {
	sync.RWMutex
	config Config

	// Client to fetch data. We should use an interface here to allow more flexibility.
	DataClient *polygon.Client

	// This keeps the presentation settings.
	PresentationSettings *models.PresentationSettings

	// Our list of tickers we want to display.
	Tickers []*models.Ticker

	// List of clients who are listening for updates.
	Clients []*UpdateClient

	// Updates is a buffered channel of generic updates to be broadcast to clients.
	// Every update added to this channel will be sent to all active clients.
	Updates chan *models.Update
}

// New creates a new ticker wall leader.
func New(cfg *Config) (*Leader, error) {
	obj := &Leader{
		config:               *cfg,
		PresentationSettings: cfg.Presentation,
		Updates:              make(chan *models.Update, 1000),
	}

	// Split out the tickers from the config.
	for _, ticker := range strings.Split(obj.config.TickerList, ",") {
		obj.Tickers = append(obj.Tickers, &models.Ticker{
			Ticker: ticker,
		})
	}

	// Create new Polygon API Client.
	obj.DataClient = polygon.NewClient(cfg.APIKey, cfg.Presentation.PerTickUpdates)

	return obj, nil
}

func (t *Leader) Run(ctx context.Context) error {
	logrus.Info("Loading ticker data..")

	if err := t.refreshTickerDetails(ctx, true); err != nil {
		return err
	}

	// Get graph data for all aggs on load.
	if err := t.refreshTickerAggs(ctx); err != nil {
		return err
	}

	logrus.Debug("All ticker data loaded..")
	logrus.Info("Ready for Clients.")

	// Create new tomb for this process.
	tomb, ctx := tombv2.WithContext(ctx)

	// Start the DataClient socket stream.
	tomb.Go(func() error {
		logrus.Debug("Starting WebSocket Listener..")
		return t.DataClient.ListenForTickerUpdates(ctx, t.getTickerSymbols())
	})

	// Listen and broadcast price updates.
	tomb.Go(func() error {
		return t.broadcastPriceUpdatesLoop(ctx)
	})

	// Broadcast updates to clients.
	tomb.Go(func() error {
		return t.clientUpdateLoop(ctx)
	})

	// Regularly get aggregates for each ticker.
	tomb.Go(func() error {
		return t.tickerAggsUpdateLoop(ctx)
	})

	// Regularly get details for each ticker.
	tomb.Go(func() error {
		return t.tickerDetailsUpdateLoop(ctx)
	})

	return tomb.Wait()
}

// getTickerSymbols returns a slice of only the ticker symbols, not the entire object.
func (t *Leader) getTickerSymbols() []string {
	tickers := make([]string, 0, len(t.Tickers))

	for _, ticker := range t.Tickers {
		tickers = append(tickers, ticker.Ticker)
	}

	return tickers
}
