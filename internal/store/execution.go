package store

import "encoding/json"

// Execution is a request definition plus the placeholders already resolved.
// It is stored in history so past runs can be replayed exactly.
type Execution struct {
	Request Request           `json:"request"`
	Vars    map[string]string `json:"vars"`
	EnvName string            `json:"envName"`
}

// MarshalSnapshot serializes the execution for history storage.
func (e Execution) MarshalSnapshot() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// ParseSnapshot rebuilds an execution from history.
func ParseSnapshot(raw string) (Execution, bool) {
	if raw == "" {
		return Execution{}, false
	}
	var e Execution
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return Execution{}, false
	}
	if e.Request.ID == "" && e.Request.URL == "" {
		return Execution{}, false
	}
	return e, true
}
