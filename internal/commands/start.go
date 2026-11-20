package commands

import (
	"fmt"
	"html"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/functions"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

func (m *command) LoadStart(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("start")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(handlers.NewCommand("start", start))
}

func start(ctx *ext.Context, u *ext.Update) error {
	if !u.EffectiveChat().IsAUser() {
		return dispatcher.EndGroups
	}
	chatId := u.EffectiveChat().GetID()
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, ext.ReplyTextString("⛔ ACCESS DENIED // Unauthorized Operator."), nil)
		return dispatcher.EndGroups
	}

	name := "OPERATOR"
	if u.EffectiveUser() != nil && u.EffectiveUser().FirstName != "" {
		name = u.EffectiveUser().FirstName
	}

	welcomeText := fmt.Sprintf(
		"<a href=\"https://media.tenor.com/wSlgxeeIIB4AAAA1/rei-da-aura-mais-ego.webp\">&#8204;</a>"+
			"💀 <b>OPERATOR:</b> <code>%s</code>\n\n"+
			"⚡ <b>High-Speed File Streaming & Link Engine</b>\n\n"+
			"┌ 🚀 <b>Fast Downloads:</b> Multi-threaded direct streams\n"+
			"├ 🎬 <b>Web Streaming:</b> Online player (VLC / MX Player)\n"+
			"├ 📥 <b>Resume Support:</b> Compatible with IDM, ADM & curl\n"+
			"└ 📦 <b>File Limit:</b> Up to 2 GB per file\n\n"+
			"📤 <b>Send or forward any file below to generate a link.</b>",
		html.EscapeString(name),
	)

	var tb entity.Builder
	if err := styling.Perform(&tb, HTMLString(welcomeText)); err != nil {
		utils.Logger.Sugar().Errorf("styling error: %v", err)
		return err
	}
	msgText, entities := tb.Complete()

	peer := u.EffectiveChat().GetInputPeer()
	if peer == nil {
		peer = functions.GetInputPeerClassFromId(ctx.PeerStorage, chatId)
	}

	var replyTo tg.InputReplyToClass
	if u.EffectiveMessage != nil && u.EffectiveMessage.ID != 0 {
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: u.EffectiveMessage.ID}
	}

	req := &tg.MessagesSendMessageRequest{
		Peer:        peer,
		Message:     msgText,
		Entities:    entities,
		InvertMedia: true,
		ReplyTo:     replyTo,
	}

	_, err := ctx.SendMessage(chatId, req)
	if err != nil {
		utils.Logger.Sugar().Errorf("start SendMessage error: %v, falling back to ctx.Reply", err)
		_, _ = ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(welcomeText)), &ext.ReplyOpts{
			NoWebpage: false,
		})
	}
	return dispatcher.EndGroups
}
