//go:build !windows
// +build !windows

package rconfig

import (
	"io/ioutil"
	"log"
	"os"
)

func ReadConfig() []byte {
	/* ---------------------------------- PATHS --------------------------------- */
	homeDir := os.Getenv("HOME")
	configDir := os.Getenv("XDG_CONFIG_HOME")

	paths := []string{
		configDir + "/hydroclock/hydroclock.yml",
		configDir + "/hydroclock.yml",
		homeDir + "/.config/hydroclock/hydroclock.yml",
		homeDir + "/.hydroclock.yml",
	}

	for _, path := range paths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println(err)
		} else {
			return b
		}
	}
	return nil
}
