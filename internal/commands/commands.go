package commands

import (
	"reflect"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/gotd/td/telegram/message/entity"
	tghtml "github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/message/styling"
	"go.uber.org/zap"
)

type command struct {
	log *zap.Logger
}

func Load(log *zap.Logger, dispatcher dispatcher.Dispatcher) {
	log = log.Named("commands")
	defer log.Info("Initialized all command handlers")
	Type := reflect.TypeOf(&command{log})
	Value := reflect.ValueOf(&command{log})
	for i := 0; i < Type.NumMethod(); i++ {
		Type.Method(i).Func.Call([]reflect.Value{Value, reflect.ValueOf(dispatcher)})
	}
}

// HTMLString creates a re-usable StyledTextOption that creates a fresh reader every time,
// preventing the MESSAGE_EMPTY bug caused by consumed io.Readers when gotgproto evaluates options twice.
func HTMLString(s string) styling.StyledTextOption {
	return styling.Custom(func(eb *entity.Builder) error {
		return tghtml.HTML(strings.NewReader(s), eb, tghtml.Options{})
	})
}
