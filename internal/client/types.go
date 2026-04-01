package client

// AgentPool represents a LightRun agent pool.
type AgentPool struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	CreatedBy       string `json:"createdBy"`
	CreatedDate     string `json:"createdDate"`
	LiveAgentsCount int    `json:"liveAgentsCount"`
	AgentsEnabled   bool   `json:"agentsEnabled"`
}

// Agent represents a LightRun agent.
type Agent struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Version       string `json:"version"`
	VersionStatus string `json:"versionStatus"`
}

// ActionSource identifies what a LightRun action is attached to.
type ActionSource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Action represents a LightRun action (snapshot, log, counter).
type Action struct {
	ID         string       `json:"id"`
	ActionType string       `json:"actionType"`
	Source     ActionSource `json:"source"`
	Location   string       `json:"location"`
	Line       int          `json:"line"`
	Status     string       `json:"status"`
	CreatedAt  string       `json:"createdAt"`
	CreatedBy  string       `json:"createdBy"`
	Enabled    bool         `json:"enabled"`
	Expired    bool         `json:"expired"`
	HasErrors  bool         `json:"hasErrors"`
}

// PagedResponse is a generic paged API response.
type PagedResponse[T any] struct {
	Items   []T  `json:"items"`
	Total   int  `json:"total"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"hasMore"`
}

// CreateActionRequest is the public API for creating an action.
type CreateActionRequest struct {
	AgentID     string `json:"agentId"`
	ActionType  string `json:"actionType"`
	Location    string `json:"location"` // filename, e.g. "Foo.java"
	Line        int    `json:"line"`
	Condition   string `json:"condition,omitempty"`
	ExpireSecs  int    `json:"expireSecs,omitempty"`
	MaxHitCount int    `json:"maxHitCount,omitempty"`
}

// captureActionExtensionDTO is the wire-format extension block required by insertCapture.
type captureActionExtensionDTO struct {
	WatchExpressions   []interface{}          `json:"watchExpressions"`
	ContextExpressions map[string]interface{} `json:"contextExpressions"`
	Expressions        []interface{}          `json:"expressions"`
}

// insertCaptureBody is the wire-format body for POST /athena/.../insertCapture/{agentId}.
type insertCaptureBody struct {
	ActionType                  string                    `json:"actionType"`
	Filename                    string                    `json:"filename"`
	Line                        int                       `json:"line"`
	Column                      int                       `json:"column"`
	Condition                   string                    `json:"condition"`
	AgentID                     string                    `json:"agentId"`
	AgentPoolID                 string                    `json:"agentPoolId"`
	CaptureActionExtensionDTO   captureActionExtensionDTO `json:"captureActionExtensionDTO"`
	ExpirationSeconds           int                       `json:"expirationSeconds"`
	Disabled                    bool                      `json:"disabled"`
	IgnoreQuota                 bool                      `json:"ignoreQuota"`
	MaxHitCount                 int                       `json:"maxHitCount"`
	BreakpointMaxHitCount       int                       `json:"breakpointMaxHitCount"`
	IsMaxHitCountDistributed    bool                      `json:"isMaxHitCountDistributed"`
}

// insertCaptureResponse is the response from POST /athena/.../insertCapture/{agentId}.
type insertCaptureResponse struct {
	Status     string `json:"status"`
	StatusCode string `json:"statusCode"`
	ID         string `json:"id"`
}

// accountResponse is a partial /api/account response used to discover the company ID.
type accountResponse struct {
	Company struct {
		ID string `json:"id"`
	} `json:"company"`
}
