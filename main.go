package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lightningandthunder/sendlater/pkg/config"
	"github.com/lightningandthunder/sendlater/pkg/discordutils"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func loopOnSendPendingMessages(session *discordgo.Session) {
	elapsedSeconds := 0
	totalFilesSent := 0
	totalFilesErrored := 0
	for {
		filesSent, filesErrored, err := fileutils.SendPendingMessages(session)
		totalFilesSent += filesSent
		totalFilesErrored += filesErrored
		if err != nil {
			fmt.Println("Error while catching up on scheduled messages:", err)
		}
		elapsedSeconds += config.LoopWaitSeconds
		if elapsedSeconds >= config.HeartbeatSeconds {
			fmt.Printf(
				"%v messages sent successfully and %v messages errored out in the last %v seconds\n",
				totalFilesSent, totalFilesErrored, elapsedSeconds,
			)
			elapsedSeconds, totalFilesSent, totalFilesErrored = 0, 0, 0
		}
		time.Sleep(time.Second * config.LoopWaitSeconds)
	}
}

func main() {
	discordutils.SetCallbackHandler(fileutils.ScheduleMessage)
	fileutils.SetErrorDmCallback(discordutils.SendDm)
	session := discordutils.GetDiscordSession()

	// Loop forever on sending pending messages in a separate goroutine
	go loopOnSendPendingMessages(session)

	// Independently, loop forever on listening for DMs via websocket connection
	err := discordutils.Listen()
	if err != nil {
		fmt.Println("Error while setting up bot:", err)
	}

}
