package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/nanovgo"
	"github.com/sirupsen/logrus"
)

type LogoManager struct {
	sync.RWMutex
	logoMap map[string]*Logo
	nanoCtx *nanovgo.Context
	// NeedsRenderAccess is a flag we set when we need access to the main render thread.
	// We cannot load images into context unless it's on the main render thread.
	NeedsRenderAccess bool
}

func NewLogosManager() *LogoManager {
	return &LogoManager{
		logoMap: make(map[string]*Logo),
	}
}

type Logo struct {
	Status     logoStatus
	NanovImgID int
	// tempImgData holds the images data until we can load it into render context.
	tempImgData []byte
}

type logoStatus int

const (
	logoStatusMissing     logoStatus = 0
	logoStatusDownloading logoStatus = 1
	logoStatusError       logoStatus = 2
	logoStatusOK          logoStatus = 3
	// logoStatusReadyToLoad is used when the ticker has the logo loaded into 'tempImgData' and
	// is ready to load it into render context.
	logoStatusReadyToLoad logoStatus = 4
)

func (l *LogoManager) DownloadLogo(ticker *models.Ticker) error {
	logrus.Debug("Downloading logo for: ", ticker.Ticker)
	tickerLogo, ok := l.logoMap[ticker.Ticker]
	// This should always exist before we get here, but just to make sure...
	if !ok {
		l.Lock()
		defer l.Unlock()
		l.logoMap[ticker.Ticker] = &Logo{
			Status: logoStatusDownloading,
		}
		// Restart the download now it exists...
		return l.DownloadLogo(ticker)
	}

	// Download URL for logos. ( Deprecated, this will not work for newer ticker symbols ).
	url := "https://s3.polygon.io/logos/" + strings.ToLower(ticker.Ticker) + "/logo.png"
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Read the logo bytes into memory.
	imgBuff, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// Create a render context reference to the image.
	l.Lock()
	defer l.Unlock()

	tickerLogo.tempImgData = imgBuff
	tickerLogo.Status = logoStatusReadyToLoad

	// Signal we are ready to load.
	l.NeedsRenderAccess = true

	// tickerLogo.NanovImgID = l.nanoCtx.CreateImageFromMemory(0, imgBuff)

	logrus.Debug("Done downloading logo for: ", ticker.Ticker)
	return nil
}

// RenderThread is called in the main rendering thread ( so be fast ). This is required to change
// the context. Without being in the main thread it will cause panics.
func (l *LogoManager) RenderThread() {
	l.RLock()
	needsThreadAccess := l.NeedsRenderAccess
	l.RUnlock()

	// We do not need access
	if !needsThreadAccess {
		return
	}

	l.Lock()
	defer l.Unlock()

	for _, tickerLogo := range l.logoMap {
		if tickerLogo.Status == logoStatusReadyToLoad {
			tickerLogo.NanovImgID = l.nanoCtx.CreateImageFromMemory(0, tickerLogo.tempImgData)
			tickerLogo.tempImgData = nil
			tickerLogo.Status = logoStatusOK
		}
	}

	l.NeedsRenderAccess = false
}

// GetTickerImage attempts to get the tickers logo. If it does not exist it will start the
// download process and return a placeholder image instead.
func (l *LogoManager) GetTickerImage(ticker *models.Ticker) *Logo {
	l.RLock()
	tickerLogo, ok := l.logoMap[ticker.Ticker]
	l.RUnlock()

	// No logo exists for this ticker.
	if !ok {
		l.Lock()
		defer l.Unlock()

		l.logoMap[ticker.Ticker] = &Logo{
			Status: logoStatusDownloading,
		}

		// Start the actual download in new go routine.
		go l.DownloadLogo(ticker)

		return nil
	}

	// Logo exists and is ready.
	if tickerLogo.Status == logoStatusOK {
		return tickerLogo
	}

	if tickerLogo.Status == logoStatusDownloading {
		// Return download image.

	}

	if tickerLogo.Status == logoStatusError {
		// Return error image.

	}
	return nil
}

func (l *LogoManager) Setup(nanoCtx *nanovgo.Context) error {
	// Load in some default images to context.
	logrus.Debug("Setup the logo manager completed.")
	l.nanoCtx = nanoCtx
	return nil
}

// renderTickerLogo renders the tickers logo at the given offset & size.
func (g *GUI) renderTickerLogo(offset, logoSize float32, ticker *models.Ticker) error {
	tickerImg := g.logos.GetTickerImage(ticker)
	if tickerImg == nil {
		return nil
	}

	// Paint the logo
	imgPaint := nanovgo.ImagePattern(offset, 182.5, logoSize, logoSize, 0.0/180.0*nanovgo.PI, int(tickerImg.NanovImgID), 1)
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(offset, 182.5, logoSize, logoSize, 5)
	g.nanoCtx.SetFillPaint(imgPaint)
	g.nanoCtx.Fill()

	return nil
}
