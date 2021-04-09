package manager

import (
	"math"
	"sort"
	"sync"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// TickerManager is the interface used for ticker managers.
type TickerManager interface {
	// AddTicker adds a ticker symbol to this manager. It will be added at the correct index ( sorted alphabetically )
	AddTicker(ticker string, price float64, priceChangePercentage float64, companyName string)

	// UpdateTicker updates a ticker which already exists in our list. If it does not exist, nothing will happen.
	UpdateTicker(*models.Ticker) error

	// DetermineTickersForRender takes a global offset and returns the ticker indices which should be rendered.
	DetermineTickersForRender(globalOffset int64) (tickers []*Ticker)

	// TickerOffset determines what the offset should be for this ticker, on this screen.
	TickerOffset(globalOffset int64, ticker *Ticker) int64

	// Presentation data getter / setter.
	GetPresentationData() *PresentationData
	SetPresentationData(*PresentationData)
}

type PresentationData struct {
	ScreenGlobalOffset int64 // ScreenGlobalOffset is the offset of this screen in the entire network. Eg: If there are 2 monitors before this one, each 500px. Our screen global offset will be 1000px.
	ScreenWidth        int   // ScreenWidth is the width of the entire applications screen ( px ).
	ScreenHeight       int   // ScreenHeight is the height of the entire applications screen ( px ).
	TickerBoxWidth     int   // TickerBoxWidth is the total px width we allocate per ticker.
	ScreenIndex        int   // ScreenIndex is index of this screen in the sequence of displays. ( left to right ) ( starting at 1 ).
	NumberOfScreens    int   // NumberOfScreens is the total number of screens in this ticker wall cluster.
	GlobalViewportSize int64 // GlobalViewportSize is the total size of all screens combined ( px ).
}

// DefaultManager is used to manage all tickers for an application. It knows important information for rendering.
type DefaultManager struct {
	sync.RWMutex

	// Constant presentation data:
	PresentationData *PresentationData

	// Our list of tickers.
	Tickers []*Ticker // Tickers is the entire list of tickers. We use a map for faster updates.
}

// NewDefaultManager creates a new default ticker manager.
func NewDefaultManager(presentationData *PresentationData) TickerManager {
	mgr := &DefaultManager{
		PresentationData: presentationData,
	}
	return mgr
}

// GetPresentationData gets the presentation data.
func (m *DefaultManager) GetPresentationData() *PresentationData {
	m.RLock()
	defer m.RUnlock()
	return m.PresentationData
}

// SetPresentationData gets the presentation data.
func (m *DefaultManager) SetPresentationData(presentationData *PresentationData) {
	m.Lock()
	defer m.Unlock()
	m.PresentationData = presentationData
}

// TickerOffset determines what the offset should be for this ticker, on this screen.
func (m *DefaultManager) TickerOffset(globalOffset int64, ticker *Ticker) int64 {
	localizedOffset := (globalOffset % int64(len(m.Tickers)*m.PresentationData.TickerBoxWidth))

	offset := (int64(int(ticker.Index)*m.PresentationData.TickerBoxWidth) - localizedOffset) - int64(m.PresentationData.ScreenGlobalOffset)
	// Too far left, probably need to wrap it around.
	if offset < 0 {
		if offset < -(int64(m.PresentationData.TickerBoxWidth)) {
			offset = int64(len(m.Tickers)*m.PresentationData.TickerBoxWidth) - int64(math.Abs(float64(offset)))
		}
	}
	return offset
}

// DetermineTickersForRender takes a global offset and returns the ticker indices which should be rendered.
func (m *DefaultManager) DetermineTickersForRender(globalOffset int64) []*Ticker {
	var tickers []*Ticker

	// Global offset does not necessarily ever reset, so we need to get the localized offset.
	localizedOffset := (globalOffset % int64(len(m.Tickers)*m.PresentationData.TickerBoxWidth)) + m.PresentationData.ScreenGlobalOffset
	logrus.Trace("Localized Offset: ", localizedOffset)

	firstIndex := int(math.Floor(float64(localizedOffset) / float64(m.PresentationData.TickerBoxWidth)))
	lastIndex := int(math.Floor(float64(localizedOffset+int64(m.PresentationData.ScreenWidth)) / float64(m.PresentationData.TickerBoxWidth)))

	logrus.Trace("offsets: ", firstIndex, lastIndex)

	// eg: -2
	if firstIndex < 0 {
		boundedFirst := int(float64(len(m.Tickers)) - math.Abs(float64(firstIndex)))
		logrus.Trace("first index short: ", boundedFirst)
		tickers = append(tickers, m.Tickers[boundedFirst:]...)
		// Now we set first index to 0 since we have the overflow items.
		firstIndex = 0
	}

	if firstIndex > len(m.Tickers) {
		// logrus.Info("Invalid slice bounds - alertttt ")
		firstIndex = 0
	}

	// If our end index is outside of the bounds.
	boundedLastIndex := lastIndex
	if lastIndex+1 > len(m.Tickers) {
		boundedLastIndex = len(m.Tickers) - 1
		logrus.Trace("last index long: ", boundedLastIndex)
	}

	// Add our valid section.
	tickers = append(tickers, m.Tickers[firstIndex:boundedLastIndex+1]...)

	// If we have overflow, now add those.
	if lastIndex+1 > len(m.Tickers) {
		boundedLast := lastIndex - len(m.Tickers)
		logrus.Trace("last index long2: ", boundedLast)
		tickers = append(tickers, m.Tickers[:boundedLast+1]...)
	}

	return tickers
}

// UpdateTicker updates a ticker which already exists in our list. If it does not exist, nothing will happen.
func (m *DefaultManager) UpdateTicker(ticker *models.Ticker) error {
	m.RLock()
	defer m.RUnlock()

	for _, t := range m.Tickers {
		if t.Ticker.Ticker == ticker.Ticker {
			t.Lock()
			t.Price = ticker.Price
			t.Unlock()
		}
	}

	return nil
}

// AddTicker creates a new ticker in this manager. Tickers should be unique by their ticker symbol.
func (m *DefaultManager) AddTicker(ticker string, price float64, priceChangePercentage float64, companyName string) {
	m.Lock()
	defer m.Unlock()

	// Create a new ticker.
	tickerObj := &Ticker{
		sync.RWMutex{},
		models.Ticker{
			Ticker:                ticker,
			Price:                 price,
			PriceChangePercentage: priceChangePercentage,
			CompanyName:           companyName,
		},
	}

	// Add ticker to our list.
	m.Tickers = append(m.Tickers, tickerObj)

	// Sort our tickers now we have a new one.
	sort.Sort(TickerSlice(m.Tickers))

	// Update everyones index, since this could potentially change it.
	for i, ticker := range m.Tickers {
		ticker.Index = int32(i)
	}
}
