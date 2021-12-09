package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lightningandthunder/sendlater/pkg/discordutils"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func loopOnSendPendingMessages(session *discordgo.Session) {
	elapsedSeconds := 0
	totalFilesSent := 0
	totalFilesErrored := 0

	// TODO - make this listen on a channel and terminate the infinite loop once a kill signal is received
	for {
		filesSent, filesErrored, err := fileutils.SendPendingMessages(session)
		totalFilesSent += filesSent
		totalFilesErrored += filesErrored
		if err != nil {
			fmt.Println("Error while catching up on scheduled messages:", err)
		}
		elapsedSeconds += 10
		if elapsedSeconds == 60 {
			fmt.Printf(
				"%v messages sent successfully and %v messages errored out in the last %v seconds\n",
				totalFilesSent, totalFilesErrored, elapsedSeconds,
			)
			elapsedSeconds, totalFilesSent, totalFilesErrored = 0, 0, 0
		}

		time.Sleep(time.Second * 10)
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
