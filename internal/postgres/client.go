package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	pgvector "github.com/pgvector/pgvector-go"
	t "github.com/scott-walker/mememory/internal/types"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Client struct {
	db *sql.DB
}

func NewClient(databaseURL string) (*Client, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return &Client{db: db}, nil
}

func (c *Client) RunMigrations(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var exists bool
		err := c.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)", entry.Name()).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if _, err := c.db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := c.db.ExecContext(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", entry.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (c *Client) Upsert(ctx context.Context, id string, embedding []float32, mem *t.Memory) error {
	vec := pgvector.NewVector(embedding)
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO memories (id, content, embedding, scope, project, persona, type, tags, weight, supersedes, created_at, updated_at, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			scope = EXCLUDED.scope,
			project = EXCLUDED.project,
			persona = EXCLUDED.persona,
			type = EXCLUDED.type,
			tags = EXCLUDED.tags,
			weight = EXCLUDED.weight,
			supersedes = EXCLUDED.supersedes,
			updated_at = EXCLUDED.updated_at,
			ttl = EXCLUDED.ttl`,
		id, mem.Content, vec, string(mem.Scope), nilIfEmpty(mem.Project), nilIfEmpty(mem.Persona),
		string(mem.Type), pq.Array(mem.Tags), mem.Weight, nilIfEmpty(mem.Supersedes),
		mem.CreatedAt, mem.UpdatedAt, mem.TTL,
	)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	return nil
}

func (c *Client) Search(ctx context.Context, embedding []float32, filter Filter, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(embedding)
	where, args := filter.toWhere(1) // $1 is the vector
	args = append([]interface{}{vec}, args...)

	query := fmt.Sprintf(`
		SELECT id, content, scope, project, persona, type, tags, weight, supersedes, created_at, updated_at, ttl,
			1 - (embedding <=> $1) AS score
		FROM memories
		%s
		ORDER BY embedding <=> $1
		LIMIT %d`, where, limit)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanResults(rows)
}

// SearchWithWhere performs vector search with a custom WHERE clause (for hierarchical recall)
func (c *Client) SearchWithWhere(ctx context.Context, embedding []float32, where string, whereArgs []interface{}, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(embedding)
	args := append([]interface{}{vec}, whereArgs...)

	query := fmt.Sprintf(`
		SELECT id, content, scope, project, persona, type, tags, weight, supersedes, created_at, updated_at, ttl,
			1 - (embedding <=> $1) AS score
		FROM memories
		%s
		ORDER BY embedding <=> $1
		LIMIT %d`, where, limit)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanResults(rows)
}

func (c *Client) GetByID(ctx context.Context, id string) (*t.Memory, error) {
	row := c.db.QueryRowContext(ctx, `
		SELECT id, content, scope, project, persona, type, tags, weight, supersedes, created_at, updated_at, ttl
		FROM memories WHERE id = $1`, id)
	mem, err := scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}
	return mem, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	_, err := c.db.ExecContext(ctx, "DELETE FROM memories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (c *Client) List(ctx context.Context, filter Filter, limit int) ([]t.Memory, error) {
	where, args := filter.toWhere(0)
	query := fmt.Sprintf(`
		SELECT id, content, scope, project, persona, type, tags, weight, supersedes, created_at, updated_at, ttl
		FROM memories %s
		ORDER BY updated_at DESC
		LIMIT %d`, where, limit)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var memories []t.Memory
	for rows.Next() {
		mem, err := scanMemoryFromRows(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, *mem)
	}
	return memories, rows.Err()
}

func (c *Client) UpdateWeight(ctx context.Context, id string, weight float64) error {
	_, err := c.db.ExecContext(ctx,
		"UPDATE memories SET weight = $1, updated_at = $2 WHERE id = $3",
		weight, time.Now().UTC(), id)
	return err
}

func (c *Client) Stats(ctx context.Context) (*t.StatsResult, error) {
	result := &t.StatsResult{
		ByScope:   make(map[string]uint64),
		ByProject: make(map[string]uint64),
		ByPersona: make(map[string]uint64),
		ByType:    make(map[string]uint64),
	}

	// Total
	if err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&result.Total); err != nil {
		return nil, err
	}

	// By scope
	rows, err := c.db.QueryContext(ctx, "SELECT scope, COUNT(*) FROM memories GROUP BY scope")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v uint64
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return nil, err
		}
		result.ByScope[k] = v
	}
	rows.Close()

	// By project
	rows, err = c.db.QueryContext(ctx, "SELECT project, COUNT(*) FROM memories WHERE project IS NOT NULL GROUP BY project")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v uint64
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return nil, err
		}
		result.ByProject[k] = v
	}
	rows.Close()

	// By type
	rows, err = c.db.QueryContext(ctx, "SELECT type, COUNT(*) FROM memories GROUP BY type")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v uint64
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return nil, err
		}
		result.ByType[k] = v
	}
	rows.Close()

	return result, nil
}

func (c *Client) CleanExpired(ctx context.Context) (int, error) {
	res, err := c.db.ExecContext(ctx, "DELETE FROM memories WHERE ttl IS NOT NULL AND ttl < NOW()")
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

// --- Filter ---

type Filter struct {
	Scope   string
	Project string
	Persona string
	Type    string
}

func (f Filter) toWhere(argOffset int) (string, []interface{}) {
	var conds []string
	var args []interface{}
	idx := argOffset + 1

	if f.Scope != "" {
		conds = append(conds, fmt.Sprintf("scope = $%d", idx))
		args = append(args, f.Scope)
		idx++
	}
	if f.Project != "" {
		conds = append(conds, fmt.Sprintf("project = $%d", idx))
		args = append(args, f.Project)
		idx++
	}
	if f.Persona != "" {
		conds = append(conds, fmt.Sprintf("persona = $%d", idx))
		args = append(args, f.Persona)
		idx++
	}
	if f.Type != "" {
		conds = append(conds, fmt.Sprintf("type = $%d", idx))
		args = append(args, f.Type)
		idx++
	}

	if len(conds) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

// HierarchicalWhere builds the OR-based hierarchical filter for recall
func HierarchicalWhere(scope, project, persona string, argOffset int) (string, []interface{}) {
	// If explicit scope with no project/persona — simple filter
	if scope != "" && project == "" && persona == "" {
		return fmt.Sprintf("WHERE scope = $%d", argOffset+1), []interface{}{scope}
	}

	var clauses []string
	var args []interface{}
	idx := argOffset + 1

	// Always include global
	clauses = append(clauses, "scope = 'global'")

	if project != "" {
		clauses = append(clauses, fmt.Sprintf("(scope = 'project' AND project = $%d)", idx))
		args = append(args, project)
		idx++
	}

	if persona != "" {
		if project != "" {
			clauses = append(clauses, fmt.Sprintf("(scope = 'persona' AND persona = $%d AND project = $%d)", idx, idx-1))
		} else {
			clauses = append(clauses, fmt.Sprintf("(scope = 'persona' AND persona = $%d)", idx))
		}
		args = append(args, persona)
	}

	return "WHERE (" + strings.Join(clauses, " OR ") + ")", args
}

// --- Scan helpers ---

func scanMemory(row *sql.Row) (*t.Memory, error) {
	var m t.Memory
	var project, persona, supersedes sql.NullString
	var ttl sql.NullTime
	var scope, typ string

	err := row.Scan(&m.ID, &m.Content, &scope, &project, &persona, &typ,
		pq.Array(&m.Tags), &m.Weight, &supersedes, &m.CreatedAt, &m.UpdatedAt, &ttl)
	if err != nil {
		return nil, err
	}

	m.Scope = t.Scope(scope)
	m.Type = t.MemoryType(typ)
	if project.Valid {
		m.Project = project.String
	}
	if persona.Valid {
		m.Persona = persona.String
	}
	if supersedes.Valid {
		m.Supersedes = supersedes.String
	}
	if ttl.Valid {
		m.TTL = &ttl.Time
	}
	return &m, nil
}

func scanMemoryFromRows(rows *sql.Rows) (*t.Memory, error) {
	var m t.Memory
	var project, persona, supersedes sql.NullString
	var ttl sql.NullTime
	var scope, typ string

	err := rows.Scan(&m.ID, &m.Content, &scope, &project, &persona, &typ,
		pq.Array(&m.Tags), &m.Weight, &supersedes, &m.CreatedAt, &m.UpdatedAt, &ttl)
	if err != nil {
		return nil, err
	}

	m.Scope = t.Scope(scope)
	m.Type = t.MemoryType(typ)
	if project.Valid {
		m.Project = project.String
	}
	if persona.Valid {
		m.Persona = persona.String
	}
	if supersedes.Valid {
		m.Supersedes = supersedes.String
	}
	if ttl.Valid {
		m.TTL = &ttl.Time
	}
	return &m, nil
}

func scanResults(rows *sql.Rows) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var m t.Memory
		var project, persona, supersedes sql.NullString
		var ttl sql.NullTime
		var scope, typ string
		var score float32

		err := rows.Scan(&m.ID, &m.Content, &scope, &project, &persona, &typ,
			pq.Array(&m.Tags), &m.Weight, &supersedes, &m.CreatedAt, &m.UpdatedAt, &ttl, &score)
		if err != nil {
			return nil, err
		}

		m.Scope = t.Scope(scope)
		m.Type = t.MemoryType(typ)
		if project.Valid {
			m.Project = project.String
		}
		if persona.Valid {
			m.Persona = persona.String
		}
		if supersedes.Valid {
			m.Supersedes = supersedes.String
		}
		if ttl.Valid {
			m.TTL = &ttl.Time
		}

		results = append(results, SearchResult{Memory: m, Score: score})
	}
	return results, rows.Err()
}

type SearchResult struct {
	Memory t.Memory
	Score  float32
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
