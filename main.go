package main

import (
	"fmt"
	"time"

	"github.com/lightningandthunder/sendlater/pkg/discordutils"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func main() {
	session := discordutils.GetDiscordSession()

	time.Sleep(time.Second * 2)
	filesSent, filesErrored, err := fileutils.SendPendingMessages(session)
	fmt.Println("Main:", filesSent, filesErrored, err)
}
