package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	IrcSocketPing = 20
	IrcSocketWait = 30
)

var re = regexp.MustCompile(`^(?:@([^\r\n ]*) +|())(?::([^\r\n ]+) +|())([^\r\n ]+)(?: +([^:\r\n ]+[^\r\n ]*(?: +[^:\r\n ]+[^\r\n ]*)*)|())?(?: +:([^\r\n]*)| +())?[\r\n]*$`)
var ircSocket net.Conn
var config Config
var ircIsConnected bool
var ircLastMessageReceived time.Time

func ircPingTimer() {
	for {
		if ircIsConnected && time.Now().Sub(ircLastMessageReceived).Seconds() > IrcSocketPing {
			log.Printf("Send PING request to IRC server %s:%d\n", config.Servers.Twitch.Address, config.Servers.Twitch.Port)
			fmt.Fprintln(ircSocket, "PING :tmi.twitch.tv")
		}

		time.Sleep(time.Second)
	}
}

func ircConnect() {
	log.Printf("Connecting to Twitch server IRC server %s:%d\n", config.Servers.Twitch.Address, config.Servers.Twitch.Port)

	var err error
	if ircSocket, err = net.Dial("tcp", fmt.Sprintf("%s:%d", config.Servers.Twitch.Address, config.Servers.Twitch.Port)); err != nil {
		log.Println("Service TWITCH connection error: ", err)
		// @TODO run reconnection procedure
	}

	ircSocket.SetReadDeadline(time.Now().Add(time.Second * IrcSocketWait))
	ircIsConnected = true

	fmt.Fprintln(ircSocket, "CAP REQ :twitch.tv/commands twitch.tv/tags twitch.tv/membership")
	fmt.Fprintln(ircSocket, fmt.Sprintf("PASS %s", config.Servers.Twitch.Password))
	fmt.Fprintln(ircSocket, fmt.Sprintf("NICK %s", config.Servers.Twitch.Login))
	fmt.Fprintln(ircSocket, fmt.Sprintf("JOIN #%s", "ewolf34"))
	fmt.Fprintln(ircSocket, fmt.Sprintf("JOIN #%s", "megadriver_16"))

	scanner := bufio.NewScanner(bufio.NewReader(ircSocket))
	for scanner.Scan() {
		ircLastMessageReceived = time.Now()

		ircSocket.SetReadDeadline(time.Now().Add(time.Second * IrcSocketWait))
		msg := message(scanner.Text())

		switch strings.ToLower(msg.Type) {
		case "001": // Welcome message
			//fmt.Fprintln(ircSocket, fmt.Sprintf("JOIN #%s", channel))
			break

		case "join":
			//if strings.EqualFold(msg.Login, login) {}
			break

		case "part":

			break

		case "privmsg":
			Out <- TypeMessage{
				Id:      "",
				Type:    "chat/message",
				Service: "twitch",
				Html:    smiles(msg),
				Text:    msg.Text,
				User: TypeUser{
					Id:       "",
					Nickname: msg.Nickname,
					Login:    msg.Login,
					Meta: TypeMeta{
						Badges: nil,
					},
				},
			}
			break

		case "ping":
			fmt.Fprintln(ircSocket, "PONG :"+msg.Text)
			break

		case "pong":
			break

		case "roomstate":
			break

		case "userstate":
			break

		case "globaluserstate":
			break

		case "002", "003", "004", "353", "366", "372", "375", "376", "cap":
			// Ignore this message types
			break

		default:
			log.Println("Service TWITCH unknown ircSocket message type:", msg.Type)
		}
	}

	ircIsConnected = false
	log.Printf("Disconnected from Twitch server IRC server %s:%d\n", config.Servers.Twitch.Address, config.Servers.Twitch.Port)
}

func tags(tags string) map[string]string {
	result := make(map[string]string)

	if len(tags) == 0 {
		return result
	}

	for _, v := range strings.Split(tags, ";") {
		kv := strings.SplitN(v, "=", 2)

		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}

	return result
}

func message(msg string) TypeIrcMessage {
	matches := re.FindStringSubmatch(msg)
	tags := tags(matches[1])
	var nickname string

	fmt.Println("IRC MESSAGE:", msg)

	if v, ok := tags["display-name"]; ok {
		nickname = v
	}

	var message = TypeIrcMessage{
		Login:    strings.Split(matches[3], "!")[0],
		Nickname: nickname,
		Type:     matches[5],
		Channel:  strings.Replace(matches[6], "#", "", 1),
		Text:     matches[8],
		Tags:     tags,
		Prefix:   matches[3],
	}

	return message
}

func smiles(message TypeIrcMessage) string {
	msg := []rune(message.Text)
	offset := 0

	if _, ok := message.Tags["emotes"]; !ok {
		return message.Text
	}

	if len(message.Tags["emotes"]) == 0 {
		return message.Text
	}

	for _, smile := range strings.Split(message.Tags["emotes"], "/") {
		smileIdFromTo := strings.Split(smile, ":")
		smileId := smileIdFromTo[0]

		for _, fromTo := range strings.Split(smileIdFromTo[1], ",") {
			smileFromTo := strings.Split(fromTo, "-")
			smileFrom, _ := strconv.Atoi(smileFromTo[0])
			smileTo, _ := strconv.Atoi(smileFromTo[1])
			smileText := msg[smileFrom+offset : smileTo+offset+1]
			smileReplacer := []rune(fmt.Sprintf("<img class=\"smile twitch\" src=\"https://static-cdn.jtvnw.net/emoticons/v2/%s/default/dark/1.0\" alt=\"%s\"/>", smileId, string(smileText)))
			msg = append(msg[:smileFrom+offset], append(smileReplacer, msg[smileTo+1+offset:]...)...)
			offset += smileFrom - smileTo + len(smileReplacer) - 1
		}
	}

	return string(msg)
}
