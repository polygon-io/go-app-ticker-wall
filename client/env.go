package client

type Config struct {
	Leader string `split_words:"true" default:"localhost:6886"`

	// Local Presentation Settings:
	ScreenWidth  int `split_words:"true" default:"1920"`
	ScreenHeight int `split_words:"true" default:"300"`
	ScreenIndex  int `split_words:"true" default:"10"`
}
