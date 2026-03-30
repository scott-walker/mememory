package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/scott/claude-memory/internal/embeddings"
	qdrantclient "github.com/scott/claude-memory/internal/qdrant"
)

const (
	// Contradiction detection threshold — similarity above this triggers a warning
	contradictionThreshold float32 = 0.75

	// Temporal decay lambda — gentle decay: 0.01 means ~37% reduction after 100 days
	decayLambda = 0.005

	// Scope weights for recall scoring
	scopeWeightPersona = 1.0
	scopeWeightProject = 0.8
	scopeWeightGlobal  = 0.6
)

type Service struct {
	qdrant *qdrantclient.Client
	embed  *embeddings.Client
}

func NewService(qdrant *qdrantclient.Client, embed *embeddings.Client) *Service {
	return &Service{qdrant: qdrant, embed: embed}
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
	if input.Weight <= 0 {
		input.Weight = 1.0
	}
	if input.Weight > 1.0 {
		input.Weight = 1.0
	}

	vector, err := s.embed.EmbedOne(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	// Contradiction detection — search for similar memories before storing
	contradictions := s.findContradictions(ctx, vector, input)

	now := time.Now().UTC()
	id := uuid.New().String()

	mem := &Memory{
		ID:         id,
		Content:    input.Content,
		Scope:      input.Scope,
		Project:    input.Project,
		Persona:    input.Persona,
		Type:       input.Type,
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

	payload := map[string]interface{}{
		"content":    mem.Content,
		"scope":      string(mem.Scope),
		"type":       string(mem.Type),
		"weight":     fmt.Sprintf("%.2f", mem.Weight),
		"created_at": mem.CreatedAt.Format(time.RFC3339),
		"updated_at": mem.UpdatedAt.Format(time.RFC3339),
	}
	if mem.Project != "" {
		payload["project"] = mem.Project
	}
	if mem.Persona != "" {
		payload["persona"] = mem.Persona
	}
	if len(mem.Tags) > 0 {
		payload["tags"] = strings.Join(mem.Tags, ",")
	}
	if mem.TTL != nil {
		payload["ttl"] = mem.TTL.Format(time.RFC3339)
	}
	if mem.Supersedes != "" {
		payload["supersedes"] = mem.Supersedes
	}

	if err := s.qdrant.Upsert(ctx, id, vector, payload); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}

	// If this memory supersedes another, lower the old one's weight
	if input.Supersedes != "" {
		_ = s.lowerWeight(ctx, input.Supersedes, 0.1)
	}

	return &RememberResult{
		Memory:         mem,
		Contradictions: contradictions,
	}, nil
}

// findContradictions searches for semantically similar existing memories that might conflict.
func (s *Service) findContradictions(ctx context.Context, vector []float32, input RememberInput) []ContradictionMatch {
	filter := buildRecallFilter(string(input.Scope), input.Project, input.Persona)
	hits, err := s.qdrant.Search(ctx, vector, filter, 5)
	if err != nil {
		return nil
	}

	var matches []ContradictionMatch
	for _, hit := range hits {
		if hit.Score < contradictionThreshold {
			continue
		}
		mem := payloadToMemory(hit.ID, hit.Payload)
		// Skip expired
		if mem.TTL != nil && mem.TTL.Before(time.Now().UTC()) {
			continue
		}
		matches = append(matches, ContradictionMatch{
			Memory:     mem,
			Similarity: hit.Score,
		})
	}
	return matches
}

// lowerWeight reduces the weight of an existing memory (used when superseded).
func (s *Service) lowerWeight(ctx context.Context, id string, newWeight float64) error {
	filter := &pb.Filter{
		Must: []*pb.Condition{idCondition(id)},
	}
	existing, err := s.qdrant.Scroll(ctx, filter, 1)
	if err != nil || len(existing) == 0 {
		return err
	}

	oldPayload := existing[0].Payload
	payload := make(map[string]interface{})
	for k, v := range oldPayload {
		payload[k] = v.GetStringValue()
	}
	payload["weight"] = fmt.Sprintf("%.2f", newWeight)
	payload["updated_at"] = time.Now().UTC().Format(time.RFC3339)

	// Re-embed not needed — only payload update. Use Upsert with same vector.
	// Since we don't have the vector, we scroll with vector — but Qdrant scroll doesn't return vectors.
	// Workaround: embed the content again.
	content := oldPayload["content"].GetStringValue()
	vector, err := s.embed.EmbedOne(ctx, content)
	if err != nil {
		return err
	}

	return s.qdrant.Upsert(ctx, id, vector, payload)
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

	filter := buildRecallFilter(input.Scope, input.Project, input.Persona)

	// Fetch more than needed to compensate for filtering
	fetchLimit := uint64(input.Limit * 3)
	if fetchLimit < 15 {
		fetchLimit = 15
	}

	hits, err := s.qdrant.Search(ctx, vector, filter, fetchLimit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	now := time.Now().UTC()

	// Collect superseded IDs for filtering
	supersededIDs := make(map[string]bool)
	for _, hit := range hits {
		if v, ok := hit.Payload["supersedes"]; ok {
			if sid := v.GetStringValue(); sid != "" {
				supersededIDs[sid] = true
			}
		}
	}

	results := make([]RecallResult, 0, len(hits))
	for _, hit := range hits {
		mem := payloadToMemory(hit.ID, hit.Payload)

		// Skip expired
		if mem.TTL != nil && mem.TTL.Before(now) {
			continue
		}
		// Skip superseded memories
		if supersededIDs[mem.ID] {
			continue
		}

		// Compute final score: similarity * scope_weight * weight * decay(age)
		sw := scopeWeight(mem.Scope)
		decay := temporalDecay(now.Sub(mem.UpdatedAt))
		finalScore := float64(hit.Score) * sw * mem.Weight * decay

		results = append(results, RecallResult{
			Memory: mem,
			Score:  float32(finalScore),
		})
	}

	// Re-sort by computed score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Trim to requested limit
	if len(results) > input.Limit {
		results = results[:input.Limit]
	}

	return results, nil
}

func scopeWeight(scope Scope) float64 {
	switch scope {
	case ScopePersona:
		return scopeWeightPersona
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
	return s.qdrant.Delete(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, content string) (*Memory, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Retrieve existing point to preserve metadata
	filter := &pb.Filter{
		Must: []*pb.Condition{
			idCondition(id),
		},
	}
	existing, err := s.qdrant.Scroll(ctx, filter, 1)
	if err != nil {
		return nil, fmt.Errorf("retrieve existing: %w", err)
	}
	if len(existing) == 0 {
		return nil, fmt.Errorf("memory %s not found", id)
	}

	vector, err := s.embed.EmbedOne(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	now := time.Now().UTC()
	oldPayload := existing[0].Payload

	payload := make(map[string]interface{})
	payload["content"] = content
	payload["updated_at"] = now.Format(time.RFC3339)

	for _, key := range []string{"scope", "project", "persona", "type", "tags", "weight", "supersedes", "created_at", "ttl"} {
		if v, ok := oldPayload[key]; ok {
			payload[key] = v.GetStringValue()
		}
	}

	if err := s.qdrant.Upsert(ctx, id, vector, payload); err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	mem := payloadToMemory(id, oldPayload)
	mem.Content = content
	mem.UpdatedAt = now
	return &mem, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]Memory, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	var conditions []*pb.Condition
	if input.Scope != "" {
		conditions = append(conditions, fieldMatch("scope", input.Scope))
	}
	if input.Project != "" {
		conditions = append(conditions, fieldMatch("project", input.Project))
	}
	if input.Persona != "" {
		conditions = append(conditions, fieldMatch("persona", input.Persona))
	}
	if input.Type != "" {
		conditions = append(conditions, fieldMatch("type", input.Type))
	}

	var filter *pb.Filter
	if len(conditions) > 0 {
		filter = &pb.Filter{Must: conditions}
	}

	hits, err := s.qdrant.Scroll(ctx, filter, uint32(input.Limit))
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	memories := make([]Memory, 0, len(hits))
	for _, hit := range hits {
		mem := payloadToMemory(hit.ID, hit.Payload)
		if mem.TTL != nil && mem.TTL.Before(time.Now().UTC()) {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, nil
}

func (s *Service) Stats(ctx context.Context) (*StatsResult, error) {
	total, err := s.qdrant.Count(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	result := &StatsResult{
		Total:     total,
		ByScope:   make(map[string]uint64),
		ByProject: make(map[string]uint64),
		ByPersona: make(map[string]uint64),
		ByType:    make(map[string]uint64),
	}

	for _, scope := range []string{"global", "project", "persona"} {
		count, err := s.qdrant.Count(ctx, &pb.Filter{Must: []*pb.Condition{fieldMatch("scope", scope)}})
		if err != nil {
			continue
		}
		if count > 0 {
			result.ByScope[scope] = count
		}
	}

	for _, typ := range []string{"fact", "rule", "decision", "feedback", "context"} {
		count, err := s.qdrant.Count(ctx, &pb.Filter{Must: []*pb.Condition{fieldMatch("type", typ)}})
		if err != nil {
			continue
		}
		if count > 0 {
			result.ByType[typ] = count
		}
	}

	return result, nil
}

// CleanExpired removes memories with TTL in the past by scrolling all points
// and deleting those with expired TTL client-side.
func (s *Service) CleanExpired(ctx context.Context) (int, error) {
	points, err := s.qdrant.Scroll(ctx, nil, 1000)
	if err != nil {
		return 0, fmt.Errorf("scroll: %w", err)
	}

	now := time.Now().UTC()
	var deleted int
	for _, p := range points {
		if v, ok := p.Payload["ttl"]; ok {
			if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil && t.Before(now) {
				if err := s.qdrant.Delete(ctx, p.ID); err == nil {
					deleted++
				}
			}
		}
	}

	return deleted, nil
}

// --- Filter builders ---

func buildRecallFilter(scope, project, persona string) *pb.Filter {
	var shouldClauses []*pb.Condition

	// Always include global scope
	shouldClauses = append(shouldClauses, fieldMatch("scope", "global"))

	// Include project scope if project specified
	if project != "" {
		shouldClauses = append(shouldClauses, &pb.Condition{
			ConditionOneOf: &pb.Condition_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						fieldMatch("scope", "project"),
						fieldMatch("project", project),
					},
				},
			},
		})
	}

	// Include persona scope if persona specified
	if persona != "" {
		personaConditions := []*pb.Condition{
			fieldMatch("scope", "persona"),
			fieldMatch("persona", persona),
		}
		if project != "" {
			personaConditions = append(personaConditions, fieldMatch("project", project))
		}
		shouldClauses = append(shouldClauses, &pb.Condition{
			ConditionOneOf: &pb.Condition_Filter{
				Filter: &pb.Filter{
					Must: personaConditions,
				},
			},
		})
	}

	// If explicit scope filter requested (no hierarchy)
	if scope != "" && project == "" && persona == "" {
		return &pb.Filter{
			Must: []*pb.Condition{fieldMatch("scope", scope)},
		}
	}

	return &pb.Filter{
		Should: shouldClauses,
	}
}

func fieldMatch(key, value string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: key,
				Match: &pb.Match{
					MatchValue: &pb.Match_Keyword{
						Keyword: value,
					},
				},
			},
		},
	}
}

func idCondition(id string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_HasId{
			HasId: &pb.HasIdCondition{
				HasId: []*pb.PointId{pb.NewIDUUID(id)},
			},
		},
	}
}

