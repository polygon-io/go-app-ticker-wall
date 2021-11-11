package leader

import "github.com/polygon-io/go-app-ticker-wall/models"

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string
	APIKey     string

	// Presentation Default Settings
	Presentation *models.PresentationSettings
}
