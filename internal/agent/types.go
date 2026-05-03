package agent

import "context"



// Agent is an agent interface.
// Agent produces string output based on the string input.
// That's it.
type Agent interface {
	Execute(ctx context.Context, input string) (string, error)
}
