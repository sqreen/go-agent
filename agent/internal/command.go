package internal

import (
	"encoding/json"
	"fmt"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
)

type CommandManager struct {
	logger   *plog.Logger
	agent    CommandManagerAgent
	handlers map[string]CommandHandler
}

// CommandHandler is a function pointer type to a command handler.
// Command arguments need to be validated by the handler itself.
type CommandHandler func(args []json.RawMessage) error

// CommandManagerAgent defines the expected agent SDK and allows to easily
// implement functional tests by mocking it up.
type CommandManagerAgent interface {
	InstrumentationEnable() error
	InstrumentationDisable() error
	ActionsReload() error
	SetCIDRWhitelist([]string) error
}

func NewCommandManager(agent CommandManagerAgent, logger *plog.Logger) *CommandManager {
	mng := &CommandManager{
		agent:  agent,
		logger: logger,
	}

	// Note: using Go's reflection to call methods would be slower.
	mng.handlers = map[string]CommandHandler{
		"instrumentation_enable": mng.InstrumentationEnable,
		"instrumentation_remove": mng.InstrumentationRemove,
		"actions_reload":         mng.ActionsReload,
		"ips_whitelist":          mng.IPSWhitelist,
	}

	return mng
}

func (m *CommandManager) Do(commands []api.CommandRequest) map[string]api.CommandResult {
	if len(commands) == 0 {
		return nil
	}

	results := make(map[string]api.CommandResult, len(commands))
	done := make(map[string]string, len(commands))
	for _, cmd := range commands {
		handler, exists := m.handlers[cmd.Name]
		var result api.CommandResult
		if exists {
			if lastUuid := done[cmd.Name]; lastUuid == "" {
				// This command has not been done yet in this list of commands
				err := handler(cmd.Params)
				result = commandResult(err)
				// Set it as done by storing the uuid that performed it
				done[cmd.Name] = cmd.Uuid
			} else {
				// The command is already done and appears several times in the list of
				// commands. So just reuse the last result
				result = results[lastUuid]
			}
		} else {
			// The command is not in the list of supported commands
			result = api.CommandResult{
				Status: false,
				Output: config.ErrorMessage_UnsupportedCommand,
			}
		}
		results[cmd.Uuid] = result
	}

	return results
}

func (m *CommandManager) InstrumentationEnable([]json.RawMessage) error {
	return m.agent.InstrumentationEnable()
}

func (m *CommandManager) InstrumentationRemove([]json.RawMessage) error {
	return m.agent.InstrumentationDisable()
}

func (m *CommandManager) ActionsReload([]json.RawMessage) error {
	return m.agent.ActionsReload()
}

func (m *CommandManager) IPSWhitelist(args []json.RawMessage) error {
	if argc := len(args); argc != 1 {
		return fmt.Errorf("unexpected number of arguments: expected 1 argument but got %d", argc)
	}
	var cidrs []string
	arg0 := args[0]
	if err := json.Unmarshal(arg0, &cidrs); err != nil {
		return err
	}
	return m.agent.SetCIDRWhitelist(cidrs)
}

// commandResult converts an error to a command result API object.
func commandResult(err error) api.CommandResult {
	if err != nil {
		return api.CommandResult{
			Status: false,
			Output: err.Error(),
		}
	}
	return api.CommandResult{
		Status: true,
	}
}
