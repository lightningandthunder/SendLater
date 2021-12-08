package fileutils

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const timeFormat = time.RFC3339
const sendFileDir = "sendfiles"

func init() {
	err := os.MkdirAll(sendFileDir, os.ModePerm)
	if err != nil {
		panic("Unable to create send file directory:" + err.Error())
	}
}

// Record a desired send timestamp and a message to send.
// Returns an error if something went wrong, or nil if it went right.
func ScheduleMessage(sendTime time.Time, message string) error {
	// Todo - see if we can get local time from discord message
	sendTimeUtc := sendTime.UTC()
	fileName := timeStringToFileName(sendTimeUtc.Format(timeFormat))

	fp, err := os.Create(filepath.Join(sendFileDir, fileName))
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

// Review and send all scheduled messages whose schedule timestamps are in the past.
// Returns number of messages sent, number of failed messages, and the most recent internal error received.
func SendPendingMessages() (filesSent int, filesErrored int, err error) {
	files, err := ioutil.ReadDir(sendFileDir)
	if err != nil {
		return 0, 0, err
	}

	// Channels to keep track of concurrent message processing
	// This is probably not necessary, but I gotta practice using goroutines somehow!
	messagesSent := make(chan bool, len(files))
	messagesErrored := make(chan bool, len(files))

	defer close(messagesSent)
	defer close(messagesErrored)

	wg := sync.WaitGroup{}

	nowUtc := time.Now().UTC()

	for _, info := range files {
		t, err := timeFromFileName(info.Name())
		if err != nil {
			fmt.Println("Error while parsing time from file name:", err)
			messagesErrored <- true
			continue
		}

		// If scheduled time is in the past, fire off a goroutine to send the message to Discord
		if t.Sub(nowUtc) < 0 {
			wg.Add(1)
			go sendFileContentsAsDiscordMessage(info.Name(), messagesSent, messagesErrored, &wg)
		}

	}

	wg.Wait()

	return len(messagesSent), len(messagesErrored), err
}

func readMessageFromFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error reading file: ", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	bytes, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("Error reading message from file:", err)
		return "", err
	}

	return string(bytes), nil
}

func stringToTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func timeStringToFileName(s string) string {
	return s + "_" + uuid.New().String() + ".txt"
}

func timeFromFileName(fileName string) (time.Time, error) {
	timeString, err := timeStringFromFileName(fileName)
	if err != nil {
		return time.Time{}, err
	}

	timeStruct, err := stringToTime(timeString)
	if err != nil {
		return time.Time{}, err
	}

	return timeStruct, nil
}

func timeStringFromFileName(fileName string) (string, error) {
	stringSlice := strings.Split(fileName, "_")
	if len(stringSlice) != 2 {
		return "", fmt.Errorf("Invalid file name:" + fileName)
	}
	return stringSlice[0], nil
}

func sendFileContentsAsDiscordMessage(fileName string, messagesSent chan bool, messagesErrored chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	fileFullPath := filepath.Join(sendFileDir, fileName)

	message, err := readMessageFromFile(fileFullPath)
	if err != nil {
		fmt.Println("Got an error in goroutine:", err)
		messagesErrored <- true
		return
	}

	// TODO - send message to Discord
	fmt.Println("Message:", message)
	messagesSent <- true

	err = os.Remove(fileFullPath)
	if err != nil {
		fmt.Println("Error removing sendfile...", err)
	}
}
