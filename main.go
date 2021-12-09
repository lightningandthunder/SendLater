package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

const scheduleSignal = "signal"
const generalChannelId = "TODO"

func main() {
	discord, err := discordgo.New("Bot " + "authentication token")
	if err != nil {
		log.Fatal("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	discord.AddHandler(handleMessage)

	// Just like the ping pong example, we only care about receiving message
	// events in this example.
	discord.Identify.Intents = discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	// Wait here until a system interrupt
	fmt.Println("Bot is up and running!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Terminating bot.")
	discord.Close()

	// For testing purposes
	for i := 0; i < 25; i++ {
		fileutils.ScheduleMessage(time.Now(), time.Now().UTC().Format(time.RFC3339))
	}

	time.Sleep(time.Second * 2)
	filesSent, filesErrored, err := fileutils.SendPendingMessages()
	fmt.Println("Main:", filesSent, filesErrored, err)
}

// This function will be called every time a DM is sent to the bot.
// It parses the original message for key information and schedules the message for
// a provided timestamp.
func handleMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	// Ignore this bot's messages - not that this is likely to happen, but still.
	if msg.Author.ID == session.State.User.ID {
		fmt.Println("Somehow, the bot sent a message to itself:", msg.Content)
		return
	}

	var scheduleTime time.Time
	var parsedMessage string

	messageParts := strings.Split(msg.Content, " ")

messageLoop:
	for index, str := range messageParts {
		switch index {
		// Skip messages that don't start with the signal
		case 0:
			if strings.ToLower(str) != scheduleSignal {
				return
			}
		// Try to parse a date and time
		case 1:
			t, err := time.Parse(time.RFC3339, str)
			if err != nil {
				sendDm(
					session,
					msg.ChannelID,
					fmt.Sprintf("Error parsing your timestamp: %s", err)+
						"\nPlease use RFC3339 date format; ex: 2019-10-12T14:20:50.52+07:00",
				)
				return
			}
			scheduleTime = t
		default:
			parsedMessage = strings.Join(messageParts[2:], " ")
			break messageLoop
		}
	}

	fileutils.ScheduleMessage(scheduleTime, parsedMessage)

	// TODO - What's the channel ID for our general chat?
	channel, err := session.Channel(generalChannelId)

	if err != nil {
		// If an error occurred, try to notify the user via DM.
		// It's possible that we have sent too many DMs and Discord is throttling us.
		session.ChannelMessageSend(
			msg.ChannelID,
			fmt.Sprintf("Error while trying to send your message: %s. It's possible that we're just overloading Discord; try again later.", err),
		)
		return
	}

	// Then we send the message through the channel we created.
	_, err = session.ChannelMessageSend(channel.ID, "Pong!")
	if err != nil {
		// If an error occurred, we failed to send the message.
		//
		// It may occur either when we do not share a server with the
		// user (highly unlikely as we just received a message) or
		// the user disabled DM in their settings (more likely).
		fmt.Println("error sending DM message:", err)
		session.ChannelMessageSend(
			msg.ChannelID,
			"Failed to send you a DM. "+
				"Did you disable DM in your privacy settings?",
		)
	}
}

func sendDm(session *discordgo.Session, channelId, msg string) {
	_, err := session.ChannelMessageSend(
		channelId,
		msg,
	)
	if err != nil {
		fmt.Println("Error while sending DM:", err)
		fmt.Println("Intended message:", msg)
	}
}