func payloadToMemory(id string, payload map[string]*pb.Value) Memory {
	mem := Memory{ID: id, Weight: 1.0} // Default weight

	if v, ok := payload["content"]; ok {
		mem.Content = v.GetStringValue()
	}
	if v, ok := payload["scope"]; ok {
		mem.Scope = Scope(v.GetStringValue())
	}
	if v, ok := payload["project"]; ok {
		mem.Project = v.GetStringValue()
	}
	if v, ok := payload["persona"]; ok {
		mem.Persona = v.GetStringValue()
	}
	if v, ok := payload["type"]; ok {
		mem.Type = MemoryType(v.GetStringValue())
	}
	if v, ok := payload["tags"]; ok {
		if tags := v.GetStringValue(); tags != "" {
			mem.Tags = strings.Split(tags, ",")
		}
	}
	if v, ok := payload["weight"]; ok {
		if w := v.GetStringValue(); w != "" {
			var parsed float64
			if _, err := fmt.Sscanf(w, "%f", &parsed); err == nil && parsed > 0 {
				mem.Weight = parsed
			}
		}
	}
	if v, ok := payload["supersedes"]; ok {
		mem.Supersedes = v.GetStringValue()
	}
	if v, ok := payload["created_at"]; ok {
		if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil {
			mem.CreatedAt = t
		}
	}
	if v, ok := payload["updated_at"]; ok {
		if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil {
			mem.UpdatedAt = t
		}
	}
	if v, ok := payload["ttl"]; ok {
		if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil {
			mem.TTL = &t
		}
	}

	return mem
}
