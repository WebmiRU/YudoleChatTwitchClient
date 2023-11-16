package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

var Out = make(chan any, 9999)
var socket net.Conn

func configLoad() {
	configFile, _ := os.ReadFile("config.json")
	err := json.Unmarshal(configFile, &config)

	if err != nil {
		log.Println("Error while read config file", err)
		os.Exit(1)
	}
}

func connect() {
	var err error
	if socket, err = net.Dial("tcp", fmt.Sprintf("%s:%d", config.Servers.Host.Server.Address, config.Servers.Host.Server.Port)); err != nil {
		log.Println("Connection to HOST server error: ", err)
		// @TODO run reconnection procedure
	}

	for {
		msg := <-Out
		msgString, _ := json.Marshal(msg)
		_, err = fmt.Fprintln(socket, string(msgString))

		if err != nil {
			// @TODO Reconnect
			return
		}
	}
}

func main() {
	configLoad()
	go ircPingTimer()
	go ircConnect()

	connect()
}
