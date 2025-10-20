# Tasks for PBI 2: Database Schema and Spatial Indexing

This document lists all tasks associated with PBI 2.

**Parent PBI**: [PBI 2: Database Schema and Spatial Indexing](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :------ | :---------- |
| 2-1 | [Research and document GORM + PostGIS integration](./2-1.md) | Done | Research how to integrate GORM with PostGIS geometry types and document approach |
| 2-2 | [Set up golang-migrate for database migrations](./2-2.md) | Done | Install and configure golang-migrate with migration structure |
| 2-3 | [Create initial migration with PostGIS extension](./2-3.md) | Proposed | Create migration to enable PostGIS extension in database |
| 2-4 | [Create GORM model with custom geometry type](./2-4.md) | Proposed | Define TaxParcel GORM model with custom Polygon geometry type |
| 2-5 | [Create tax_parcels table schema migration](./2-5.md) | Proposed | Create migration for tax_parcels table with all columns and spatial index |
| 2-6 | [Create standard indexes migration](./2-6.md) | Proposed | Create migration for standard B-tree indexes on common query columns |
| 2-7 | [E2E CoS Test - Schema validation and testing](./2-7.md) | Proposed | End-to-end testing to verify all acceptance criteria are met |

