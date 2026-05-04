// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	pluginapilicense "github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/locale"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
)

// Handler handles commands
type Command struct {
	Engine    engine.Engine
	Context   *plugin.Context
	Args      *model.CommandArgs
	Config    *config.Config
	ChannelID string
	I18n      *mmi18n.Bundle
}

// T localizes a string for the invoking user's locale.
func (c *Command) T(id, defaultOther string, data map[string]any) string {
	return locale.User(c.I18n, c.Args.UserId, id, defaultOther, data)
}

func (c *Command) notConnectedText() string {
	connectPath := "/oauth2/connect"
	if config.Provider.Features.PasswordAuth {
		connectPath = "/caldav/connect"
	}
	connectURL := strings.TrimRight(c.Config.PluginURL, "/") + connectPath
	return c.T("ycal.cmd.not_connected",
		"It looks like your Mattermost account is not connected to a {{.DisplayName}} account. [Click here to connect your account]({{.ConnectURL}}) or use `/{{.Trigger}} connect`.",
		map[string]any{
			"DisplayName": config.Provider.DisplayName,
			"ConnectURL":  connectURL,
			"Trigger":     config.Provider.CommandTrigger,
		})
}

type handleFunc func(parameters ...string) (string, bool, error)

// localizeCmd resolves autocomplete / help strings. When bundle is nil, id is ignored and defaultOther is template-expanded.
func localizeCmd(loc func(id, defaultOther string, data map[string]any) string, id, defaultOther string) string {
	data := map[string]any{"DisplayName": config.Provider.DisplayName}
	return loc(id, defaultOther, data)
}

func getCommands(loc func(id, defaultOther string, data map[string]any) string) []*model.AutocompleteData {
	d := func(id, def string) string { return localizeCmd(loc, id, def) }

	cmds := []*model.AutocompleteData{
		model.NewAutocompleteData("connect", "", d("ycal.cmd.ac.connect", "Connect to your {{.DisplayName}} account")),
		model.NewAutocompleteData("disconnect", "", d("ycal.cmd.ac.disconnect", "Disconnect from your {{.DisplayName}} account")),
		{ // Summary
			Trigger:  "summary",
			HelpText: d("ycal.cmd.ac.summary", "View your events for today, or edit the settings for your daily summary."),
			SubCommands: []*model.AutocompleteData{
				model.NewAutocompleteData("view", "", d("ycal.cmd.ac.summary_view", "View your daily summary.")),
				model.NewAutocompleteData("today", "", d("ycal.cmd.ac.summary_today", "Display today's events.")),
				model.NewAutocompleteData("tomorrow", "", d("ycal.cmd.ac.summary_tomorrow", "Display tomorrow's events.")),
				model.NewAutocompleteData("settings", "", d("ycal.cmd.ac.summary_settings", "View your settings for the daily summary.")),
				model.NewAutocompleteData("time", "", d("ycal.cmd.ac.summary_time", "Set the time you would like to receive your daily summary.")),
				model.NewAutocompleteData("enable", "", d("ycal.cmd.ac.summary_enable", "Enable your daily summary.")),
				model.NewAutocompleteData("disable", "", d("ycal.cmd.ac.summary_disable", "Disable your daily summary.")),
			},
		},
		model.NewAutocompleteData("viewcal", "", d("ycal.cmd.ac.viewcal", "View your events for the upcoming 14 days, including today.")),
	}

	if !config.Provider.Features.HideCreateEventFromCommand {
		cmds = append(cmds, &model.AutocompleteData{
			Trigger:  "event",
			HelpText: d("ycal.cmd.ac.event", "Manage events."),
			SubCommands: []*model.AutocompleteData{
				model.NewAutocompleteData("create", "", d("ycal.cmd.ac.event_create", "Creates a new event.")),
			},
		})
	}

	cmds = append(cmds,
		model.NewAutocompleteData("today", "", d("ycal.cmd.ac.today", "Display today's events.")),
		model.NewAutocompleteData("tomorrow", "", d("ycal.cmd.ac.tomorrow", "Display tomorrow's events.")),
		model.NewAutocompleteData("settings", "", d("ycal.cmd.ac.settings", "Edit your user personal settings.")),
		model.NewAutocompleteData("info", "", d("ycal.cmd.ac.info", "Read information about this version of the plugin.")),
		model.NewAutocompleteData("help", "", d("ycal.cmd.ac.help", "Read help text for the commands")),
	)

	return cmds
}

