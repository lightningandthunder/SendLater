package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lightningandthunder/sendlater/pkg/discordutils"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func loopOnSendPendingMessages(session *discordgo.Session) {
	// TODO - make this listen on a channel and terminate the infinite loop once a kill signal is received
	for {
		filesSent, filesErrored, err := fileutils.SendPendingMessages(session)
		fmt.Println("Main:", filesSent, filesErrored, err)
		time.Sleep(time.Minute)
	}
}

func main() {
	discordutils.SetCallbackHandler(fileutils.ScheduleMessage)
	session := discordutils.GetDiscordSession()

	// Loop forever on sending pending messages in a separate goroutine
	go loopOnSendPendingMessages(session)

	// Independently, loop forever on listening for DMs via websocket connection
	err := discordutils.Listen()
	if err != nil {
		fmt.Println("Error while setting up bot:", err)
	}

}
