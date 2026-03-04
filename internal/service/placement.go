package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/dcm-project/placement-manager/internal/api/server"
	"github.com/dcm-project/placement-manager/internal/store"
	"github.com/google/uuid"
)

// PlacementService handles business logic for placement request management.
type PlacementService struct {
	store store.Store
}

// NewPlacementService creates a new PlacementService with the given store.
func NewPlacementService(store store.Store) *PlacementService {
	return &PlacementService{store: store}
}

// CreateResource creates a new placement request.
func (s *PlacementService) CreateResource(ctx context.Context, req *server.Resource, queryId *string) (*server.Resource, error) {
	// Get or Generate ID
	resourceIDStr := getOrGenerateStringId(queryId)

	// Generate path
	path := fmt.Sprintf("resources/%s", resourceIDStr)

	// Validate request with policy engine

	// Convert to store model
	requestModel := resourceToStoreModel(req, resourceIDStr, path)

	// Create in store
	created, err := s.store.Resource().Create(ctx, requestModel)
	if err != nil {
		return nil, NewInternalError(fmt.Sprintf("failed to create database record for resource %s: %v", resourceIDStr, err))
	}

	// Send request to SP Resource Manager

	log.Printf("Successfully created resource: %s (catalog_item_instance_id: %s)", created.ID, created.CatalogItemInstanceId)
	return storeModelToResource(created), nil
}

// GetResource retrieves a placement request by ID.
func (s *PlacementService) GetResource(ctx context.Context, requestID string) (*server.Resource, error) {
	id, err := uuid.Parse(requestID)
	if err != nil {
		return nil, NewValidationError("invalid resource ID format")
	}

	request, err := s.store.Resource().Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrRequestNotFound) {
			return nil, NewNotFoundError(fmt.Sprintf("resource %s not found", requestID))
		}
		return nil, NewInternalError(fmt.Sprintf("failed to retrieve resource: %v", err))
	}

	return storeModelToResource(request), nil
}

// ListResources returns placement requests with optional filtering and pagination.
func (s *PlacementService) ListResources(ctx context.Context, providerName *string, maxPageSize *int, pageToken *string) (*server.ResourceList, error) {
	opts := &store.ResourceListOptions{
		ProviderName: providerName,
	}

	// Apply max page size
	if maxPageSize != nil {
		if *maxPageSize > 0 && *maxPageSize <= 100 {
			opts.PageSize = *maxPageSize
		} else {
			return nil, NewValidationError("page size must be between 1 and 100")
		}
	}

	// Apply page token
	if pageToken != nil && *pageToken != "" {
		opts.PageToken = pageToken
	}

	// Get resources from store
	result, err := s.store.Resource().List(ctx, opts)
	if err != nil {
		return nil, NewInternalError(fmt.Sprintf("failed to list resources: %v", err))
	}

	// Convert to API types
	resources := make([]server.Resource, len(result.Resources))
	for i, resource := range result.Resources {
		resources[i] = *storeModelToResource(&resource)
	}

	return &server.ResourceList{
		Resources:     resources,
		NextPageToken: result.NextPageToken,
	}, nil
}

// DeleteResource removes a placement request by ID.
func (s *PlacementService) DeleteResource(ctx context.Context, requestID string) error {
	id, err := uuid.Parse(requestID)
	if err != nil {
		return NewValidationError("invalid resource ID format")
	}

	err = s.store.Resource().Delete(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrRequestNotFound) {
			return NewNotFoundError(fmt.Sprintf("resource %s not found", requestID))
		}
		return NewInternalError(fmt.Sprintf("failed to delete database record for resource %s: %v", requestID, err))
	}

	log.Printf("Deleted resource from DB record: %s", requestID)
	return nil
}

func getOrGenerateStringId(id *string) string {
	if id != nil && *id != "" {
		return *id
	}
	// Generate UUID if not provided
	return uuid.New().String()
}
