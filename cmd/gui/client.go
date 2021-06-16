package main

// // downloadLogo downloads the logo from our predictable S3 endpoint. This is deprecated,
// // so we will need to update this soon...
// func (t *TickerWallClient) DownloadLogo(ticker *models.Ticker) error {
// 	logrus.Debug("Downloading logo for: ", ticker.Ticker)
// 	url := "https://s3.polygon.io/logos/" + strings.ToLower(ticker.Ticker) + "/logo.png"
// 	response, e := http.Get(url)
// 	if e != nil {
// 		log.Fatal(e)
// 	}
// 	defer response.Body.Close()

// 	imgData, err := ioutil.ReadAll(response.Body)
// 	if err != nil {
// 		return fmt.Errorf("unable to download ticker logo: %w", err)
// 	}

// 	ticker.Img = int32(0)

// 	logrus.Debug("Done downloading logo for: ", ticker.Ticker)
// 	return nil
// }
