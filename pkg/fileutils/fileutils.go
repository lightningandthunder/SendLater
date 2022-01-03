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

	"github.com/lightningandthunder/sendlater/pkg/discordutils"
)

const timeFormat = time.RFC3339
const sendFileDir = "sendfiles"

var sendErrorDm func(string, string)

func init() {
	err := os.MkdirAll(sendFileDir, os.ModePerm)
	if err != nil {
		panic("Unable to create send file directory:" + err.Error())
	}
}

// Dependency-inject a function to send a DM to a user in the event of an error
func SetErrorDmCallback(f func(string, string)) {
	sendErrorDm = f
}

// Record a desired send timestamp and a message to send.
// Returns an error if something went wrong, or nil if it went right.
func ScheduleMessage(sendTime time.Time, message string, userID string) error {
	sendTimeUtc := sendTime.UTC()
	fileName := timeAndUserIdToFileName(sendTimeUtc.Format(timeFormat), userID)

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
func SendPendingMessages(session discordutils.DiscordAdapter) (filesSent int, filesErrored int, err error) {
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
		t, userId, err := timeAndUserIdFromFileName(info.Name())
		if err != nil {
			fmt.Println("Error while parsing time from file name:", err)
			messagesErrored <- true
			continue
		}

		// If scheduled time is in the past, fire off a goroutine to send the message to Discord
		if t.Sub(nowUtc) < 0 {
			wg.Add(1)
			go sendFileContentsAsDiscordMessage(info.Name(), userId, messagesSent, messagesErrored, &wg)
		}

	}

	wg.Wait()

	return len(messagesSent), len(messagesErrored), err
}

// Open a selected file in the sendfiles directory.
// Returns file contents and error, if any.
func readMessageFromFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
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

// Parse a string to a time.Time struct.
func stringToTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

// Encode a parsable time string and user ID in a file name.
func timeAndUserIdToFileName(timeString, userId string) string {
	return timeString + "_" + userId + "_.txt"
}

// Extract a time.Time struct and user ID from a file
func timeAndUserIdFromFileName(fileName string) (time.Time, string, error) {
	stringSlice := strings.Split(fileName, "_")

	// We expect the string to split into target_time, user_id, and ".txt"
	if len(stringSlice) != 3 {
		return time.Now(), "", fmt.Errorf("Invalid file name:" + fileName)
	}
	timeString, userId := stringSlice[0], stringSlice[1]

	timeStruct, err := stringToTime(timeString)
	if err != nil {
		return time.Time{}, userId, err
	}

	return timeStruct, userId, nil
}

// Read a scheduled message from a file and send it to General Chat
func sendFileContentsAsDiscordMessage(fileName string, userId string, messagesSent chan bool, messagesErrored chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	fileFullPath := filepath.Join(sendFileDir, fileName)

	message, err := readMessageFromFile(fileFullPath)
	if err != nil {
		fmt.Println("Error reading from file:", err)
		messagesErrored <- true
		return
	}

	fmt.Println("Sending error to ", userId)
	// Send the message to general chat
	err = discordutils.SendMessageToGeneralChat(message)
	if err != nil {
		errorMsg := fmt.Errorf("There was an error sending your scheduled message: %v", err)
		sendErrorDm(userId, errorMsg.Error())
		fmt.Println("Error sending scheduled message:", err)
		messagesErrored <- true
		return
	}

	// Otherwise, message was successful, so we can clean up.
	messagesSent <- true

	err = os.Remove(fileFullPath)
	if err != nil {
		fmt.Println("Error removing sendfile...", err)
	}
}
