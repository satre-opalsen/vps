package commands

import (
	"fmt"
	"html"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/storage"
	tghtml "github.com/gotd/td/telegram/message/html"
)

func (m *command) LoadStart(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("start")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(handlers.NewCommand("start", start))
}

func start(ctx *ext.Context, u *ext.Update) error {
	chatId := u.EffectiveChat().GetID()
	peerChatId := ctx.PeerStorage.GetPeerById(chatId)
	if peerChatId.Type != int(storage.TypeUser) {
		return dispatcher.EndGroups
	}
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, ext.ReplyTextString("⛔ You are not authorized to use this bot."), nil)
		return dispatcher.EndGroups
	}

	name := "there"
	if u.EffectiveUser() != nil && u.EffectiveUser().FirstName != "" {
		name = u.EffectiveUser().FirstName
	}

	welcomeText := fmt.Sprintf(
		"👋 <b>Hey %s! Welcome to File to Link Bot</b>\n\n"+
			"⚡ <b>High-Speed Direct Link & Streaming Engine</b>\n\n"+
			"Send or forward any Telegram media or document to generate permanent, fast HTTP download and stream links.\n\n"+
			"✨ <b>Key Features:</b>\n"+
			"• 🚀 <b>Fast Downloads:</b> Multi-threaded direct HTTP download links\n"+
			"• 🎬 <b>Web Streaming:</b> Watch videos & listen to audio directly online\n"+
			"• 📥 <b>Resume Support:</b> Compatible with IDM, ADM, curl, and wget\n"+
			"• 📦 <b>File Support:</b> Videos, Music, Documents, and Files up to 2 GB\n\n"+
			"📤 <b>Get Started:</b>\n"+
			"Simply <b>send or forward any file</b> here right now!",
		html.EscapeString(name),
	)

	_, err := ctx.Reply(u, ext.ReplyTextStyledText(tghtml.String(nil, welcomeText)), &ext.ReplyOpts{
		NoWebpage: true,
	})
	if err != nil {
		utils.Logger.Sugar().Error(err)
	}
	return dispatcher.EndGroups
}
