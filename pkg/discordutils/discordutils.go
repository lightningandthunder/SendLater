package discordutils

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const scheduleSignal = "signal"

var discord *discordgo.Session
var generalChatId string
var callbackHandler func(time.Time, string) error

func init() {
	// set General chat ID
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed while loading .env file:", err)
		dir, _ := os.Getwd()
		fmt.Println("Current directory:", dir)
		os.Exit(1)
	}

	generalChatId = os.Getenv("GENERAL_CHAT_ID")

	d, err := discordgo.New("Bot" + os.Getenv("BOT_TOKEN"))
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		os.Exit(1)
	}
	discord = d

	// Register handler function specifically for DMs to the bot
	discord.AddHandler(handleMessage)
	discord.Identify.Intents = discordgo.IntentsDirectMessages
}

// A "constant" representing the channel ID for General Chat on our server
func GetGeneralChannelID() string {
	return generalChatId
}

// Return a discord session struct
func GetDiscordSession() *discordgo.Session {
	return discord
}

// Set a package-level handler function to be invoked upon DMing the bot
func SetCallbackHandler(callback func(time.Time, string) error) {
	if callbackHandler == nil {
		callbackHandler = callback
	}
}

// Open a websocket connection to Discord and listen until an interrupt signal is received
func Listen() error {
	if callbackHandler == nil {
		return fmt.Errorf("DM callback handler function not set")
	}

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
// A package-level callback needs to be passed in to avoid circular imports with the discordutils package.
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

	// This callback function has to be set at the package level
	err := callbackHandler(scheduleTime, parsedMessage)
	if err != nil {
		SendDm(
			msg.ChannelID,
			fmt.Sprintf("Error scheduling your message: %s", err),
		)
		return
	}
}
