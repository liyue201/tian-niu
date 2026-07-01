package agent

const (
	EventError      = "error"
	EventReasoning  = "reasoning"
	EventContent    = "content"
	EventToolCall   = "tool_call"
	EventToolResult = "tool_result"
)

// StreamEvent is the event type for internal streaming output from the agent, independent of the transport layer
type StreamEvent struct {
	Event            string
	Content          string
	ReasoningContent string
	ToolCall         string
	ToolArguments    string
	ToolResult       string
}
