package service

import (
	"context"
	"fmt"
	"strings"

	"selfsystems/internal/domain"
)

type GraphService struct {
	categories domain.CategoryRepository
	resources  domain.ResourceRepository
}

type GraphNode struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	CategoryID string `json:"category_id,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
	URL        string `json:"url,omitempty"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type GraphStats struct {
	CategoryCount int `json:"category_count"`
	ResourceCount int `json:"resource_count"`
	EdgeCount     int `json:"edge_count"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

func NewGraphService(categories domain.CategoryRepository, resources domain.ResourceRepository) *GraphService {
	return &GraphService{categories: categories, resources: resources}
}

func (s *GraphService) Build(ctx context.Context, limit int) (GraphData, error) {
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}

	categories, err := s.categories.List(ctx)
	if err != nil {
		return GraphData{}, fmt.Errorf("list categories for graph: %w", err)
	}

	resources, err := s.resources.List(ctx, limit, 0)
	if err != nil {
		return GraphData{}, fmt.Errorf("list resources for graph: %w", err)
	}

	nodes := make([]GraphNode, 0, len(categories)+len(resources))
	edges := make([]GraphEdge, 0, len(resources))

	categoryNodeIDs := map[string]string{}
	for _, category := range categories {
		nodeID := "c:" + category.ID
		categoryNodeIDs[category.ID] = nodeID
		nodes = append(nodes, GraphNode{
			ID:         nodeID,
			Type:       "category",
			Label:      category.Name,
			CategoryID: category.ID,
		})
	}

	for _, resource := range resources {
		resourceNodeID := "r:" + resource.ID
		nodes = append(nodes, GraphNode{
			ID:         resourceNodeID,
			Type:       "resource",
			Label:      graphResourceLabel(resource),
			ResourceID: resource.ID,
			CategoryID: resource.CategoryID,
			URL:        resource.URL,
		})

		if strings.TrimSpace(resource.CategoryID) == "" {
			continue
		}

		categoryNodeID, exists := categoryNodeIDs[resource.CategoryID]
		if !exists {
			categoryNodeID = "c:" + resource.CategoryID
			categoryNodeIDs[resource.CategoryID] = categoryNodeID
			nodes = append(nodes, GraphNode{
				ID:         categoryNodeID,
				Type:       "category",
				Label:      graphCategoryLabel(resource.CategoryName, resource.CategoryID),
				CategoryID: resource.CategoryID,
			})
		}

		edges = append(edges, GraphEdge{
			ID:     "e:" + resource.ID,
			Source: resourceNodeID,
			Target: categoryNodeID,
			Type:   "belongs_to",
		})
	}

	return GraphData{
		Nodes: nodes,
		Edges: edges,
		Stats: GraphStats{
			CategoryCount: countCategoryNodes(nodes),
			ResourceCount: countResourceNodes(nodes),
			EdgeCount:     len(edges),
		},
	}, nil
}

func graphResourceLabel(resource domain.Resource) string {
	if trimmed := strings.TrimSpace(resource.Title); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(resource.URL); trimmed != "" {
		return trimmed
	}
	return "Untitled Resource"
}

func graphCategoryLabel(name, id string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	return "Category " + strings.TrimSpace(id)
}

func countCategoryNodes(nodes []GraphNode) int {
	count := 0
	for _, node := range nodes {
		if node.Type == "category" {
			count++
		}
	}
	return count
}

func countResourceNodes(nodes []GraphNode) int {
	count := 0
	for _, node := range nodes {
		if node.Type == "resource" {
			count++
		}
	}
	return count
}
