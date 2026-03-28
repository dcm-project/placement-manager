# Rehydration Flow Endpoint

**Commit:** 3aac5ca
**Date:** 2026-03-27
**Branch:** rehydration-flow
**Context:** [Rehydration Flow Enhancement](https://github.com/dcm-project/enhancements/blob/1f357c1213ccfbb8638f9b5baed82ada86114c15/enhancements/rehydration-flow/rehydration-flow.md), Placement Manager side

## Summary

Implemented `POST /resources/{resourceId}:rehydrate` endpoint that re-evaluates an existing resource against current policies and creates a new resource with a new instance ID. The old resource is deleted using deferred deletion for graceful degradation.

## Design Decisions

### Create-before-delete with graceful degradation

The rehydration flow creates the new resource before deleting the old one. If SPRM provisioning of the new resource fails, the operation is rolled back (new DB record deleted) and an error is returned. If the old resource's SPRM deletion fails, the flow still succeeds -- the SPRM's deferred deletion mechanism (`DELETE ?deferred=true`) queues the old instance for background cleanup.

### Error handling tiers

- **Hard failures** (steps 1-4): Old resource retrieval, policy re-evaluation, new DB creation, SPRM provisioning. Any failure here returns an error with no state change.
- **Soft failures** (steps 5-6): SPRM deferred delete and old DB record deletion. Failures are logged but do not fail the request. The 202 response is returned with the new resource.

### Original spec preservation

The DB stores the original spec (not the policy-evaluated spec). During rehydration, the original spec from the old resource is re-evaluated through the Policy Manager. The evaluated spec is sent to SPRM; the original spec is stored in the new DB record.

### CatalogItemInstanceId continuity

The `CatalogItemInstanceId` is preserved across rehydration. Both old and new resources share the same catalog ID since it represents the Catalog Manager's identifier, while the resource ID is the Placement Manager's internal identifier.

### SPRM addressed by Placement Manager resource ID

The SPRM is addressed using the Placement Manager's resource ID (not the `CatalogItemInstanceId`) for all operations: create, delete, and deferred delete. The `CatalogItemInstanceId` is only stored in the Placement Manager's DB and not passed to the SPRM.

## Files Changed

| File | Change |
|------|--------|
| `api/v1alpha1/openapi.yaml` | Added `:rehydrate` endpoint and `RehydrateRequest` schema |
| `api/v1alpha1/types.gen.go` | Generated `RehydrateRequest` type |
| `api/v1alpha1/spec.gen.go` | Regenerated embedded spec |
| `internal/api/server/server.gen.go` | Generated server interface, request/response types |
| `pkg/client/client.gen.go` | Generated client methods for rehydrate |
| `internal/sprm/client.go` | Added `DeleteResourceDeferred` method; refactored `DeleteResource` to share `deleteResource` helper; SPRM operations use Placement Manager resource ID (not CatalogItemInstanceId) |
| `internal/service/placement.go` | Added `RehydrateResource` method with full flow |
| `internal/handlers/handler.go` | Added `RehydrateResource` handler |
| `internal/handlers/errors.go` | Added `handleRehydrateResourceError` mapping |
| `internal/handlers/handler_test.go` | Updated mock SPRM client with `DeleteResourceDeferred` |
| `internal/service/placement_test.go` | Added 10 unit tests for rehydration scenarios |
| `test/subsystem/setup_test.go` | Added WireMock stubs for deferred delete |
| `test/subsystem/placement_test.go` | Added 5 subsystem tests for rehydration |
| `go.mod` | Updated SPRM dependency (replace directive to `ygalblum/dcm-service-provider-manager@deffered-delete`) |
| `go.sum` | Updated checksums |

## API

### Endpoint

```
POST /api/v1alpha1/resources/{resourceId}:rehydrate
```

### Request Body

```json
{
  "new_instance_id": "<new-resource-id>"
}
```

### Responses

| Status | Meaning |
|--------|---------|
| 202 | Rehydration accepted, returns new `Resource` |
| 400 | Invalid request |
| 404 | Old resource not found |
| 406 | Policy rejected re-evaluation |
| 409 | New instance ID conflict or policy conflict |
| 422 | SPRM provider error |
| 5xx | Internal / SPRM / policy error |

## Test Coverage

### Unit tests (`internal/service/placement_test.go`)

- Happy path: rehydrate succeeds, old resource removed, new resource created
- Policy re-evaluates to different provider
- Original spec preserved, evaluated spec sent to SPRM
- Old resource not found -> NOT_FOUND
- Policy rejects (406) -> POLICY_REJECTED, old resource unchanged
- Policy fails (500) -> POLICY_INTERNAL_ERROR, old resource unchanged
- Policy returns empty provider -> POLICY_INTERNAL_ERROR
- New instance ID conflict -> CONFLICT, old resource unchanged
- SPRM creation fails -> error, new DB record rolled back, old resource unchanged
- SPRM deferred delete fails -> rehydration still succeeds (graceful degradation)

### Subsystem tests (`test/subsystem/placement_test.go`)

- Happy path with WireMock: create, rehydrate, verify old gone and new exists
- Resource not found -> 404
- Policy rejects -> 406, SPRM create not called
- SPRM creation fails -> 500, old resource preserved
- SPRM deferred delete fails -> 202, new resource created (graceful degradation)

## Dependencies

Requires SPRM with deferred deletion support: [dcm-project/service-provider-manager#43](https://github.com/dcm-project/service-provider-manager/pull/43). Currently using a `replace` directive in `go.mod` pointing to `ygalblum/dcm-service-provider-manager@deffered-delete`. This should be updated to the upstream version once the PR is merged.
