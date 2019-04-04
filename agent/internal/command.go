package internal

import (
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
)

type CommandManager struct {
	logger   *plog.Logger
	agent    CommandManagerAgent
	handlers map[string]CommandHandler
}

type CommandHandler func() api.CommandResult

// CommandManagerAgent defines the expected agent SDK and allows to easily
// implement functional tests by mocking it up.
type CommandManagerAgent interface {
	InstrumentationEnable() error
	InstrumentationDisable() error
	ActionsReload() error
}

func NewCommandManager(agent CommandManagerAgent, logger *plog.Logger) *CommandManager {
	mng := &CommandManager{
		agent:  agent,
		logger: plog.NewLogger("command", logger),
	}

	// Note: using Go's reflection to call methods would be slower.
	mng.handlers = map[string]CommandHandler{
		"instrumentation_enable": mng.InstrumentationEnable,
		"instrumentation_remove": mng.InstrumentationRemove,
		"actions_reload":         mng.ActionsReload,
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
				// The command has not been done
				result = handler()
				// Set this command as done by storing the uuid that performed it
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

	if len(results) == 0 {
		return nil
	}
	return results
}

func (m *CommandManager) InstrumentationEnable() api.CommandResult {
	err := m.agent.InstrumentationEnable()
	return commandResult(err)
}

func (m *CommandManager) InstrumentationRemove() api.CommandResult {
	err := m.agent.InstrumentationDisable()
	return commandResult(err)
}

func (m *CommandManager) ActionsReload() api.CommandResult {
	err := m.agent.ActionsReload()
	return commandResult(err)
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
