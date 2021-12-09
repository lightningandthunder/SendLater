package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lightningandthunder/sendlater/pkg/fileutils"
)

const scheduleSignal = "signal"

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

// This function will be called every time a new
// message is created in a direct message with the bot.
func handleMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {

	// Ignore this bot's messages
	if msg.Author.ID == session.State.User.ID {
		return
	}

	// Parse message content
	containsSchedule, err := regexp.MatchString("^"+scheduleSignal, msg.Content)
	if err != nil {
		fmt.Println("Error with regex signal check:", err)
	}

	// Skip messages that don't start with the signal
	if !containsSchedule {
		return
	}

	// We create the private channel with the user who sent the message.
	channel, err := session.UserChannelCreate(msg.Author.ID)
	if err != nil {
		// If an error occurred, we failed to create the channel.
		//
		// Some common causes are:
		// 1. We don't share a server with the user (not possible here).
		// 2. We opened enough DM channels quickly enough for Discord to
		//    label us as abusing the endpoint, blocking us from opening
		//    new ones.
		fmt.Println("error creating channel:", err)
		session.ChannelMessageSend(
			msg.ChannelID,
			"Something went wrong while sending the DM!",
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
