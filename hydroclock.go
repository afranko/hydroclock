package main

import (
	"fmt"
	"hydroclock/rconfig"
	"io/ioutil"
	"log"
	"math"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"gopkg.in/yaml.v2"
)

/* ------------------------------ SEX CONSTANTS ----------------------------- */
// Men need 3.7l per day, while women need 2.7l -> if you sleep 8 hours a day
// it means that you have to drink 230ml/h (M) or 170ml/h (F)
const menHourlyWater = 230   // 230ml/h
const womenHourlyWater = 170 // 170ml/h

/* ---------------------------- CONFIG CONSTANTS ---------------------------- */

const notificationFreq = 20 // 20 minutes
const notificationType = "light"

const defaultVolume = 230 // 230 ml

/* ---------------------------- MESSAGE CONSTANTS --------------------------- */
const appName = "HydroClock"
const drinkMessage = "Stay hydrated and drink water! (~%.2f cup)"
const refillMessage = "Stay hydrated! Finish your glass and refill it!"

/* ------------------------------- CONFIG TYPE ------------------------------ */
type Notification struct {
	Freq int    `yaml:"intervals"`
	Type string `yaml:"type"`
}

type Config struct {
	Notification Notification
	Vol          int    `yaml:"volume"`
	Sex          string `yaml:"sex"`
}

const minimumSupportedVolume = 230 // 230 ml

var validFreqs = map[int]bool{60: true, 30: true, 20: true, 15: true}
var validTypes = map[string]bool{"light": true, "hard": true}
var validSex = map[string]bool{"female": true, "male": true}

/* -------------------------------------------------------------------------- */
/*                                 MAIN BLOCK                                 */
/* -------------------------------------------------------------------------- */
func main() {
	/* --------------------------------- CONFIG --------------------------------- */
	config := Config{
		Notification{
			notificationFreq,
			notificationType,
		},
		defaultVolume,
		"none",
	}

	// READ CONFIG
	c := rconfig.ReadConfig()
	if c != nil {
		err := yaml.Unmarshal(c, &config)
		if err != nil {
			log.Fatal(err)
		}
		// VALIDITY CHECKS
		if !validFreqs[config.Notification.Freq] {
			log.Fatal("Notification frequency value is not supported!")
		}
		if !validTypes[config.Notification.Type] {
			log.Fatal("Notification type value is not supported!")
		}
		if !validSex[config.Sex] {
			log.Fatal("Sex value is not supported!")
		}
		if config.Vol < minimumSupportedVolume {
			log.Fatal("Volume value is not supported!")
		}
	}

	appIcon, err := ioutil.ReadFile("hydroclock_icon.png") // TODO loc
	if err != nil {
		log.Println(err)
	}

	/* ---------------------------------- INIT ---------------------------------- */
	var hourlyWater int
	switch config.Sex {
	case "female":
		hourlyWater = womenHourlyWater
	case "male":
		hourlyWater = menHourlyWater
	default:
		hourlyWater = (womenHourlyWater + menHourlyWater) / 2
	}

	drinksPerGlass := (float64(config.Vol) / float64(hourlyWater)) * 60.0 / float64(config.Notification.Freq)
	volumePerDrink := float64(config.Vol) / drinksPerGlass
	drinkPerCup := math.Round((volumePerDrink/2/float64(config.Vol))/0.05) * 0.05

	fDrinkMessage := fmt.Sprintf(drinkMessage, drinkPerCup*2)

	/* ------------------------------ MAIN CLOSURE ------------------------------ */
	turnOff := make(chan bool)

	mainRoutine := func() {
		for {
			mainTimer := time.NewTimer(time.Duration(config.Notification.Freq) * time.Second)

			for i := 0.0; i < drinksPerGlass-1; i++ {
				select {
				case <-turnOff:
					return // STOP goroutine
				case <-mainTimer.C:
					// Continue
				}

				drinkNotify(config.Notification.Type, appName, fDrinkMessage, "hydroclock_icon.png")
				mainTimer = time.NewTimer(time.Duration(config.Notification.Freq) * time.Minute)
			}

			// Last or almost last notification
			select {
			case <-turnOff:
				return // STOP goroutine
			case <-mainTimer.C:
				// Continue
			}

			_, frac := math.Modf(drinksPerGlass)
			remainingVol := frac * volumePerDrink

			if remainingVol < 0.25*float64(hourlyWater) {
				drinkNotify(config.Notification.Type, appName, refillMessage, "hydroclock_icon.png")
			} else if remainingVol < 0.8*float64(hourlyWater) {
				drinkNotify(config.Notification.Type, appName, fDrinkMessage, "hydroclock_icon.png")

				mainTimer = time.NewTimer(time.Duration(config.Notification.Freq) / 2 * time.Minute)

				select {
				case <-turnOff:
					return // STOP goroutine
				case <-mainTimer.C:
					// Continue
				}

				drinkNotify(config.Notification.Type, appName, refillMessage, "hydroclock_icon.png")
			} else {
				mainTimer = time.NewTimer(time.Duration(config.Notification.Freq) * time.Minute)
				select {
				case <-turnOff:
					return // STOP goroutine
				case <-mainTimer.C:
					// Continue
				}

				drinkNotify(config.Notification.Type, appName, refillMessage, "hydroclock_icon.png")
			}
		}
	}

	/* ----------------------------- SYSTRAY CONFIG ----------------------------- */
	systray.SetIcon(appIcon)
	systray.SetTitle("HydroClock")

	/* -------------------------------------------------------------------------- */
	/*                                MENU ELEMENTS                               */
	/* -------------------------------------------------------------------------- */
	// ENABLE/DISABLE
	mEnabled := systray.AddMenuItemCheckbox("Enabled", "Enable/Disable Hydroclock", true)
	go func() {
		for {
			<-mEnabled.ClickedCh
			if mEnabled.Checked() {
				mEnabled.Uncheck()
				turnOff <- true
			} else {
				mEnabled.Check()
				go mainRoutine()
			}
		}
	}()

	// REFILL
	mRefill := systray.AddMenuItem("Refill", "I refilled my bottle!")
	go func() {
		for {
			<-mRefill.ClickedCh
			turnOff <- true
			go mainRoutine()
		}
	}()

	// QUIT
	mQuit := systray.AddMenuItem("Quit", "Quit HydroClock")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	systray.Run(func() { go mainRoutine() }, func() {})
}

func drinkNotify(nType string, name string, message string, icon string) {
	switch nType {
	case "light":
		err := beeep.Alert(name, message, icon)
		if err != nil {
			log.Println(err)
		}
	case "hard":
		go func() {
			_, err := dlgs.MessageBox(name, message)
			if err != nil {
				log.Println(err)
			}
		}()
	}
}
