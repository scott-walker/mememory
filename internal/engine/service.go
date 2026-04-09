package engine

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/scott-walker/mememory/internal/embeddings"
	pg "github.com/scott-walker/mememory/internal/postgres"
)

const (
	contradictionThreshold float32 = 0.75
	decayLambda            float64 = 0.005
	scopeWeightProject     float64 = 1.0
	scopeWeightGlobal      float64 = 0.8
)

type Service struct {
	pg    *pg.Client
	embed embeddings.Embedder
}

func NewService(pgClient *pg.Client, embed embeddings.Embedder) *Service {
	return &Service{pg: pgClient, embed: embed}
}

func (s *Service) Remember(ctx context.Context, input RememberInput) (*RememberResult, error) {
	if input.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if input.Scope == "" {
		input.Scope = ScopeGlobal
	}
	if input.Type == "" {
		input.Type = TypeFact
	}
	if input.Delivery == "" {
		input.Delivery = DeliveryOnDemand
	}
	if input.Weight <= 0 || input.Weight > 1.0 {
		input.Weight = 1.0
	}

	vector, err := s.embed.EmbedOne(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	contradictions := s.findContradictions(ctx, vector, input)

	now := time.Now().UTC()
	id := uuid.New().String()

	mem := &Memory{
		ID:         id,
		Content:    input.Content,
		Scope:      input.Scope,
		Project:    input.Project,
		Type:       input.Type,
		Delivery:   input.Delivery,
		Tags:       input.Tags,
		Weight:     input.Weight,
		Supersedes: input.Supersedes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if input.TTL != "" {
		dur, err := time.ParseDuration(input.TTL)
		if err != nil {
			return nil, fmt.Errorf("invalid ttl %q: %w", input.TTL, err)
		}
		ttl := now.Add(dur)
		mem.TTL = &ttl
	}

	if err := s.pg.Upsert(ctx, id, vector, mem); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}

	if input.Supersedes != "" {
		_ = s.pg.UpdateWeight(ctx, input.Supersedes, 0.1)
	}

	return &RememberResult{
		Memory:         mem,
		Contradictions: contradictions,
	}, nil
}

func (s *Service) findContradictions(ctx context.Context, vector []float32, input RememberInput) []ContradictionMatch {
	where, args := pg.HierarchicalWhere(string(input.Scope), input.Project, 1)
	hits, err := s.pg.SearchWithWhere(ctx, vector, where, args, 5)
	if err != nil {
		return nil
	}

	var matches []ContradictionMatch
	for _, hit := range hits {
		if hit.Score < contradictionThreshold {
			continue
		}
		if hit.Memory.TTL != nil && hit.Memory.TTL.Before(time.Now().UTC()) {
			continue
		}
		matches = append(matches, ContradictionMatch{
			Memory:     hit.Memory,
			Similarity: hit.Score,
		})
	}
	return matches
}

func (s *Service) Recall(ctx context.Context, input RecallInput) ([]RecallResult, error) {
	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if input.Limit <= 0 {
		input.Limit = 5
	}

	vector, err := s.embed.EmbedOne(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	fetchLimit := input.Limit * 3
	if fetchLimit < 15 {
		fetchLimit = 15
	}

	where, args := pg.HierarchicalWhere(input.Scope, input.Project, 1)
	hits, err := s.pg.SearchWithWhere(ctx, vector, where, args, fetchLimit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	now := time.Now().UTC()

	supersededIDs := make(map[string]bool)
	for _, hit := range hits {
		if hit.Memory.Supersedes != "" {
			supersededIDs[hit.Memory.Supersedes] = true
		}
	}

	results := make([]RecallResult, 0, len(hits))
	for _, hit := range hits {
		if hit.Memory.TTL != nil && hit.Memory.TTL.Before(now) {
			continue
		}
		if supersededIDs[hit.Memory.ID] {
			continue
		}

		sw := scopeWeight(hit.Memory.Scope)
		decay := temporalDecay(now.Sub(hit.Memory.UpdatedAt))
		finalScore := float64(hit.Score) * sw * hit.Memory.Weight * decay

		results = append(results, RecallResult{
			Memory: hit.Memory,
			Score:  float32(finalScore),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > input.Limit {
		results = results[:input.Limit]
	}

	return results, nil
}

func scopeWeight(scope Scope) float64 {
	switch scope {
	case ScopeProject:
		return scopeWeightProject
	default:
		return scopeWeightGlobal
	}
}

func temporalDecay(age time.Duration) float64 {
	days := age.Hours() / 24
	if days < 0 {
		days = 0
	}
	return math.Exp(-decayLambda * days)
}

func (s *Service) Forget(ctx context.Context, id string) error {
	return s.pg.Delete(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, content string) (*Memory, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	existing, err := s.pg.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("retrieve existing: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("memory %s not found", id)
	}

	vector, err := s.embed.EmbedOne(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	existing.Content = content
	existing.UpdatedAt = time.Now().UTC()

	if err := s.pg.Upsert(ctx, id, vector, existing); err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	return existing, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]Memory, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	filter := pg.Filter{
		Scope:    input.Scope,
		Project:  input.Project,
		Type:     input.Type,
		Delivery: input.Delivery,
	}

	memories, err := s.pg.List(ctx, filter, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	// Filter expired client-side (simple, covers edge cases)
	now := time.Now().UTC()
	var result []Memory
	for _, m := range memories {
		if m.TTL != nil && m.TTL.Before(now) {
			continue
		}
		result = append(result, m)
	}

	return result, nil
}

func (s *Service) Stats(ctx context.Context) (*StatsResult, error) {
	return s.pg.Stats(ctx)
}

func (s *Service) CleanExpired(ctx context.Context) (int, error) {
	return s.pg.CleanExpired(ctx)
}
