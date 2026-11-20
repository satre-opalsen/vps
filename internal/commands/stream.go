package commands

import (
	"fmt"
	"html"
	"strings"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/dustin/go-humanize"
	"github.com/gotd/td/tg"
)

func (m *command) LoadStream(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("start")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(
		handlers.NewMessage(nil, sendLink),
	)
}

func supportedMediaFilter(m *types.Message) (bool, error) {
	if not := m.Media == nil; not {
		return false, dispatcher.EndGroups
	}
	switch m.Media.(type) {
	case *tg.MessageMediaDocument:
		return true, nil
	case *tg.MessageMediaPhoto:
		return true, nil
	case tg.MessageMediaClass:
		return false, dispatcher.EndGroups
	default:
		return false, nil
	}
}

func sendLink(ctx *ext.Context, u *ext.Update) error {
	if !u.EffectiveChat().IsAUser() {
		return dispatcher.EndGroups
	}
	chatId := u.EffectiveChat().GetID()
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, ext.ReplyTextString("⛔ You are not authorized to use this bot."), nil)
		return dispatcher.EndGroups
	}
	supported, err := supportedMediaFilter(u.EffectiveMessage)
	if err != nil {
		return err
	}
	if !supported {
		ctx.Reply(u, ext.ReplyTextString("⚠️ Please send or forward a valid file, document, video, or audio."), nil)
		return dispatcher.EndGroups
	}
	update, err := utils.ForwardMessages(ctx, chatId, config.ValueOf.LogChannelID, u.EffectiveMessage.ID)
	if err != nil {
		utils.Logger.Sugar().Error(err)
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Error storing file: %s", err.Error())), nil)
		return dispatcher.EndGroups
	}
	if len(update.Updates) < 2 {
		ctx.Reply(u, ext.ReplyTextString("❌ Error: unexpected update structure from Telegram."), nil)
		return dispatcher.EndGroups
	}
	msgIDUpdate, ok := update.Updates[0].(*tg.UpdateMessageID)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Error: unexpected update type."), nil)
		return dispatcher.EndGroups
	}
	messageID := msgIDUpdate.ID
	newMsg, ok := update.Updates[1].(*tg.UpdateNewChannelMessage)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Error: unexpected channel message update."), nil)
		return dispatcher.EndGroups
	}
	msg, ok := newMsg.Message.(*tg.Message)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Error: unexpected message structure."), nil)
		return dispatcher.EndGroups
	}
	doc := msg.Media
	file, err := utils.FileFromMedia(doc)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Error reading media: %s", err.Error())), nil)
		return dispatcher.EndGroups
	}
	fullHash := utils.PackFile(
		file.FileName,
		file.FileSize,
		file.MimeType,
		file.ID,
	)
	hash := utils.GetShortHash(fullHash)
	link := fmt.Sprintf("%s/stream/%d?hash=%s", config.ValueOf.Host, messageID, hash)
	downloadLink := link + "&d=true"
	streamLink := link

	fileName := file.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("file_%d", messageID)
	}

	sizeStr := "Unknown"
	if file.FileSize > 0 {
		sizeStr = humanize.Bytes(uint64(file.FileSize))
	}

	isMedia := strings.Contains(file.MimeType, "video") ||
		strings.Contains(file.MimeType, "audio") ||
		strings.Contains(file.MimeType, "pdf")

	var responseText string
	if isMedia {
		responseText = fmt.Sprintf(
			"⚡ <b>File Ready!</b>\n\n"+
				"📁 <b>Name:</b> <code>%s</code>\n"+
				"📦 <b>Size:</b> <code>%s</code>\n\n"+
				"🚀 <b>Direct Download Link:</b>\n<code>%s</code>\n\n"+
				"🎬 <b>Fast Online Stream:</b>\n<code>%s</code>\n\n"+
				"💡 <i>Tip: Paste the direct download link into IDM / ADM for maximum speed.</i>",
			html.EscapeString(fileName),
			sizeStr,
			downloadLink,
			streamLink,
		)
	} else {
		responseText = fmt.Sprintf(
			"⚡ <b>File Ready!</b>\n\n"+
				"📁 <b>Name:</b> <code>%s</code>\n"+
				"📦 <b>Size:</b> <code>%s</code>\n\n"+
				"🚀 <b>Direct Download Link:</b>\n<code>%s</code>\n\n"+
				"💡 <i>Tip: Paste the direct download link into IDM / ADM for maximum speed.</i>",
			html.EscapeString(fileName),
			sizeStr,
			downloadLink,
		)
	}

	buttons := []tg.KeyboardButtonClass{
		&tg.KeyboardButtonURL{
			Text: "📥 Fast Download",
			URL:  downloadLink,
		},
	}
	if isMedia {
		buttons = append(buttons, &tg.KeyboardButtonURL{
			Text: "🎬 Stream / Watch",
			URL:  streamLink,
		})
	}
	row := tg.KeyboardButtonRow{
		Buttons: buttons,
	}
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{row},
	}

	if strings.Contains(link, "http://localhost") {
		_, err = ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(responseText)), &ext.ReplyOpts{
			NoWebpage:        true,
			ReplyToMessageId: u.EffectiveMessage.ID,
		})
	} else {
		_, err = ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(responseText)), &ext.ReplyOpts{
			Markup:           markup,
			NoWebpage:        true,
			ReplyToMessageId: u.EffectiveMessage.ID,
		})
	}
	if err != nil {
		utils.Logger.Sugar().Errorf("sendLink reply error: %v", err)
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Error: %s", err.Error())), nil)
	}
	return dispatcher.EndGroups
}
