package maildiranasaurus

import (
	b "github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/envelope"
	"github.com/flashmob/go-maildir"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"os"
	"time"
	"errors"
	"fmt"
)

const MailDirFilePerms = 0600

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
	// maildir_path may contain a [user] placeholder. This will be substituted at run time
	// eg /home/[user]/Maildir will get substituted to /home/test/Maildir for test@example.com
	Path    string `json:"maildir_path"`
	// This is a string holding user to group/id mappings - in other words, the recipient table
	// Each record separated by ","
	// Records have the following format: <username>=<id>:<group>
	// Example: "test=1002:2003,guerrilla=1001:1001"
	UserMap string `json:"maildir_user_map"`
}

type MailDir struct {
	userMap map[string][]int
	dirs map[string]*maildir.Maildir
	config  *maildirConfig
}

// check to see if we have configured
func (m *MailDir) checkUsers(rcpt []envelope.EmailAddress, mailDirs map[string]*maildir.Maildir) bool {
	for i:=range rcpt {
		if _ , ok := mailDirs[rcpt[i].User]; !ok {
			return false
		}
	}
	return true
}

var mdirMux sync.Mutex

// initDirs creates the mail dir folders if they haven't been created already
func (m *MailDir) initDirs() error {
	if m.dirs == nil {
		m.dirs = make (map[string]*maildir.Maildir, 0)
	}
	// initialize some maildirs
	mdirMux.Lock()
	defer mdirMux.Unlock()
	for str, ids := range m.userMap {
		path := strings.Replace(m.config.Path, "[user]", str, 1)
		if mdir, err := maildir.NewWithPerm(path, true, MailDirFilePerms, ids[0], ids[1]); err == nil {
			m.dirs[str] = mdir
		} else {
			b.Log().WithError(err).Error("could not create Maildir. Please check the config")
			return err
		}
	}
	return nil
}

// validateRcpt validates if the user u is valid
// not currently used, accepting pull requests for those who want to get their hands dirty ;-)
func (m *MailDir) validateRcpt(u string) bool {

	mdir , ok := m.dirs[u]
	if !ok {
		return false
	}
	if _, err := os.Stat(mdir.Path); err != nil {
		return false;
	} else {
		// TDOD not sure of another way of testing to see if the directory is writable
		test := mdir.Path + "/test123" + string(time.Now().UnixNano())
		if fd, err := os.Create(test); err != nil {
			return false
		} else {
			fd.Close()
			os.Remove(test)
		}
	}
	return ok

}

func newMailDir(config *maildirConfig) (*MailDir, error) {
	m := &MailDir{}
	m.config = config
	m.userMap = usermap(m.config.UserMap)
	if strings.Index(m.config.Path, "~/") == 0 {
		// expand the ~/ to home dir
		usr, err := user.Current()
		if err != nil {
			b.Log().WithError(err).Error("could not expand ~/ to homedir")
			return nil, err
		}
		m.config.Path = usr.HomeDir + m.config.Path[1:]
	}
	if err := m.initDirs(); err != nil {
		return nil, err
	}
	return m, nil
}

var maildirProcessor = func() b.Decorator {

	// The following initialization is run when the program first starts

	// config will be populated by the initFunc
	var (
		m *MailDir
	)
	// initFunc is an initializer function which is called when our processor gets created.
	// It gets called for every worker
	initFunc := b.Initialize(func(backendConfig b.BackendConfig) error {
		configType := b.BaseConfig(&maildirConfig{})
		bcfg, err := b.Service.ExtractConfig(backendConfig, configType)
		if err != nil {
			return err
		}
		c := bcfg.(*maildirConfig)
		m, err = newMailDir(c);
		if  err!= nil {
			return err
		}
		return nil
	})
	// register our initializer
	b.Service.AddInitializer(initFunc)

	// Todo would be great if to add it as a new Service, so the SMTP server could ask the backend
	// the backend could call all the validators to ask if user is valid..
	//b.Service.AddRcptValidator(rcptValidate)

	return func(c b.Processor) b.Processor {
		// The function will be called on each email transaction.
		// On success, it forwards to the next step in the processor call-stack,
		// or returns with an error if failed
		return b.ProcessorFunc(func(e *envelope.Envelope) (b.BackendResult, error) {
			// check the recipients
			for i := range e.RcptTo {
				u := strings.ToLower(e.RcptTo[i].User)
				if _, ok := m.dirs[u]; !ok {
					err := errors.New("no such user: "+u)
					b.Log().WithError(err).Info("recipient not configured: ", u)
					return b.NewBackendResult(fmt.Sprintf("554 Error: %s", err)), err
				}
			}
			for i := range e.RcptTo {
				u := strings.ToLower(e.RcptTo[i].User)
				mdir, ok := m.dirs[u]
				if !ok {
					// no such user
					continue
				}
				if filename, err := mdir.CreateMail(e.NewReader()); err != nil {
					b.Log().WithError(err).Error("Could not save email")
					if i ==0 {
						// todo need a better to check if the directory has write perms
						// to catch the error early.
						return b.NewBackendResult(fmt.Sprintf("554 Error: could not save email for [%s]", u)), err
					}
				} else {
					b.Log().Debug("saved email as", filename)
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
