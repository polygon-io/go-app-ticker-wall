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

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string `split_words:"true" default:"AAPL,AMD,NVDA"`
	// TickerList string `split_words:"true" default:"AAPL,AMD,NVDA,FB,NFLX,LPL,AMZN,SNAP,NKE,SBUX,SQ,INTC,IBM"`
	APIKey string `split_words:"true" required:"true"` // polygon.io API key.

	// Presentation Default Settings
	Presentation struct {
		TickerBoxWidthPx  int    `split_words:"true" default:"1300"`
		ScrollSpeed       int    `split_words:"true" default:"16"`
		AnimationDuration int    `split_words:"true" default:"500"`
		UpColor           string `split_words:"true" default:"TBI"`
		DownColor         string `split_words:"true" default:"TBI"`
		BGColor           string `split_words:"true" default:"TBI"`
		ShowLogos         bool   `split_words:"true" default:"true"`
	}
}

// UpdateClient is a generic wrapper which is used for all clients which are requesting
// updates be sent to them.
type UpdateClient struct {
	UUID    string
	Screen  *models.Screen
	Updates chan *models.Update
	Stream  models.Leader_JoinClusterServer
}

// New creates a new ticker wall leader.
func New() (*Leader, error) {
	// Parse environment variables.
	var cfg Config
	err := envconfig.Process("LEADER", &cfg)
	if err != nil {
		return nil, err
	}

	obj := &Leader{
		config: cfg,
		PresentationSettings: &models.PresentationSettings{
			TickerBoxWidth:      int32(cfg.Presentation.TickerBoxWidthPx),
			ScrollSpeed:         int32(cfg.Presentation.ScrollSpeed),
			UpColor:             cfg.Presentation.UpColor,
			DownColor:           cfg.Presentation.DownColor,
			BGColor:             cfg.Presentation.BGColor,
			ShowLogos:           cfg.Presentation.ShowLogos,
			AnimationDurationMS: int32(cfg.Presentation.AnimationDuration),
		},
	}

	// Split out the tickers from the config.
	for _, ticker := range strings.Split(obj.config.TickerList, ",") {
		obj.Tickers = append(obj.Tickers, &models.Ticker{
			Ticker: ticker,
		})
	}

	// Create new Polygon API Client.
	obj.DataClient = polygon.NewClient(cfg.APIKey)

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
		return t.DataClient.ListenForTickerUpdates(ctx, t.getTickerSymbols())
	})

	// Listen and broadcast price updates.
	tomb.Go(func() error {
		return t.broadcastPriceUpdates(ctx)
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

// broadcastPriceUpdates listens to updates from the DataClient and sends that to all gRPC clients.
func (t *Leader) broadcastPriceUpdates(ctx context.Context) error {
	// Read from DataClient price updates channel onto our update channel.
	for priceUpdate := range t.DataClient.PriceUpdates {
		t.Updates <- &models.Update{
			UpdateType:  int32(models.UpdateTypePrice),
			PriceUpdate: priceUpdate,
		}
	}

	return nil
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
