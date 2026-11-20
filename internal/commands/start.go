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
		"<a href=\"https://media.tenor.com/AmZv66Mg1DYAAAAM/ghost-call-of-duty.gif\">&#8204;</a>"+
			"💀 <b>OPERATOR:</b> <code>%s</code>\n\n"+
			"⚡ <b>High-Speed File Streaming & Link Engine</b>\n\n"+
			"┌ 🚀 <b>Fast Downloads:</b> Multi-threaded direct streams\n"+
			"├ 🎬 <b>Web Streaming:</b> Online player (VLC / MX Player)\n"+
			"├ 📥 <b>Resume Support:</b> Compatible with IDM, ADM & curl\n"+
			"└ 📦 <b>File Limit:</b> Up to 2 GB per file\n\n"+
			"📤 <b>Send or forward any file below to generate a link.</b>",
		html.EscapeString(name),
	)

	var rows []tg.KeyboardButtonRow
	if config.ValueOf.Host != "" {
		hostTitle := strings.TrimPrefix(config.ValueOf.Host, "https://")
		hostTitle = strings.TrimPrefix(hostTitle, "http://")
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonURL{
					Text: fmt.Sprintf("🌐 %s", hostTitle),
					URL:  config.ValueOf.Host,
				},
			},
		})
	}

	opts := &ext.ReplyOpts{
		NoWebpage: false,
	}
	if len(rows) > 0 {
		opts.Markup = &tg.ReplyInlineMarkup{Rows: rows}
	}

	_, err := ctx.Reply(u, ext.ReplyTextStyledText(HTMLString(welcomeText)), opts)
	if err != nil {
		utils.Logger.Sugar().Errorf("start reply error: %v", err)
	}
	return dispatcher.EndGroups
}
