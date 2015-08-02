package main

import (
	"encoding/json"
	"io/ioutil"
)

var (
	config  Config
	cfgfile string = "config.json"
)

type Config struct {
	BindAddress string `json:"bindaddress"`
	BindPort    string `json:"port"`
	CommPort    string `json:"commport"`
	Baud        int    `json:"baud"`
	Debug       bool   `json:"debug"`
	BufferSize  int    `json:"buffer"`
}

func InitialiseConfig(cfg string) (err error) {

	// read in json file
	dat, err := ioutil.ReadFile(cfg)
	if err != nil {
		return err
	}

	// convert json to config struct
	err = json.Unmarshal(dat, &config)
	if err != nil {
		return err
	}

	return nil
}
