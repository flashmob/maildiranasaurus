package maildiranasaurus

import (
	"fmt"
	"github.com/flashmob/go-guerrilla/backends"
)

// custom configuration we will parse from the json
// see guerrillaDBAndRedisConfig struct for a more complete example
type maildirConfig struct {
	LogReceivedMails bool `json:"log_received_mails"`
}

// putting all the paces we need together
type MaildirBackend struct {
	config maildirConfig
	// embed functions form AbstractBackend so that DummyBackend satisfies the Backend interface
	backends.AbstractBackend
}

func (b *MaildirBackend) loadConfig(backendConfig backends.BackendConfig) (err error) {
	fmt.Println("hello")
	return err
}
