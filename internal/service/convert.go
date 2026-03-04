package service

import (
	"fmt"
	"time"

	"github.com/dcm-project/placement-manager/internal/api/server"
	"github.com/dcm-project/placement-manager/internal/store/model"
	"github.com/google/uuid"
)

// storeModelToResource converts a database model to an API response type
func storeModelToResource(m *model.Resource) *server.Resource {
	idStr := m.ID.String()
	path := fmt.Sprintf("resources/%s", idStr)

	resource := &server.Resource{
		Id:                    &idStr,
		Path:                  &path,
		CatalogItemInstanceId: m.CatalogItemInstanceId,
		Spec:                  m.Spec,
		ProviderName:          m.ProviderName,
		ApprovalStatus:        m.ApprovalStatus,
		CreateTime:            PtrTime(m.CreateTime),
		UpdateTime:            PtrTime(m.UpdateTime),
	}
	return resource
}

// resourceToStoreModel converts an API request to a database model
func resourceToStoreModel(req *server.Resource, id, path string) model.Resource {
	return model.Resource{
		ID:                    uuid.MustParse(id),
		CatalogItemInstanceId: req.CatalogItemInstanceId,
		Spec:                  req.Spec,
		Path:                  path,
	}
}

// Helper functions for pointer conversions

func PtrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
