package discordutils

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const scheduleSignal = "schedule"

var discord DiscordAdapter
var generalChatId string
var callbackHandler func(time.Time, string, string) error

func init() {
	// set General chat IDƒ
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed while loading .env file:", err)
		dir, _ := os.Getwd()
		fmt.Println("Current directory:", dir)
		os.Exit(1)
	}

	generalChatId = os.Getenv("GENERAL_CHAT_ID")

	discordSession, err := newDiscordAdapter(os.Getenv("BOT_TOKEN"))
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		os.Exit(1)
	}
	discord = discordSession

	fmt.Println("Discord session initialized.")
}

// Send a DM to a provided user ID
func SendDm(userId, msg string) {
	dmChannel, err := discord.UserChannelCreate(userId)
	if err != nil {
		fmt.Println("Error while getting DM channel:", err)
		return
	}

	_, err = discord.ChannelMessageSend(
		dmChannel.ID,
		msg,
	)
	if err != nil {
		fmt.Println("Error while sending DM:", err)
		fmt.Println("Intended message:", msg)
	}
}

// Take a message and send it to our server's General Chat channel.
func SendMessageToGeneralChat(message string) error {
	_, err := discord.ChannelMessageSend(generalChatId, message)
	if err != nil {
		return err
	}
	return nil
}

func GetGeneralChannelID() string {
	return generalChatId
}

func GetDiscordSession() DiscordAdapter {
	return discord
}

// Set a package-level handler function to be invoked upon DMing the bot
func SetCallbackHandler(callback func(time.Time, string, string) error) {
	if callbackHandler == nil {
		callbackHandler = callback
	}
}

// Open a websocket connection to Discord and listen until an interrupt signal is received
func Listen() error {
	if callbackHandler == nil {
		return errors.New("DM callback handler function not set")
	}

	// Open a websocket connection to Discord and begin listening.
	err := discord.Open()
	if err != nil {
		return err
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Wait here until a system interrupt
	fmt.Println("Bot is up and running!")
	<-signalChannel

	fmt.Println("\nTerminating bot.")
	_ = discord.Close()
	return nil
}

// Send an explanation for how to use this bot to a given userID.
func sendHelpDm(userId string) {
	message := `
To schedule a message in X seconds, minutes, or hours, DM the bot in this format:
schedule <X seconds/minutes/hours> <message>

Or to schedule a message at a specific time:
schedule <RC3339 timestamp in the format yyyy-mm-ddThh:mm:ss.mm-gmt_offset> <message> 
where gmt_offset is a GMT offset, such as +07:00 or -05:00
`
	SendDm(userId, message)
}

// Get an UTC time from an offset such as "10 minutes."
// Returns a time.Time struct and any errors.
func getTargetTimeFromOffset(increment int, unit string) (time.Time, error) {
	now := time.Now().UTC()

	var _unit time.Duration
	switch unit {
	case "second":
	case "seconds":
		_unit = time.Second
	case "minute":
	case "minutes":
		_unit = time.Minute
	case "hour":
	case "hours":
		_unit = time.Hour
	default:
		return now, fmt.Errorf("invalid time unit %v", unit)
	}

	return now.Add(_unit * time.Duration(increment)), nil
}

// Extract a target time from the string elements of a message.
// Return the target time, the first index after target time-related strings, and any errors that occurred.
func extractTargetTimeFromMessage(messageParts []string) (time.Time, int, error) {
	now := time.Now().UTC()

	// Try to parse a "(int) seconds/minutes/hours" format string
	matched, err := regexp.MatchString(`\d+ (seconds?|minutes?|hours?)`, strings.Join(messageParts[1:3], " "))
	if err != nil {
		return now, 0, err
	}

	// Try to schedule a message X seconds/minutes/hours from now
	if matched {
		intValue, err := strconv.Atoi(messageParts[1])
		if err != nil {
			return now, 0, err
		}

		targetTime, err := getTargetTimeFromOffset(intValue, messageParts[2])
		if err != nil {
			return now, 0, err
		}
		return targetTime, 3, nil
	}
	// Otherwise, parse an exact time
	t, err := time.Parse(time.RFC3339, messageParts[1])
	if err != nil {
		return now, 0, err
	}
	return t, 2, nil

}

// This function will be called every time a DM is sent to the bot.
// It parses the original message for key information and schedules the message for
// a provided timestamp.
// A package-level callback needs to be referenced to avoid circular imports with the discordutils package.
func handleMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	// Ignore this bot's messages
	if msg.Author.ID == session.State.User.ID {
		return
	}

	messageParts := strings.Split(msg.Content, " ")

	// If the user asked for help, just send them a help message
	if strings.ToLower(messageParts[0]) == "help" {
		sendHelpDm(msg.Author.ID)
		return
	}

	// If the user sent something else not starting with "schedule" or "help"; ignore it
	if strings.ToLower(messageParts[0]) != scheduleSignal {
		return
	}

	// Otherwise, strip out and calculate target time from the message
	targetTime, messageStartIndex, err := extractTargetTimeFromMessage(messageParts)
	if err != nil {
		SendDm(
			msg.Author.ID,
			fmt.Sprintf("Error scheduling your message: %s", err),
		)
		return
	}

	// Prepend the message with some boilerplate
	msgSliceWithAttribution := append(
		[]string{msg.Author.Username + " scheduled a message to say: "},
		messageParts[messageStartIndex:]...,
	)
	parsedMessage := strings.Join(msgSliceWithAttribution, " ")

	// This callback function to schedule the message has to be dependency-injected at the package level
	err = callbackHandler(targetTime, parsedMessage, msg.Author.ID)
	if err != nil {
		SendDm(
			msg.Author.ID,
			fmt.Sprintf("Error scheduling your message: %s", err),
		)
	}

	// Try to add a thumbs-up emoji to the message if everything went well
	userDmChannel, err := discord.UserChannelCreate(msg.Author.ID)
	if err != nil {
		fmt.Println("Couldn't react to a message:", err)
	}

	err = discord.MessageReactionAdd(userDmChannel.ID, msg.ID, "👍")
	if err != nil {
		fmt.Println("Error adding emoji:", err)
	}
}
