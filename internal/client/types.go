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

// CreateActionRequest is the payload for creating an action.
type CreateActionRequest struct {
	AgentID     string `json:"agentId"`
	ActionType  string `json:"actionType"`
	Location    string `json:"location"`
	Line        int    `json:"line"`
	Condition   string `json:"condition,omitempty"`
	ExpireSecs  int    `json:"expireSecs,omitempty"`
	MaxHitCount int    `json:"maxHitCount,omitempty"`
}
