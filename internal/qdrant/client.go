package qdrant

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
)

const CollectionName = "memory"

type Client struct {
	conn *pb.Client
}

func NewClient(host string, port int) (*Client, error) {
	conn, err := pb.NewClient(&pb.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant connect: %w", err)
	}
	return &Client{conn: conn}, nil
}

func (c *Client) EnsureCollection(ctx context.Context) error {
	collections, err := c.conn.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("qdrant list collections: %w", err)
	}
	for _, col := range collections {
		if col == CollectionName {
			return nil
		}
	}

	vectorSize := uint64(768)
	err = c.conn.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: CollectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant create collection: %w", err)
	}

	indexes := []struct {
		field  string
		schema pb.FieldType
	}{
		{"scope", pb.FieldType_FieldTypeKeyword},
		{"project", pb.FieldType_FieldTypeKeyword},
		{"persona", pb.FieldType_FieldTypeKeyword},
		{"type", pb.FieldType_FieldTypeKeyword},
		{"ttl", pb.FieldType_FieldTypeDatetime},
	}
	for _, idx := range indexes {
		_, _ = c.conn.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
			CollectionName: CollectionName,
			FieldName:      idx.field,
			FieldType:      &idx.schema,
		})
	}

	return nil
}

func (c *Client) Upsert(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error {
	wait := true
	_, err := c.conn.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Wait:           &wait,
		Points: []*pb.PointStruct{
			{
				Id:      pb.NewIDUUID(id),
				Vectors: pb.NewVectors(vector...),
				Payload: pb.NewValueMap(payload),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant upsert: %w", err)
	}
	return nil
}

type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]*pb.Value
}

func (c *Client) Search(ctx context.Context, vector []float32, filter *pb.Filter, limit uint64) ([]SearchResult, error) {
	resp, err := c.conn.Query(ctx, &pb.QueryPoints{
		CollectionName: CollectionName,
		Query:          pb.NewQuery(vector...),
		Filter:         filter,
		WithPayload:    pb.NewWithPayload(true),
		Limit:          &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	results := make([]SearchResult, 0, len(resp))
	for _, point := range resp {
		results = append(results, SearchResult{
			ID:      point.GetId().GetUuid(),
			Score:   point.GetScore(),
			Payload: point.GetPayload(),
		})
	}
	return results, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	wait := true
	_, err := c.conn.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Wait:           &wait,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{pb.NewIDUUID(id)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant delete: %w", err)
	}
	return nil
}

func (c *Client) Scroll(ctx context.Context, filter *pb.Filter, limit uint32) ([]SearchResult, error) {
	resp, err := c.conn.Scroll(ctx, &pb.ScrollPoints{
		CollectionName: CollectionName,
		Filter:         filter,
		WithPayload:    pb.NewWithPayload(true),
		Limit:          &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant scroll: %w", err)
	}

	results := make([]SearchResult, 0, len(resp))
	for _, point := range resp {
		results = append(results, SearchResult{
			ID:      point.GetId().GetUuid(),
			Payload: point.GetPayload(),
		})
	}
	return results, nil
}

func (c *Client) DeleteByFilter(ctx context.Context, filter *pb.Filter) error {
	wait := true
	_, err := c.conn.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Wait:           &wait,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: filter,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant delete by filter: %w", err)
	}
	return nil
}

func (c *Client) Count(ctx context.Context, filter *pb.Filter) (uint64, error) {
	exact := true
	count, err := c.conn.Count(ctx, &pb.CountPoints{
		CollectionName: CollectionName,
		Filter:         filter,
		Exact:          &exact,
	})
	if err != nil {
		return 0, fmt.Errorf("qdrant count: %w", err)
	}
	return count, nil
}
