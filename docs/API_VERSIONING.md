# API Versioning Guide

## Overview

The Gpay-Remit API supports versioning to ensure backward compatibility when introducing breaking changes. This document explains how to use API versions and migrate between them.

## Supported Versions

- **v1**: Legacy version (Deprecated)
- **v2**: Current version (Recommended)

## Version Negotiation

You can specify the API version in three ways:

### 1. URL Path (Recommended)

```
GET /api/v1/remittances
GET /api/v2/remittances
```

### 2. X-API-Version Header

```
GET /api/remittances
X-API-Version: v2
```

### 3. Accept-Version Header

```
GET /api/remittances
Accept-Version: v2
```

## Default Behavior

If no version is specified, the API defaults to **v2** (the current version).

## Versioning Policy

The rules below are enforced by `backend/middleware/versioning.go`. The list of versions it accepts is `SupportedAPIVersions`.

1. **Only known versions are accepted.** The supported versions are `v1` and `v2`. Any other value in the path, `X-API-Version` or `Accept-Version` gets `400 UNSUPPORTED_API_VERSION`. This includes `/api/v3/...`, `v2.1` and `latest`. An unknown version is never treated as the default.
2. **Values are normalized.** Header values are case-insensitive, and a missing `v` is added, so `V2` and `2` both mean `v2`. A blank header counts as absent.
3. **Every version in a request must match.** If the path and the headers name different versions, the request gets `400 UNSUPPORTED_API_VERSION`. For example, `/api/v1/...` with `X-API-Version: v2` is rejected. Otherwise the request would go to a v1 handler while reporting v2.
4. **Deprecated versions keep working until their sunset date.** Every response to a deprecated version includes the standard deprecation headers listed below.
5. **New versions are added by updating `SupportedAPIVersions`.** A version is deprecated by adding its deprecation and sunset dates to the `deprecations` table in the middleware. No other code needs to change.

### Unsupported or conflicting version

```http
GET /api/v3/remittances
```

```json
{
  "error": {
    "code": "UNSUPPORTED_API_VERSION",
    "message": "Unsupported API version",
    "details": {
      "requested_version": "v3",
      "source": "path",
      "supported_versions": ["v1", "v2"]
    }
  }
}
```

For conflicting versions, the message is `"Conflicting API versions in request"`, and `details.versions` lists the value each source sent, for example `{"path": "v1", "X-API-Version": "v2"}`.

## Deprecation Warnings

Responses to a deprecated version (v1) include the standard deprecation headers:

```
Deprecation: @1798675200
Sunset: Wed, 30 Jun 2027 00:00:00 GMT
Link: </docs/migration>; rel="deprecation"; type="text/html"
```

- `Deprecation` follows RFC 9745. The value is the deprecation date as a Unix timestamp, here 2026-12-31.
- `Sunset` follows RFC 8594 and gives the date the version stops working.
- `Link` points to the migration guide.

The legacy `X-API-*` headers are still sent for existing clients:

```
X-API-Deprecation-Warning: This API version is deprecated
X-API-Deprecation-Date: 2026-12-31
X-API-Sunset-Date: 2027-06-30
X-API-Deprecation-Info: Please migrate to v2. See documentation at /docs/migration
```

## Response Headers

All API responses include the version that was used:

```
X-API-Version: v2
```

## Error Response Format

All API errors return a standardized payload using the `ErrorResponse` structure:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "A human readable summary of the error",
    "details": {
      "field": "description"
    }
  }
}
```

- `code` is a stable error code such as `VALIDATION_ERROR`, `NOT_FOUND`, `UNAUTHORIZED`, or `INTERNAL_ERROR`.
- `message` is a client-friendly error summary.
- `details` is optional and may include validation metadata or extra information.

## Version-Specific Endpoints

Some endpoints may only be available in specific versions. If you request an endpoint that's not available in your requested version, you'll receive:

```json
{
  "error": "This endpoint is not available in the requested API version",
  "current_version": "v1",
  "required_versions": ["v2"]
}
```

## Migration from v1 to v2

### Breaking Changes

1. Enhanced analytics endpoints with better caching
2. Improved error response formats
3. Additional validation on request parameters

### Migration Steps

1. Update your client to specify `v2` in the URL or headers
2. Test all endpoints in a staging environment
3. Update error handling to accommodate new error formats
4. Deploy to production

### Example Migration

**Before (v1):**
```bash
curl -H "Authorization: Bearer TOKEN" \
  https://api.gpay-remit.com/api/v1/remittances
```

**After (v2):**
```bash
curl -H "Authorization: Bearer TOKEN" \
  https://api.gpay-remit.com/api/v2/remittances
```

## Timeline

- **Now**: v2 is stable and recommended
- **2026-12-31**: v1 officially deprecated
- **2027-06-30**: v1 sunset (no longer supported)

## Support

For questions or assistance with migration, please contact the development team or refer to the main API documentation.
