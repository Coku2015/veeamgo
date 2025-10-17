package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ScaleOutRepositoriesFilter defines filters used when listing scale-out repositories.
type ScaleOutRepositoriesFilter struct {
	Name           string
	OrderColumn    string
	OrderAscending *bool
	Skip           int
	MaxItems       int
}

// ScaleOutRepository models the scale-out repository payload returned by the API.
type ScaleOutRepository struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	UniqueID        string                   `json:"uniqueId"`
	PerformanceTier ScaleOutPerformanceTier  `json:"performanceTier"`
	PlacementPolicy *ScaleOutPlacementPolicy `json:"placementPolicy,omitempty"`
	CapacityTier    *ScaleOutCapacityTier    `json:"capacityTier,omitempty"`
	ArchiveTier     *ScaleOutArchiveTier     `json:"archiveTier,omitempty"`
}

// ScaleOutPerformanceTier contains performance tier details.
type ScaleOutPerformanceTier struct {
	PerformanceExtents []ScaleOutPerformanceExtent      `json:"performanceExtents"`
	AdvancedSettings   *ScaleOutPerformanceTierSettings `json:"advancedSettings,omitempty"`
}

// ScaleOutPerformanceExtent describes a single performance extent.
type ScaleOutPerformanceExtent struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Status []string `json:"status,omitempty"`
}

// ScaleOutPerformanceTierSettings exposes selected advanced settings.
type ScaleOutPerformanceTierSettings struct {
	PerVMBackup           bool `json:"perVmBackup"`
	FullWhenExtentOffline bool `json:"fullWhenExtentOffline"`
}

// ScaleOutPlacementPolicy outlines the placement strategy configuration.
type ScaleOutPlacementPolicy struct {
	Type                         string                     `json:"type"`
	Settings                     []ScaleOutPlacementSetting `json:"settings,omitempty"`
	EnforceStrictPlacementPolicy bool                       `json:"enforceStrictPlacementPolicy"`
}

// ScaleOutPlacementSetting captures per-extent placement preferences.
type ScaleOutPlacementSetting struct {
	Name           string `json:"name"`
	ExtentID       string `json:"extentId"`
	AllowedBackups string `json:"allowedBackups"`
}

// ScaleOutCapacityTier holds optional capacity tier configuration.
type ScaleOutCapacityTier struct {
	IsEnabled         bool                     `json:"isEnabled"`
	Extents           []ScaleOutCapacityExtent `json:"extents,omitempty"`
	CopyPolicyEnabled bool                     `json:"copyPolicyEnabled"`
	MovePolicyEnabled bool                     `json:"movePolicyEnabled"`
}

// ScaleOutCapacityExtent references object storage capacity extents.
type ScaleOutCapacityExtent struct {
	ID string `json:"id"`
}

// ScaleOutArchiveTier holds archive tier configuration.
type ScaleOutArchiveTier struct {
	IsEnabled         bool   `json:"isEnabled"`
	ExtentID          string `json:"extentId"`
	ArchivePeriodDays int    `json:"archivePeriodDays"`
}

type scaleOutRepositoriesResponse struct {
	Data       []ScaleOutRepository `json:"data"`
	Pagination paginationResult     `json:"pagination"`
}

// ScaleOutRepositories retrieves scale-out repositories using the supplied filter.
func (c *Client) ScaleOutRepositories(ctx context.Context, filter ScaleOutRepositoriesFilter) ([]ScaleOutRepository, error) {
	results := make([]ScaleOutRepository, 0)
	skip := filter.Skip
	remaining := filter.MaxItems

	for {
		limit := defaultPageSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}

		query := url.Values{}
		query.Set("skip", strconv.Itoa(skip))
		query.Set("limit", strconv.Itoa(limit))
		if filter.Name != "" {
			query.Set("nameFilter", filter.Name)
		}
		if filter.OrderColumn != "" {
			query.Set("orderColumn", filter.OrderColumn)
		}
		if filter.OrderAscending != nil {
			query.Set("orderAsc", strconv.FormatBool(*filter.OrderAscending))
		}

		var resp scaleOutRepositoriesResponse
		if err := c.getJSONWithQuery(ctx, "/api/v1/backupInfrastructure/scaleOutRepositories", query, &resp); err != nil {
			return nil, err
		}

		results = append(results, resp.Data...)

		if filter.MaxItems > 0 && len(results) >= filter.MaxItems {
			return results[:filter.MaxItems], nil
		}
		if resp.Pagination.Skip+resp.Pagination.Count >= resp.Pagination.Total || len(resp.Data) == 0 {
			break
		}
		skip = resp.Pagination.Skip + resp.Pagination.Count
		if filter.MaxItems > 0 {
			remaining = filter.MaxItems - len(results)
			if remaining <= 0 {
				break
			}
		}
	}

	return results, nil
}

// ScaleOutRepository fetches a single scale-out repository by its ID.
func (c *Client) ScaleOutRepository(ctx context.Context, id string) (*ScaleOutRepository, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("repository id cannot be empty")
	}
	path := fmt.Sprintf("/api/v1/backupInfrastructure/scaleOutRepositories/%s", id)
	var repo ScaleOutRepository
	if err := c.getJSON(ctx, path, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// ScaleOutRepositoryByName resolves a scale-out repository by its name.
func (c *Client) ScaleOutRepositoryByName(ctx context.Context, name string) (*ScaleOutRepository, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	repos, err := c.ScaleOutRepositories(ctx, ScaleOutRepositoriesFilter{Name: clean})
	if err != nil {
		return nil, err
	}

	exactMatches := make([]ScaleOutRepository, 0)
	for _, repo := range repos {
		if strings.EqualFold(repo.Name, clean) {
			exactMatches = append(exactMatches, repo)
		}
	}

	switch len(exactMatches) {
	case 1:
		return &exactMatches[0], nil
	case 0:
		if len(repos) == 0 {
			return nil, fmt.Errorf("scale-out repository named %q not found", clean)
		}

		suggestions := make([]string, 0, len(repos))
		for _, repo := range repos {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", repo.Name, repo.ID))
			if len(suggestions) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("scale-out repository named %q not found; did you mean: %s", clean, strings.Join(suggestions, ", "))
	default:
		dups := make([]string, 0, len(exactMatches))
		for _, repo := range exactMatches {
			dups = append(dups, repo.ID)
			if len(dups) == 5 {
				break
			}
		}
		return nil, fmt.Errorf("multiple scale-out repositories named %q found (ids: %s)", clean, strings.Join(dups, ", "))
	}
}
