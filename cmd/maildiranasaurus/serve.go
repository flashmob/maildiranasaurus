/*

                 _..--+~/@-~--.
             _-=~      (  .   "}
          _-~     _.--=.\ \""""
        _~      _-       \ \_\
       =      _=          '--'
      '      =                             .
     :      :       ____                   '=_. ___
___  |      ;                            ____ '~--.~.
     ;      ;                               _____  } |
  ___=       \ ___ __     __..-...__           ___/__/__
     :        =_     _.-~~          ~~--.__
_____ \         ~-+-~                   ___~=_______
     ~@#~~ == ...______ __ ___ _--~~--_
                                                    .=
Art by Peter Weighill

*/

package main

import (
	"errors"
	"github.com/flashmob/fastcgi-processor"
	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/log"
	"github.com/flashmob/maildir-processor"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const (
	defaultPidFile = "/var/run/maildiranasaurus.pid"
)

var (
	configPath string
	pidFile    string

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "start the small SMTP server",
		Run:   serve,
	}


	signalChannel = make(chan os.Signal, 1) // for trapping SIG_HUP
	mainlog       log.Logger

	d guerrilla.Daemon
)

func init() {
	// log to stderr on startup
	var logOpenError error
	if mainlog, logOpenError = log.GetLogger(log.OutputStderr.String()); logOpenError != nil {
		mainlog.WithError(logOpenError).Errorf("Failed creating a logger to %s", log.OutputStderr)
	}
	serveCmd.PersistentFlags().StringVarP(&configPath, "config", "c",
		"maildiranasaurus.conf", "Path to the configuration file")
	// intentionally didn't specify default pidFile; value from config is used if flag is empty
	serveCmd.PersistentFlags().StringVarP(&pidFile, "pidFile", "p",
		"", "Path to the pid file")

	rootCmd.AddCommand(serveCmd)

	d = guerrilla.Daemon{Logger : mainlog}

	// add the Processor to be identified as "MailDir"
	backends.Svc.AddProcessor("MailDir", maildir_processor.Processor)

	backends.Svc.AddProcessor("FastCGI", fcgi_processor.Processor)
}

func sigHandler() {
	// handle SIGHUP for reloading the configuration while running
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL)

	for sig := range signalChannel {
		if sig == syscall.SIGHUP {
			d.ReloadConfig(configPath)
		} else if sig == syscall.SIGUSR1 {
			d.ReopenLogs()
		} else if sig == syscall.SIGTERM || sig == syscall.SIGQUIT || sig == syscall.SIGINT {
			mainlog.Infof("Shutdown signal caught")
			d.Shutdown()
			mainlog.Infof("Shutdown completed, exiting.")
			return
		} else {
			mainlog.Infof("Shutdown, unknown signal caught")
			return
		}
	}
}


func serve(cmd *cobra.Command, args []string) {
	logVersion()
	err := readConfig(configPath, pidFile)
	if err != nil {
		mainlog.WithError(err).Fatal("Error while reading config")
	}
	// Check that max clients is not greater than system open file limit.
	fileLimit := getFileLimit()
	if fileLimit > 0 {
		maxClients := 0
		for _, s := range d.Config.Servers {
			maxClients += s.MaxClients
		}
		if maxClients > fileLimit {
			mainlog.Fatalf("Combined max clients for all servers (%d) is greater than open file limit (%d). "+
				"Please increase your open file limit or decrease max clients.", maxClients, fileLimit)
		}
	}

	err = d.Start()
	if err != nil {
		mainlog.WithError(err).Error("Error(s) when starting server(s)")
	}

	// write out our pid whenever the file name changes in the config
	d.Subscribe(guerrilla.EventConfigPidFile, func(ac *guerrilla.AppConfig) {
		d.WritePid()
	})

	if err := d.ChangeLog(); err == nil {
		mainlog.Infof("main log configured to %s", d.Config.LogFile)
	}

	sigHandler()
}

// Superset of `guerrilla.AppConfig` containing options specific
// the the command line interface.
type CmdConfig struct {
	guerrilla.AppConfig
}



func (c *CmdConfig) emitChangeEvents(oldConfig *CmdConfig, app guerrilla.Guerrilla) {
	// if your CmdConfig has any extra fields, you can emit events here
	// ...

	// call other emitChangeEvents
	c.AppConfig.EmitChangeEvents(&oldConfig.AppConfig, app)
}

// ReadConfig which should be called at startup, or when a SIG_HUP is caught
func readConfig(path string, pidFile string) error {

	if err := d.ReadConfig(path); err != nil {
		return err
	}

	// override config pidFile with with flag from the command line
	if len(pidFile) > 0 {
		d.Config.PidFile = pidFile
	} else if len(d.Config.PidFile) == 0 {
		d.Config.PidFile = defaultPidFile
	}

	if len(d.Config.AllowedHosts) == 0 {
		return errors.New("Empty `allowed_hosts` is not allowed")
	}
	return nil
}

func getFileLimit() int {
	cmd := exec.Command("ulimit", "-n")
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return -1
	}
	return limit
}


