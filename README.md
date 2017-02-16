# Maildiranasaurus

![Dino](/dino.png)

## RoaaaarrrRRRRR!

This started as a project to get an insight into using [go-guerrilla](https://github.com/flashmob/go-guerrilla) as a package.

It is uses go-guerrilla with an Maildir backend. See [maildir.go](https://github.com/flashmob/maildiranasaurus/blob/master/maildir.go) how the Maildir processor was added!

## Building

You'll need GNU make and Go installed

     $ make maildirasaurus

## Running

copy maildiranasaurus.conf.sample to maildiranasaurus.conf
customize it to how you like it, then:

./maildiranasaurus serve

## Config

Customize your servers, and customize your backend_config like so:

    "backend_config" :
    {
        "process_stack": "HeadersParser|Debugger|Hasher|Header|MailDir",
        "maildir_user_map" : "test=1002:2003,guerrilla=1001:1001,flashmob=1000:1000",
        "maildir_path" : "/home/[user]/Maildir",
        "save_workers_size" : 1,
        "primary_mail_host":"sharklasers.com",
        "log_received_mails" : false
    },
    
`process_stack` - a line of stacked processors (decorators) that work on the email envelope. 
Working from left to right, i.e. mail will be saved using the MailDir processor

`maildir_user_map` - user settings. `<username>=<user id>:<group id>` comma separated

`maildir_path` - the `[user]` part will be replaced with with the actual user from maildir_user_map once the config is loaded. Usually, no need to change this as the default is conventional. 

`save_workers_size` - how many dinosaur workers to spawn. Roaaaar!
