# Tasks for PBI 5: Point-in-Polygon Query API

This document lists all tasks associated with PBI 5.

**Parent PBI**: [PBI 5: Point-in-Polygon Query API](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :------ | :---------- |
| 5-1 | [Create parcel repository with FindByPoint method](./5-1.md) | Done | Implement repository layer with ST_Contains spatial query |
| 5-2 | [Create parcel service with GetParcelAtPoint method](./5-2.md) | Done | Implement service layer for point-in-polygon business logic |
| 5-3 | [Implement at-point endpoint with DTOs and validation](./5-3.md) | Done | Create handler, request/response DTOs, validation, and route registration |
| 5-4 | [Extend repository and service for nearby query](./5-4.md) | Proposed | Add FindNearby and GetNearbyParcels methods with ST_DWithin |
| 5-5 | [Implement nearby endpoint with DTOs and validation](./5-5.md) | Proposed | Create handler, DTOs, validation for nearby properties endpoint |
| 5-6 | [Extend repository and service for get-by-id query](./5-6.md) | Proposed | Add FindByID and GetParcelByID methods |
| 5-7 | [Implement get-by-id endpoint with DTOs and validation](./5-7.md) | Proposed | Create handler, DTOs, validation for parcel lookup endpoint |
| 5-8 | [Query performance optimization and monitoring](./5-8.md) | Proposed | Verify spatial index usage, add query timing logs, performance testing |
| 5-9 | [E2E CoS Test for Point-in-Polygon Query API](./5-9.md) | Proposed | End-to-end test verifying all acceptance criteria are met |

