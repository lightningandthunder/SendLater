package discordutils

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

type mockDiscord struct{}

// Functions to be mocked in each test case, as necessary
var mockOpen func() error
var mockClose func() error
var mockMessageReactionAdd func(channelId, messageId, emojiId string) error
var mockUserChannelCreate func(userId string) (discordChannel, error)
var mockChannelMessageSend func(channelId, content string) (discordMessage, error)

func (d mockDiscord) Open() error {
	return mockOpen()
}

func (d mockDiscord) Close() error {
	return mockClose()
}

func (d mockDiscord) MessageReactionAdd(channelId, messageId, emojiId string) error {
	return mockMessageReactionAdd(channelId, messageId, emojiId)
}

func (d mockDiscord) UserChannelCreate(userId string) (discordChannel, error) {
	return mockUserChannelCreate(userId)
}

func (d mockDiscord) ChannelMessageSend(channelId, content string) (discordMessage, error) {
	return mockChannelMessageSend(channelId, content)
}

func TestSendDmChannelOpenedFromUserId(t *testing.T) {
	discord = mockDiscord{}
	mockUserChannelCreate = func(userId string) (discordChannel, error) {
		dc := new(discordgo.Channel)
		dc.ID = "Channel ID"
		return dc, nil
	}
	mockChannelMessageSend = func(channelId, content string) (discordMessage, error) {
		if channelId != "Channel ID" {
			t.Error("Expected 'Channel ID', received ", channelId)
		}
		return new(discordgo.Message), nil

	}
	SendDm("Test string", "Test message")
}
