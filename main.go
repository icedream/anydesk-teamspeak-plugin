package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/icedream/go-ts3plugin"
	"github.com/icedream/go-ts3plugin/teamspeak"
	"github.com/sethvargo/go-password/password"
)

var rxAnyDeskID = regexp.MustCompile(`(?i)\b(\d{9,10}|[a-z0-9\.\-_]+@ad(/[a-z0-9/\.\-_]+)?)\b`)

const (
	Name        = "AnyDesk for TeamSpeak"
	Author      = "Carl Kittelberger"
	Version     = "0.0.0"
	Description = "Converts AnyDesk meeting IDs to clickable links."
)

func init() {
	ts3plugin.Author = Author
	ts3plugin.Description = Description
	ts3plugin.Name = Name
	ts3plugin.Version = Version

	var anydesk *AnydeskCommandLineInterface

	ts3plugin.Init = func() (ok bool) {
		anydesk = NewAnydeskCommandLineInterface("")
		ok = true
		return
	}

	ts3plugin.OnTextMessageEvent = func(serverConnectionHandlerID uint64, targetMode, toID, fromID teamspeak.AnyID, fromName, fromUniqueIdentifier, message string, ffIgnored bool) int {
		for _, m := range rxAnyDeskID.FindAllString(message, -1) {
			ts3plugin.Functions().PrintMessage(serverConnectionHandlerID, fmt.Sprintf("[URL=anydesk:%s]Open %s in AnyDesk[/URL]", m, m), 1)
		}
		return 0
	}

	ts3plugin.CommandKeyword = "anydesk"
	ts3plugin.ProcessCommand = func(serverConnectionHandlerID uint64, command string) (handled bool) {
		fields := strings.Fields(command)
		command = fields[0]

		handled = true
		switch strings.ToLower(command) {
		case "invite":
			// Run in a separate goroutine to make UI not hang
			go func(serverConnectionHandlerID uint64, command string) {
				forceID := false
				withPassword := false
				for _, arg := range fields {
					switch arg {
					case "id":
						forceID = true
					case "password",
						"pass",
						"pwd",
						"pw":
						withPassword = true
					}
				}

				inviteText := ""

				// Status check
				ts3plugin.Functions().PrintMessageToCurrentTab("Now asking AnyDesk for information, that may take a few seconds…")
				status, err := anydesk.GetStatus()
				if err == ErrServiceNotRunning {
					ts3plugin.Functions().PrintMessageToCurrentTab("AnyDesk is currently not running.")
					return
				} else if err != nil {
					ts3plugin.Functions().PrintMessageToCurrentTab("AnyDesk failed while checking whether you're online. Make sure AnyDesk is running properly. Error was: " + err.Error())
					return
				}
				if status != "online" {
					ts3plugin.Functions().PrintMessageToCurrentTab(fmt.Sprintf("AnyDesk says you are %s. Make sure AnyDesk is running properly.", status))
					return
				}

				// Try and get an alias or ID we can use for the invite
				var ref string
				if !forceID {
					ref, err = anydesk.GetAlias()
					if err != nil {
						return
					}
				}
				if len(ref) <= 0 {
					ref, err = anydesk.GetID()
					if err != nil {
						ts3plugin.Functions().PrintMessage(serverConnectionHandlerID, "Can't get an alias or ID for your client. Make sure AnyDesk is running properly. Error was: "+err.Error(), 1)
						return
					}
				}
				inviteText += fmt.Sprintf("[B]AnyDesk:[/B]\n%s", ref)

				// If we want a password-based session, set a proper shareable, random password
				if withPassword {
					var pw string
					pw, err = password.Generate(12, 0, 0, false, false)
					if err != nil {
						ts3plugin.Functions().PrintMessageToCurrentTab(fmt.Sprintf("Could not generate a password for the session, error was: %s", err.Error()))
						return
					}
					anydesk.SetPassword(pw)
					ts3plugin.Functions().PrintMessageToCurrentTab(fmt.Sprintf("Your password has been changed to: %s", pw))
					inviteText += fmt.Sprintf("\nPassword: %s", pw)
				}

				ts3plugin.Functions().RequestSendChannelTextMsg(serverConnectionHandlerID, inviteText, 0, "")
			}(serverConnectionHandlerID, command)
		case "unshare",
			"uninvite",
			"remove-password":
			ts3plugin.Functions().PrintMessageToCurrentTab("Removing AnyDesk password…")
			err := anydesk.RemovePassword()
			if err != nil {
				ts3plugin.Functions().PrintMessageToCurrentTab(fmt.Sprintf("Could not remove AnyDesk password, error was: %s", err.Error()))
				return
			}
			ts3plugin.Functions().PrintMessageToCurrentTab("AnyDesk password has been removed.")
		case "version":
			version, err := anydesk.Version()
			if err != nil {
				return
			}
			ts3plugin.Functions().PrintMessage(serverConnectionHandlerID, version, 1)
		default:
			handled = false
		}

		return
	}
}

// This will never be run!
func main() {
	fmt.Println("=======================================")
	fmt.Println("This is a TeamSpeak3 plugin, do not run this as a CLI application!")
	fmt.Println("Args were: ", os.Args)
	fmt.Println("=======================================")
}
