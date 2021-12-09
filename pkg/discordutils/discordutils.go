package discordutils

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

// TODO: Need to figure out how to pass in the ScheduleMessage function here

const scheduleSignal = "signal"

var discord *discordgo.Session

func init() {
	d, err := discordgo.New("Bot " + "authentication token")
	if err != nil {
		log.Fatal("error creating Discord session,", err)
		return
	}
	discord = d

	// Register handler function specifically for DMs to the bot
	discord.AddHandler(handleMessage)
	discord.Identify.Intents = discordgo.IntentsDirectMessages
}

// A "constant" representing the channel ID for General Chat on our server
func GetGeneralChannelID() string {
	// Need to read this in from .env
	return "TODO"
}

// Open a websocket connection to Discord and listen until an interrupt signal is received
func Listen() error {
	// Open a websocket connection to Discord and begin listening.
	err := discord.Open()
	if err != nil {
		return err
	}

	// Wait here until a system interrupt
	fmt.Println("Bot is up and running!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Terminating bot.")
	discord.Close()

	return nil
}

// Send a DM to a provided channel ID
func SendDm(channelId, msg string) {
	_, err := discord.ChannelMessageSend(
		channelId,
		msg,
	)
	if err != nil {
		fmt.Println("Error while sending DM:", err)
		fmt.Println("Intended message:", msg)
	}
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
				SendDm(
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

	err := fileutils.ScheduleMessage(scheduleTime, parsedMessage)
	if err != nil {
		SendDm(
			msg.ChannelID,
			fmt.Sprintf("Error scheduling your message: %s", err),
		)
		return
	}

	// TODO - What's the channel ID for our general chat?
	channel, err := session.Channel(GetGeneralChannelID())

	if err != nil {
		// If an error occurred, try to notify the user via DM.
		// It's possible that we have sent too many DMs and Discord is throttling us.
		SendDm(
			msg.ChannelID,
			fmt.Sprintf("Error while trying to send your message: %s."+
				"It's possible that we're just overloading Discord; try again later.", err),
		)
		return
	}

}
