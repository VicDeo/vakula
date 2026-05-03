package agent

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

const (
	// Gemini tends to trim the response by default so we tweak the tokens
	geminiMaxTokens = 8129
)

// Config is a model config.
type Config struct {
	Provider     string // "openai", "gemini"
	ModelName    string
	SystemPrompt string
}

// Create is an agent runner factory.
// It creates the agent runner instance based on the provided Config and Tools.
func Create(ctx context.Context, cfg Config, tools []tools.Tool) (Agent, error) {
	var model llms.Model
	var err error

	switch cfg.Provider {
	case "openai":
		model, err = openai.New(openai.WithModel(cfg.ModelName))
	case "gemini":
		model, err = googleai.New(ctx, googleai.WithDefaultModel(cfg.ModelName), googleai.WithDefaultMaxTokens(geminiMaxTokens))
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	return NewLangChainAgent(model, tools, cfg.SystemPrompt), nil
}
