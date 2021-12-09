package main

import (
	"fmt"
	"time"

	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func main() {
	// For testing purposes
	for i := 0; i < 25; i++ {
		fileutils.ScheduleMessage(time.Now(), time.Now().UTC().Format(time.RFC3339))
	}

	time.Sleep(time.Second * 2)
	filesSent, filesErrored, err := fileutils.SendPendingMessages()
	fmt.Println("Main:", filesSent, filesErrored, err)
}
