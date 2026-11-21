package commands

import (
	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
)

func (m *command) LoadClear(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("clear")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(handlers.NewCommand("clear", clearChat))
}

func clearChat(ctx *ext.Context, u *ext.Update) error {
	if !u.EffectiveChat().IsAUser() {
		return dispatcher.EndGroups
	}
	chatId := u.EffectiveChat().GetID()
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, ext.ReplyTextString("⛔ ACCESS DENIED // Unauthorized Operator."), nil)
		return dispatcher.EndGroups
	}

	currentID := u.EffectiveMessage.ID
	welcomeID := utils.GetWelcomeMessageID(chatId)

	minID := currentID - 250
	if welcomeID > 0 && welcomeID < minID {
		minID = welcomeID
	}
	if minID < 1 {
		minID = 1
	}

	var idsToDelete []int
	for id := currentID; id >= minID; id-- {
		if welcomeID > 0 && id == welcomeID {
			continue // Do NOT delete the welcome message!
		}
		idsToDelete = append(idsToDelete, id)
	}

	// Delete in chunks of 100
	const chunkSize = 100
	for i := 0; i < len(idsToDelete); i += chunkSize {
		end := i + chunkSize
		if end > len(idsToDelete) {
			end = len(idsToDelete)
		}
		chunk := idsToDelete[i:end]
		_, err := ctx.Raw.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
			Revoke: true,
			ID:     chunk,
		})
		if err != nil {
			utils.Logger.Sugar().Warnf("MessagesDeleteMessages chunk error: %v", err)
		}
	}

	// If there was no known welcome message recorded, send one now so the chat is left with only the welcome message
	if welcomeID == 0 {
		_, _ = sendWelcomeMessage(ctx, u)
	}

	return dispatcher.EndGroups
}
