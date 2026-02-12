package handlers_test

import (
	"context"

	"github.com/dcm-project/placement-manager/internal/api/server"
	"github.com/dcm-project/placement-manager/internal/handlers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handler", func() {
	var (
		handler *handlers.Handler
		ctx     context.Context
	)

	BeforeEach(func() {
		handler = handlers.NewHandler()
		ctx = context.Background()
	})

	Describe("GetHealth", func() {
		It("returns healthy status", func() {
			resp, err := handler.GetHealth(ctx, server.GetHealthRequestObject{})

			Expect(err).NotTo(HaveOccurred())
			jsonResp, ok := resp.(server.GetHealth200JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(jsonResp.Status).To(Equal("healthy"))
			Expect(*jsonResp.Path).To(Equal("/api/v1alpha1/health"))
		})
	})

	Describe("ListResources", func() {
		It("returns empty list (stub)", func() {
			req := server.ListResourcesRequestObject{}

			resp, err := handler.ListResources(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			jsonResp, ok := resp.(server.ListResources200JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(jsonResp.Resources).To(BeEmpty())
		})
	})

	Describe("CreateResource", func() {
		It("returns 201 with request body (stub)", func() {
			catalogItemInstanceID := "test-catalog-item"
			spec := map[string]interface{}{"test": "data"}

			req := server.CreateResourceRequestObject{
				Body: &server.Resource{
					CatalogItemInstanceId: catalogItemInstanceID,
					Spec:                  spec,
				},
			}

			resp, err := handler.CreateResource(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			jsonResp, ok := resp.(server.CreateResource201JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(jsonResp.CatalogItemInstanceId).To(Equal(catalogItemInstanceID))
			Expect(jsonResp.Spec).To(Equal(spec))
		})
	})

	Describe("GetResource", func() {
		It("returns 404 (stub)", func() {
			req := server.GetResourceRequestObject{
				ResourceId: "non-existent-id",
			}

			resp, err := handler.GetResource(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			_, ok := resp.(server.GetResource404ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
		})
	})

	Describe("DeleteResource", func() {
		It("returns 204 (stub)", func() {
			req := server.DeleteResourceRequestObject{
				ResourceId: "some-id",
			}

			resp, err := handler.DeleteResource(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			_, ok := resp.(server.DeleteResource204Response)
			Expect(ok).To(BeTrue())
		})
	})
})
