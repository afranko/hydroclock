//go:build windows
// +build windows

package rconfig

func ReadConfig() []byte {
	appData := os.Getenv("%APPDATA%")
	path := appData + "\\alacritty\\alacritty.yml"

	b, err := ioutil.ReadFile(path)

	if err != nil {
		log.Println(err)
		return nil
	}

	return b
}
