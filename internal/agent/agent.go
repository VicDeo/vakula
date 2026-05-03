package agent

import (
	"context"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

const (
	maxIterations = 10
)

// LangChainAgent is an agent runner.
type LangChainAgent struct {
	executor agents.Executor
}

// NewLangChainAgent creates an agent leveraging the particular model, tools and the system prompt.
func NewLangChainAgent(model llms.Model, agentTools []tools.Tool, systemPrompt string) *LangChainAgent {
	handler := callbacks.LogHandler{}

	a := agents.NewConversationalAgent(
		model,
		agentTools,
		agents.WithPromptPrefix(systemPrompt),
		agents.WithCallbacksHandler(handler),
	)

	executor := agents.NewExecutor(
		a,
		agents.WithMaxIterations(maxIterations),
		agents.WithParserErrorHandler(agents.NewParserErrorHandler(func(s string) string {
			return "Invalid format. Please use the strictly defined ReAct format: Thought, Action, Action Input."
		})),
	)

	return &LangChainAgent{
		executor: *executor,
	}
}

// Execute executes the task.
func (l *LangChainAgent) Execute(ctx context.Context, input string) (string, error) {
	results, err := l.executor.Call(ctx, map[string]any{
		"input": input,
	})
	if err != nil {
		return "", err
	}

	output, ok := results["output"].(string)
	if !ok {
		return "no response: wrong data type", nil
	}

	return output, nil
}
