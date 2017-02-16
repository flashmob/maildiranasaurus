package maildiranasaurus

import (
	b "github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/envelope"
	"github.com/flashmob/go-maildir"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func init() {
	// example, instead of using UseMailDir()
	//b.Service.AddProcessor("MailDir", maildirProcessor)
}

// Call UseMailDir to "import" the maildir processor into your program.
// alternatively, use the init() way above
func UseMailDir() {
	b.Service.AddProcessor("MailDir", maildirProcessor)
}

type maildirConfig struct {
	Path    string `json:"maildir_path"`
	UserMap string `json:"maildir_user_map"`
}

var maildirProcessor = func() b.Decorator {

	// The following initialization is run when the program first starts

	// config will be populated by the initFunc
	var (
		config  *maildirConfig
		userMap map[string][]int
	)
	mailDirs := make(map[string]*maildir.Maildir, 1)
	// initFunc is an initializer function which is called when our processor gets created.
	// It gets called for every worker
	initFunc := b.Initialize(func(backendConfig b.BackendConfig) error {
		configType := b.BaseConfig(&maildirConfig{})
		bcfg, err := b.Service.ExtractConfig(backendConfig, configType)
		if err != nil {
			return err
		}
		config = bcfg.(*maildirConfig)

		if strings.Index(config.Path, "~/") == 0 {
			// expand the ~/ to home dir
			usr, err := user.Current()
			if err != nil {
				return err
			}
			config.Path = usr.HomeDir + config.Path[1:]
		}

		if err != nil {
			return err
		}
		userMap = usermap(config.UserMap)
		// initialize some maildirs
		for str, ids := range userMap {
			path := strings.Replace(config.Path, "[user]", str, 1)
			if mdir, err := maildir.NewWithPerm(path, true, 0600, ids[0], ids[1]); err == nil {
				mailDirs[str] = mdir
			}

		}
		return nil
	})
	// register our initializer
	b.Service.AddInitializer(initFunc)

	return func(c b.Processor) b.Processor {
		// The function will be called on each email.
		// On success, it forwards to the next step in the processor call-stack,
		// or returns with an error if failed
		return b.ProcessorFunc(func(e *envelope.Envelope) (b.BackendResult, error) {

			// using a for loop in Go is all the range these days.
			for i := range e.RcptTo {
				u := strings.ToLower(e.RcptTo[i].User)

				// get the cached maildir
				mdir, ok := mailDirs[u]
				if !ok {
					usr, ok := userMap[u]
					if !ok {
						continue
						// no such user
						//return b.NewBackendResult(fmt.Sprintf("554 Error: no such user [%s]", u)), errors.New("no such user " + u)
					}

					b.Log().Infof("uid %d guid %d", usr[0], usr[1])
					path := strings.Replace(config.Path, "[user]", u, 1)
					if info, infoErr := os.Stat(path); infoErr != nil {
						b.Log().WithError(infoErr).Error("Cannot reach user's directory")
						//return b.NewBackendResult(fmt.Sprintf("554 Error: %s", infoErr.Error())), infoErr
						continue
					} else {

						b.Log().Info(info)
					}
					mdir, err := maildir.NewWithPerm(path, true, 0600, usr[0], usr[1])
					if err != nil {
						//return b.NewBackendResult(fmt.Sprintf("554 Error: %s", err.Error())), err
						continue
					}
					// cache it for later
					mailDirs[u] = mdir
				}

				if filename, err := mdir.CreateMail(e.NewReader()); err != nil {
					b.Log().WithError(err).Info("Could not save email")
					//return b.NewBackendResult(fmt.Sprintf("554 Error: %s", err.Error())), err
				} else {
					b.Log().Info("saved email as", filename)
				}

			}


			// continue to the next Processor in the decorator chain
			return c.Process(e)
		})
	}
}

// usermap parses the usermap config strings and returns the result in a map
// Example: "test=1002:2003,guerrilla=1001:1001"
// test and guerrilla are usernames
// number 1002 is the uid, 2003 is gid
func usermap(usermap string) (ret map[string][]int) {
	ret = make(map[string][]int, 0)
	users := strings.Split(usermap, ",")
	for i := range users {
		u := strings.Split(users[i], "=")
		if len(u) != 2 {
			return
		}
		ids := strings.Split(u[1], ":")
		if len(ids) != 2 {
			return
		}
		n := make([]int, 0)
		ret[u[0]] = n
		for k := range ids {
			s, _ := strconv.Atoi(ids[k])
			ret[u[0]] = append(ret[u[0]], s)
		}
	}
	return
}
