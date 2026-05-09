package productgraph

import "time"

// EventType represents the type of ProductGraph event.
type EventType string

const (
	// Page events
	EventTypePageView  EventType = "page.view"
	EventTypePageLeave EventType = "page.leave"

	// UI events
	EventTypeUIClick  EventType = "ui.click"
	EventTypeUIInput  EventType = "ui.input"
	EventTypeUIScroll EventType = "ui.scroll"
	EventTypeUIFocus  EventType = "ui.focus"
	EventTypeUIBlur   EventType = "ui.blur"
	EventTypeUISubmit EventType = "ui.submit"

	// State events
	EventTypeStateChange EventType = "state.change"

	// API events
	EventTypeAPIRequest  EventType = "api.request"
	EventTypeAPIResponse EventType = "api.response"

	// Journey events
	EventTypeJourneyStep EventType = "journey.step"

	// Error events
	EventTypeError EventType = "error"

	// Performance events
	EventTypePerformance EventType = "performance"

	// Custom events
	EventTypeCustom EventType = "custom"
)

// Event represents a ProductGraph event following OTel semantic conventions.
type Event struct {
	// Identity
	EventID   string `json:"event_id"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session.id"`
	UserID    string `json:"user.id,omitempty"`

	// Event classification
	EventType EventType `json:"event.type"`
	EventName string    `json:"event.name,omitempty"`
	Timestamp string    `json:"event.timestamp"`
	Sequence  int       `json:"event.sequence,omitempty"`

	// Page context
	PagePath     string `json:"page.path,omitempty"`
	PageTitle    string `json:"page.title,omitempty"`
	PageURL      string `json:"page.url,omitempty"`
	PageReferrer string `json:"page.referrer,omitempty"`

	// UI context
	UIComponentName string `json:"ui.component.name,omitempty"`
	UIComponentPath string `json:"ui.component.path,omitempty"`
	UIComponentType string `json:"ui.component.type,omitempty"`
	UIAction        string `json:"ui.action,omitempty"`
	UIElement       string `json:"ui.element,omitempty"`
	UIElementText   string `json:"ui.element.text,omitempty"`

	// State changes
	UIStateKey    string `json:"ui.state.key,omitempty"`
	UIStateBefore string `json:"ui.state.before,omitempty"`
	UIStateAfter  string `json:"ui.state.after,omitempty"`

	// Journey tracking
	JourneyID   string `json:"gen_ai.journey.id,omitempty"`
	JourneyStep string `json:"gen_ai.journey.step.id,omitempty"`
	JourneyName string `json:"gen_ai.journey.step.name,omitempty"`

	// API tracking
	APIMethod     string `json:"api.method,omitempty"`
	APIPath       string `json:"api.path,omitempty"`
	APIStatusCode int    `json:"api.status_code,omitempty"`
	APIDurationMs int    `json:"api.duration_ms,omitempty"`

	// Error tracking
	ErrorType    string `json:"error.type,omitempty"`
	ErrorMessage string `json:"error.message,omitempty"`
	ErrorStack   string `json:"error.stack,omitempty"`

	// Performance
	DurationMs int `json:"duration_ms,omitempty"`

	// Organization
	OrgID   string `json:"org.id,omitempty"`
	OrgName string `json:"org.name,omitempty"`

	// Custom metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewEvent creates a new Event with required fields populated.
func NewEvent(eventType EventType) Event {
	return Event{
		EventType: eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// APIResponseEvent creates an event for tracking an API response.
func APIResponseEvent(method, path string, statusCode int, durationMs int) Event {
	return Event{
		EventType:     EventTypeAPIResponse,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		APIMethod:     method,
		APIPath:       path,
		APIStatusCode: statusCode,
		APIDurationMs: durationMs,
	}
}

// ErrorEvent creates an event for tracking an error.
func ErrorEvent(errType, message string) Event {
	return Event{
		EventType:    EventTypeError,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		ErrorType:    errType,
		ErrorMessage: message,
	}
}

// JourneyStepEvent creates an event for tracking a journey step.
func JourneyStepEvent(journeyID, stepID, stepName string) Event {
	return Event{
		EventType:   EventTypeJourneyStep,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		JourneyID:   journeyID,
		JourneyStep: stepID,
		JourneyName: stepName,
	}
}

// Payload represents the request body for POST /v1/events.
type Payload struct {
	Events []Event `json:"events"`
}

// Response represents the response from POST /v1/events.
type Response struct {
	Accepted int            `json:"accepted"`
	Rejected int            `json:"rejected"`
	Errors   []PayloadError `json:"errors,omitempty"`
}

// PayloadError represents an error for a specific event in a batch.
type PayloadError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}
