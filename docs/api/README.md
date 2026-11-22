# API specification

The OpenAPI specification lives in one place:
[`backend/handlers/openapi.yaml`](../../backend/handlers/openapi.yaml).

- It is embedded into the backend binary and served at `GET /api/docs/openapi.yaml`, with a Swagger UI at `GET /api/docs`.
- `backend/openapi_routes_test.go` fails CI if a route is registered in `backend/main.go` but missing from the spec, or the other way round. Paths in the spec are relative to the `/api/v1` and `/api/v2` servers. Root-mounted probes such as `/health` declare their own `servers`.
- When you add, rename or remove a route, update the spec in the same change. `./scripts/validate_openapi.sh` runs the same checks locally.
