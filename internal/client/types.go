package client

// Agent represents a LightRun agent.
type Agent struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Host        string `json:"host"`
	Status      string `json:"status"`
	Tags        []Tag  `json:"tags"`
}

// Tag represents a tag attached to an agent.
type Tag struct {
	Name string `json:"name"`
}

// Action represents a LightRun action (snapshot, log, counter).
type Action struct {
	ID         string `json:"id"`
	Type       string `json:"actionType"`
	AgentID    string `json:"agentId"`
	FileName   string `json:"fileName"`
	LineNumber int    `json:"lineNumber"`
	Status     string `json:"status"`
	CreateTime int64  `json:"createTime"`
	Condition  string `json:"condition,omitempty"`
}

// SnapshotData holds data captured when a snapshot fires.
type SnapshotData struct {
	ActionID  string     `json:"actionId"`
	Variables []Variable `json:"variables"`
	Frames    []Frame    `json:"frames"`
}

// Variable represents a captured variable.
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Frame represents a stack frame in a snapshot.
type Frame struct {
	FileName   string     `json:"fileName"`
	MethodName string     `json:"methodName"`
	LineNumber int        `json:"lineNumber"`
	Variables  []Variable `json:"locals"`
}

// PagedResponse is a generic paged API response.
type PagedResponse[T any] struct {
	Data       []T `json:"data"`
	TotalCount int `json:"totalCount"`
	PageCount  int `json:"pageCount"`
}

// CreateActionRequest is the payload for creating an action.
type CreateActionRequest struct {
	AgentID     string `json:"agentId"`
	Type        string `json:"type"`
	FileName    string `json:"fileName"`
	LineNumber  int    `json:"lineNumber"`
	Condition   string `json:"condition,omitempty"`
	ExpireSecs  int    `json:"expireSecs,omitempty"`
	MaxHitCount int    `json:"maxHitCount,omitempty"`
}