// Register should be called by the plugin to register all necessary commands.
// bundle may be nil; English template defaults are used for autocomplete metadata.
func Register(client *pluginapilicense.Client, bundle *mmi18n.Bundle) error {
	loc := func(id, def string, data map[string]any) string {
		return locale.Server(bundle, id, def, data)
	}
	cmds := getCommands(loc)

	names := []string{}
	for _, subCommand := range cmds {
		names = append(names, subCommand.Trigger)
	}

	hint := "[" + strings.Join(names[:4], "|") + "...]"

	slashHelp := localizeCmd(loc, "ycal.cmd.register.slash_help", "Interact with your {{.DisplayName}} calendar.")
	cmd := model.NewAutocompleteData(config.Provider.CommandTrigger, hint, slashHelp)
	cmd.SubCommands = cmds

	iconData, err := command.GetIconData(&client.System, fmt.Sprintf("assets/profile-%s.svg", config.Provider.Name))
	if err != nil {
		return errors.Wrap(err, "failed to get icon data")
	}

	return client.SlashCommand.Register(&model.Command{
		Trigger:              config.Provider.CommandTrigger,
		DisplayName:          config.Provider.DisplayName,
		Description:          slashHelp,
		AutoComplete:         true,
		AutoCompleteDesc:     strings.Join(names, ", "),
		AutoCompleteHint:     "(subcommand)",
		AutocompleteData:     cmd,
		AutocompleteIconData: iconData,
	})
}

// Handle should be called by the plugin when a command invocation is received from the Mattermost server.
func (c *Command) Handle() (string, bool, error) {
	cmd, parameters, err := c.isValid()
	if err != nil {
		return "", false, err
	}

	handler := c.help
	switch cmd {
	case "info":
		handler = c.info
	case "connect":
		handler = c.connect
	case "disconnect":
		handler = c.requireConnectedUser(c.disconnect)
	case "summary":
		handler = c.requireConnectedUser(c.dailySummary)
	case "viewcal":
		handler = c.requireConnectedUser(c.viewCalendar)
	case "settings":
		handler = c.requireConnectedUser(c.settings)
	case "event":
		if !config.Provider.Features.HideCreateEventFromCommand {
			handler = c.requireConnectedUser(c.event)
		}
	// Admin only
	case "showcals":
		handler = c.requireConnectedUser(c.requireAdminUser(c.showCalendars))
	case "avail":
		handler = c.requireConnectedUser(c.requireAdminUser(c.debugAvailability))
	case "subscribe":
		handler = c.requireConnectedUser(c.requireAdminUser(c.subscribe))
	case "unsubscribe":
		handler = c.requireConnectedUser(c.requireAdminUser(c.unsubscribe))
	// Aliases
	case "today":
		parameters = []string{"today"}
		handler = c.requireConnectedUser(c.dailySummary)
	case "tomorrow":
		parameters = []string{"tomorrow"}
		handler = c.requireConnectedUser(c.dailySummary)
	}
	out, mustRedirectToDM, err := handler(parameters...)
	if err != nil {
		return out, false, errors.WithMessagef(err, "Command /%s %s failed", config.Provider.CommandTrigger, cmd)
	}

	return out, mustRedirectToDM, nil
}

func (c *Command) isValid() (subcommand string, parameters []string, err error) {
	if c.Context == nil || c.Args == nil {
		return "", nil, errors.New("invalid arguments to command.Handler")
	}
	split := strings.Fields(c.Args.Command)
	cmd := split[0]
	if cmd != "/"+config.Provider.CommandTrigger {
		return "", nil, fmt.Errorf("%q is not a supported command. Please contact your system administrator", cmd)
	}

	parameters = []string{}
	subcommand = ""
	if len(split) > 1 {
		subcommand = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	return subcommand, parameters, nil
}

func (c *Command) user() *engine.User {
	return engine.NewUser(c.Args.UserId)
}

func (c *Command) requireConnectedUser(handle handleFunc) handleFunc {
	return func(parameters ...string) (string, bool, error) {
		connected, err := c.isConnected()
		if err != nil {
			return "", false, err
		}

		if !connected {
			return c.notConnectedText(), false, nil
		}
		return handle(parameters...)
	}
}

func (c *Command) requireAdminUser(handle handleFunc) handleFunc {
	return func(parameters ...string) (string, bool, error) {
		authorized, err := c.Engine.IsAuthorizedAdmin(c.Args.UserId)
		if err != nil {
			return "", false, err
		}
		if !authorized {
			return c.T("ycal.cmd.not_authorized", "Not authorized", nil), false, nil
		}

		return handle(parameters...)
	}
}

func (c *Command) isConnected() (bool, error) {
	_, err := c.Engine.GetRemoteUser(c.Args.UserId)
	if err == store.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
