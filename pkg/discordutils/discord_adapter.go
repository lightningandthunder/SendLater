package discordutils

import (
	"github.com/bwmarrin/discordgo"
)

type discordChannel *discordgo.Channel
type discordMessage *discordgo.Message

type DiscordAdapter interface {
	Open() error
	Close() error
	MessageReactionAdd(channelId, messageId, emojiId string) error
	UserChannelCreate(userId string) (discordChannel, error)
	ChannelMessageSend(channelId, content string) (discordMessage, error)
}

type discordWrapper struct {
	session *discordgo.Session
}

func (discord *discordWrapper) Open() error {
	return discord.session.Open()
}

func (discord *discordWrapper) Close() error {
	return discord.session.Close()
}

func (discord *discordWrapper) MessageReactionAdd(channelId, messageId, emojiId string) error {
	return discord.session.MessageReactionAdd(channelId, messageId, emojiId)
}

func (discord *discordWrapper) UserChannelCreate(userId string) (discordChannel, error) {
	channel, err := discord.session.UserChannelCreate(userId)
	return discordChannel(channel), err
}

func (discord *discordWrapper) ChannelMessageSend(channelId, content string) (discordMessage, error) {
	message, err := discord.session.ChannelMessageSend(channelId, content)
	return discordMessage(message), err
}

func newDiscordAdapter(botToken string) (DiscordAdapter, error) {
	discordSession, err := discordgo.New("Bot " + botToken)
	discordSession.AddHandler(handleMessage)
	discordSession.Identify.Intents = discordgo.IntentsDirectMessages
	return &discordWrapper{session: discordSession}, err

}
