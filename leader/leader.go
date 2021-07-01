package leader

import (
	"context"
	"strings"
	"sync"

	"github.com/kelseyhightower/envconfig"
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
func New() (*Leader, error) {
	// Parse environment variables.
	var cfg Config
	if err := envconfig.Process("LEADER", &cfg); err != nil {
		return nil, err
	}

	obj := &Leader{
		config: cfg,
		PresentationSettings: &models.PresentationSettings{
			AnimationDurationMS: int32(cfg.Presentation.AnimationDuration),
			TickerBoxWidth:      int32(cfg.Presentation.TickerBoxWidthPx),
			ScrollSpeed:         int32(cfg.Presentation.ScrollSpeed),
			ShowLogos:           cfg.Presentation.ShowLogos,
			UpColor:             constructRGBA(cfg.Presentation.UpColor),
			DownColor:           constructRGBA(cfg.Presentation.DownColor),
			TickerBoxBGColor:    constructRGBA(cfg.Presentation.TickerBoxBGColor),
			BGColor:             constructRGBA(cfg.Presentation.BGColor),
			FontColor:           constructRGBA(cfg.Presentation.FontColor),
		},
		Updates: make(chan *models.Update, 1000),
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
	logrus.Debug("Loading ticker data..")

	for i, ticker := range t.Tickers {
		// Make sure context hasn't closed on us.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		newTickerObj, err := t.DataClient.LoadTickerData(ctx, ticker.Ticker)
		if err != nil {
			return err
		}

		t.Tickers[i] = newTickerObj
	}

	logrus.Info("All ticker data loaded..")

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
