package engine

import t "github.com/scott-walker/mememory/internal/types"

// Re-export types for backward compatibility with MCP tools and API
type (
	Memory             = t.Memory
	Scope              = t.Scope
	MemoryType         = t.MemoryType
	Delivery           = t.Delivery
	RememberInput      = t.RememberInput
	RememberResult     = t.RememberResult
	ContradictionMatch = t.ContradictionMatch
	RecallInput        = t.RecallInput
	RecallResult       = t.RecallResult
	ListInput          = t.ListInput
	StatsResult        = t.StatsResult
)

const (
	ScopeGlobal  = t.ScopeGlobal
	ScopeProject = t.ScopeProject
	TypeFact     = t.TypeFact
	TypeRule     = t.TypeRule
	TypeDecision = t.TypeDecision
	TypeFeedback = t.TypeFeedback
	TypeContext  = t.TypeContext

	DeliveryBootstrap = t.DeliveryBootstrap
	DeliveryOnDemand  = t.DeliveryOnDemand
	DeliveryPinned    = t.DeliveryPinned
)
