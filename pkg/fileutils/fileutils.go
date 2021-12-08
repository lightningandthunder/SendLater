package fileutils

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const timeFormat = time.RFC3339
const sendFileDir = "../sendfiles/"

func init() {
	err := os.Mkdir(sendFileDir, os.ModeDir)
	if err != nil {
		panic("Unable to create send file directory: " + err.Error())
	}
}

// Todo - convert to UTC
// Todo - see if we can get local time from discord message
func SaveToFile(sendTime time.Time, message string) error {
	fileName := timeStringToFileName(sendTime.Format(timeFormat))

	fp, err := os.Create(fileName + ".txt")
	if err != nil {
		fmt.Println("Could not create file:", err)
		return err
	}

	_, err = fmt.Fprintln(fp, message)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	return nil
}

func readMessageFromFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error reading file: ", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, 400)
	for {
		bytesWritten, err := io.ReadFull(reader, buffer)
		if err == io.EOF {
			return string(buffer[0:bytesWritten]), nil
		}
	}
}

func stringToTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func timeStringToFileName(s string) string {
	return s + "_" + uuid.New().String()
}

func timeStringFromFileName(fileName string) (string, error) {
	stringSlice := strings.Split(fileName, "_")
	if len(stringSlice) != 2 {
		return "", fmt.Errorf("Invalid file name:", fileName)
	}
	return stringSlice[0], nil
}

func sendFileContentsAsDiscordMessage(fileName string, messagesSent chan bool, messagesErrored chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	message, err := readMessageFromFile(fileName)
	if err != nil {
		messagesErrored <- true
		return
	}

	// TODO - send message to Discord
	fmt.Println("Message:", message)
	messagesSent <- true
}

// TODO - make sure comparisons are all done in UTC
func SendPendingMessages() (filesSent int, filesErrored int, err error) {
	files, err := ioutil.ReadDir(sendFileDir)
	if err != nil {
		return 0, 0, err
	}

	// Channels to keep track of concurrent message processing
	// This is not necessary, but I gotta practice using goroutines somehow!
	messagesSent := make(chan bool, len(files))
	messagesErrored := make(chan bool, len(files))

	defer close(messagesSent)
	defer close(messagesErrored)

	wg := sync.WaitGroup{}

	for _, info := range files {
		timeString, err := timeStringFromFileName(info.Name())
		if err != nil {
			fmt.Println("Error while parsing time from file name:", err)
			messagesErrored <- true
			continue
		}

		t, err := stringToTime(timeString)
		if err != nil {
			fmt.Println("Error while parsing time from file name:", err)
			messagesErrored <- true
			continue
		}

		if time.Until(t) < 0 {
			go sendFileContentsAsDiscordMessage(info.Name(), messagesSent, messagesErrored, &wg)
		}

	}
	wg.Wait()

	return len(messagesSent), len(messagesErrored), err
}
