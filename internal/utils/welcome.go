package utils

import (
	"encoding/json"
	"os"
	"sync"
)

var (
	welcomeMu   sync.RWMutex
	welcomeMap  = make(map[int64]int)
	welcomeFile = "welcome_msgs.json"
)

func init() {
	welcomeMu.Lock()
	defer welcomeMu.Unlock()
	data, err := os.ReadFile(welcomeFile)
	if err == nil {
		_ = json.Unmarshal(data, &welcomeMap)
	}
}

func SetWelcomeMessageID(chatId int64, msgID int) {
	welcomeMu.Lock()
	defer welcomeMu.Unlock()
	welcomeMap[chatId] = msgID
	data, err := json.Marshal(welcomeMap)
	if err == nil {
		_ = os.WriteFile(welcomeFile, data, 0644)
	}
}

func GetWelcomeMessageID(chatId int64) int {
	welcomeMu.RLock()
	defer welcomeMu.RUnlock()
	return welcomeMap[chatId]
}
