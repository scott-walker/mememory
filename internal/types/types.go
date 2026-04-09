package types

import "time"

type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

type MemoryType string

const (
	TypeFact     MemoryType = "fact"
	TypeRule     MemoryType = "rule"
	TypeDecision MemoryType = "decision"
	TypeFeedback MemoryType = "feedback"
	TypeContext  MemoryType = "context"
)

type Delivery string

const (
	DeliveryBootstrap Delivery = "bootstrap"
	DeliveryOnDemand  Delivery = "on_demand"
)

type Memory struct {
	ID         string     `json:"id"`
	Content    string     `json:"content"`
	Scope      Scope      `json:"scope"`
	Project    string     `json:"project,omitempty"`
	Type       MemoryType `json:"type"`
	Delivery   Delivery   `json:"delivery"`
	Tags       []string   `json:"tags,omitempty"`
	Weight     float64    `json:"weight"`
	Supersedes string     `json:"supersedes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	TTL        *time.Time `json:"ttl,omitempty"`
}

type RememberInput struct {
	Content    string     `json:"content"`
	Scope      Scope      `json:"scope"`
	Project    string     `json:"project,omitempty"`
	Type       MemoryType `json:"type"`
	Delivery   Delivery   `json:"delivery,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	Weight     float64    `json:"weight,omitempty"`
	Supersedes string     `json:"supersedes,omitempty"`
	TTL        string     `json:"ttl,omitempty"` // duration string, e.g. "24h"
}

// RememberResult is returned by Remember() — includes stored memory and any contradiction warnings.
type RememberResult struct {
	Memory         *Memory              `json:"memory"`
	Contradictions []ContradictionMatch `json:"contradictions,omitempty"`
}

// ContradictionMatch represents a potentially conflicting existing memory.
type ContradictionMatch struct {
	Memory     Memory  `json:"memory"`
	Similarity float32 `json:"similarity"`
}

type RecallInput struct {
	Query   string `json:"query"`
	Scope   string `json:"scope,omitempty"`
	Project string `json:"project,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type RecallResult struct {
	Memory Memory  `json:"memory"`
	Score  float32 `json:"score"`
}

type ListInput struct {
	Scope    string `json:"scope,omitempty"`
	Project  string `json:"project,omitempty"`
	Type     string `json:"type,omitempty"`
	Delivery string `json:"delivery,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type StatsResult struct {
	Total      uint64            `json:"total"`
	ByScope    map[string]uint64 `json:"by_scope"`
	ByProject  map[string]uint64 `json:"by_project"`
	ByType     map[string]uint64 `json:"by_type"`
	ByDelivery map[string]uint64 `json:"by_delivery"`
}
