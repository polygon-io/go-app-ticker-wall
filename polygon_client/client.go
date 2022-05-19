package polygon

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	polygon "github.com/polygon-io/client-go/rest"
	polygon_models "github.com/polygon-io/client-go/rest/models"
	polygonws "github.com/polygon-io/client-go/websocket"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// We SERIOUSLY need our own Go library... wtf lol
// This library is awful and is a stop gap until we have a client library.
// It also reaches across modules, and does other bad things.
// Shame... Shame... Shame...

// bufferedChannelSize defines how many items we buffer internally before we start blocking.
const bufferedChannelSize = 10_000

// company is the metadata about a company.
type company struct {
	CompanyName       string `json:"name"`
	OutstandingShares int64  `json:"outstanding_shares"`
}

type Client struct {
	PriceUpdates   chan *models.PriceUpdate
	perTickUpdates bool
	wsClient       *websocket.Conn

	restClient      *polygon.Client
	websocketClient *polygonws.Client
}

// NewClient creates a new polygon API client.
func NewClient(apiKey string, perTickUpdate bool) (*Client, error) {
	wsclient, err := polygonws.New(polygonws.Config{
		APIKey: apiKey,
		Feed:   polygonws.RealTime,
		Market: polygonws.Stocks,
	})

	if err != nil {
		return nil, err
	}

	return &Client{
		PriceUpdates:    make(chan *models.PriceUpdate, bufferedChannelSize),
		perTickUpdates:  perTickUpdate,
		restClient:      polygon.New(apiKey),
		websocketClient: wsclient,
	}, nil
}

func (c *Client) LoadTickerData(ctx context.Context, tickerSymbol string) (*models.Ticker, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	logrus.WithField("ticker", tickerSymbol).Debug("Loading ticker data..")
	ticker := &models.Ticker{
		Ticker: tickerSymbol,
	}

	// Get Yesterdays Price
	previousClosePrice, err := c.GetTickerYesterdaysClose(ctx, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.PreviousClosePrice = previousClosePrice

	// Get Current Price
	currentPrice, err := c.GetTickerCurrentPrice(ctx, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.Price = currentPrice

	// Get company Info
	companyInfo, err := c.GetCompanyDetails(ctx, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.CompanyName = companyInfo.CompanyName
	ticker.OutstandingShares = companyInfo.OutstandingShares

	return ticker, nil
}

func (c *Client) GetTickerTodayAggs(ctx context.Context, t time.Time, ticker string, rangeSize int) ([]*models.Agg, error) {
	loc, _ := time.LoadLocation("America/New_York")

	// Start at 9am instead of 930am because sometimes pre market is significant to the charts.
	openTime := time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, loc)
	closeTime := time.Date(t.Year(), t.Month(), t.Day(), 16, 30, 0, 0, loc)

	aggsParams := polygon_models.GetAggsParams{
		Ticker:     ticker,
		Multiplier: rangeSize,
		Timespan:   polygon_models.Minute,
		From:       polygon_models.Millis(openTime),
		To:         polygon_models.Millis(closeTime),
	}.WithLimit(int(closeTime.Sub(openTime) / time.Minute))

	resp, err := c.restClient.GetAggs(ctx, aggsParams)
	if err != nil {
		return nil, err
	}

	// Transform our polygon.io aggregates into the "model" aggregates.
	results := make([]*models.Agg, 0, len(resp.Results))
	for _, agg := range resp.Results {
		results = append(results, &models.Agg{
			Price:     agg.Close,
			Volume:    int32(agg.Volume),
			Timestamp: time.Time(agg.Timestamp).UnixMilli(),
		})
	}

	return results, nil
}

func (c *Client) GetTickerCurrentPrice(ctx context.Context, ticker string) (float64, error) {
	resp, err := c.restClient.GetLastTrade(ctx, &polygon_models.GetLastTradeParams{Ticker: ticker})
	if err != nil {
		return 0, err
	}

	return resp.Results.Price, nil
}

func (c *Client) GetCompanyDetails(ctx context.Context, ticker string) (*company, error) {
	resp, err := c.restClient.GetTickerDetails(ctx, &polygon_models.GetTickerDetailsParams{Ticker: ticker})
	if err != nil {
		return nil, err
	}

	return &company{
		CompanyName:       resp.Results.Name,
		OutstandingShares: resp.Results.WeightedSharesOutstanding,
	}, nil
}

// GetTickerYesterdaysClose is the previous days close price. Takes into account weekends, holidays.
// This should always return a price for a ticker if it has ever traded previously.
func (c *Client) GetTickerYesterdaysClose(ctx context.Context, ticker string) (float64, error) {
	resp, err := c.restClient.AggsClient.GetPreviousCloseAgg(ctx, &polygon_models.GetPreviousCloseAggParams{Ticker: ticker})
	if err != nil {
		return 0, err
	}

	if len(resp.Results) < 1 {
		return 0, nil
	}

	return resp.Results[0].Close, nil
}
