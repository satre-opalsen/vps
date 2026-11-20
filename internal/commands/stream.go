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

func supportedMediaFilter(m *types.Message) bool {
	if m == nil || m.Media == nil {
		return false
	}
	switch m.Media.(type) {
	case *tg.MessageMediaDocument:
		return true
	case *tg.MessageMediaPhoto:
		return true
	default:
		return false
	}
}

func sendLink(ctx *ext.Context, u *ext.Update) error {
	if !u.EffectiveChat().IsAUser() {
		return dispatcher.EndGroups
	}
	chatId := u.EffectiveChat().GetID()
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, ext.ReplyTextString("⛔ ACCESS DENIED // Unauthorized Operator."), nil)
		return dispatcher.EndGroups
	}

	// Skip bot commands like /start so they are handled by their dedicated handlers
	if u.EffectiveMessage != nil && strings.HasPrefix(strings.TrimSpace(u.EffectiveMessage.Text), "/") {
		return nil
	}

	// Handle non-media text messages by asking for a document/file
	if !supportedMediaFilter(u.EffectiveMessage) {
		textReply := "⚠️ <b>NO FILE DETECTED</b>\n\n" +
			"Please send or forward a <b>document, video, audio, or file</b> to generate a fast direct link!"
		_, _ = ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(textReply)), &ext.ReplyOpts{
			NoWebpage:        true,
			ReplyToMessageId: u.EffectiveMessage.ID,
		})
		return dispatcher.EndGroups
	}

	update, err := utils.ForwardMessages(ctx, chatId, config.ValueOf.LogChannelID, u.EffectiveMessage.ID)
	if err != nil {
		utils.Logger.Sugar().Error(err)
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Storage Error: %s", err.Error())), nil)
		return dispatcher.EndGroups
	}
	if len(update.Updates) < 2 {
		ctx.Reply(u, ext.ReplyTextString("❌ Unexpected Telegram update structure."), nil)
		return dispatcher.EndGroups
	}
	msgIDUpdate, ok := update.Updates[0].(*tg.UpdateMessageID)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Unexpected update message ID."), nil)
		return dispatcher.EndGroups
	}
	messageID := msgIDUpdate.ID
	newMsg, ok := update.Updates[1].(*tg.UpdateNewChannelMessage)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Unexpected channel message update."), nil)
		return dispatcher.EndGroups
	}
	msg, ok := newMsg.Message.(*tg.Message)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("❌ Unexpected message type."), nil)
		return dispatcher.EndGroups
	}
	doc := msg.Media
	file, err := utils.FileFromMedia(doc)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Media extraction error: %s", err.Error())), nil)
		return dispatcher.EndGroups
	}

	code := utils.EncodeMessageID(messageID)
	downloadLink := fmt.Sprintf("%s/telegram/%s/dl", config.ValueOf.Host, code)
	streamLink := fmt.Sprintf("%s/telegram/%s/stream", config.ValueOf.Host, code)

	fileName := file.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("file_%d", messageID)
	}

	sizeStr := "Unknown"
	if file.FileSize > 0 {
		sizeStr = humanize.Bytes(uint64(file.FileSize))
	}

	lowerName := strings.ToLower(fileName)
	isMedia := strings.Contains(file.MimeType, "video") ||
		strings.Contains(file.MimeType, "audio") ||
		strings.Contains(file.MimeType, "pdf") ||
		strings.HasSuffix(lowerName, ".mp4") ||
		strings.HasSuffix(lowerName, ".mkv") ||
		strings.HasSuffix(lowerName, ".webm") ||
		strings.HasSuffix(lowerName, ".avi") ||
		strings.HasSuffix(lowerName, ".mov") ||
		strings.HasSuffix(lowerName, ".mp3") ||
		strings.HasSuffix(lowerName, ".m4a") ||
		strings.HasSuffix(lowerName, ".flac") ||
		strings.HasSuffix(lowerName, ".wav") ||
		strings.HasSuffix(lowerName, ".ogg")

	var responseText string
	if isMedia {
		responseText = fmt.Sprintf(
			"⚡ <b>FILE DEPLOYED SUCCESSFULLY</b>\n\n"+
				"┌ 📁 <b>Name:</b> <code>%s</code>\n"+
				"└ 📦 <b>Size:</b> <code>%s</code>\n\n"+
				"🚀 <b>Download:</b> <a href=\"%s\">Click to Start Download</a>\n"+
				"🎬 <b>Stream:</b> <a href=\"%s\">Click to Stream Online</a>\n\n"+
				"💡 <i>Tap buttons below to stream or copy link:</i>",
			html.EscapeString(fileName),
			sizeStr,
			downloadLink,
			streamLink,
		)
	} else {
		responseText = fmt.Sprintf(
			"⚡ <b>FILE DEPLOYED SUCCESSFULLY</b>\n\n"+
				"┌ 📁 <b>Name:</b> <code>%s</code>\n"+
				"└ 📦 <b>Size:</b> <code>%s</code>\n\n"+
				"🚀 <b>Download:</b> <a href=\"%s\">Click to Start Download</a>\n\n"+
				"💡 <i>Tap buttons below to start download or copy link:</i>",
			html.EscapeString(fileName),
			sizeStr,
			downloadLink,
		)
	}

	var rows []tg.KeyboardButtonRow
	if isMedia {
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonURL{
					Text: "🎬 Stream Online",
					URL:  streamLink,
				},
				&tg.KeyboardButtonURL{
					Text: "📥 Start Download",
					URL:  downloadLink,
				},
			},
		})
	} else {
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonURL{
					Text: "📥 Start Download",
					URL:  downloadLink,
				},
			},
		})
	}

	// Full width copy button underneath
	rows = append(rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCopy{
				Text:     "📋 Copy Download Link",
				CopyText: downloadLink,
			},
		},
	})

	markup := &tg.ReplyInlineMarkup{
		Rows: rows,
	}

	_, err = ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(responseText)), &ext.ReplyOpts{
		Markup:           markup,
		NoWebpage:        true,
		ReplyToMessageId: u.EffectiveMessage.ID,
	})
	if err != nil {
		utils.Logger.Sugar().Errorf("sendLink reply error: %v", err)
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("❌ Error: %s", err.Error())), nil)
	}
	return dispatcher.EndGroups
}
