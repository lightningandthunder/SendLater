package main

import (
	"fmt"

	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

func main() {
	// fileutils.SaveToFile(time.Now(), "First test woo!")
	// time.Sleep(time.Second * 2)
	filesSent, filesErrored, err := fileutils.SendPendingMessages()
	fmt.Println("Main:", filesSent, filesErrored, err)
}
