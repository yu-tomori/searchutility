package crowler

import (
	"github.com/sclevine/agouti"
	"net/url"
)

func CrowlAndShot(urlstring string) {
	agoutiDriver := agouti.ChromeDriver()
	agoutiDriver.Start()
	defer agoutiDriver.Stop()
	page, _ := agoutiDriver.NewPage()

	page.Navigate(urlstring)

	u, err := url.Parse(urlstring)
	if err != nil {
		panic(err)
	}

	filename := "./" + u.Host + ".png"
	if err := page.Screenshot(filename); err != nil {
		panic(err)
	}
}
