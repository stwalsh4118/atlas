# Product Backlog - Property Boundary Viewer

## Project Overview
This backlog tracks all Product Backlog Items (PBIs) for the Property Boundary Viewer application - a geospatial web application for exploring property boundaries and ownership information.

**Project PRD**: [property-viewer-prd.md](../../property-viewer-prd.md)

## Backlog Items

| ID | Actor | User Story | Status | Conditions of Satisfaction (CoS) |
| :--- | :--- | :--- | :--- | :--- |
| 1 | Developer | As a developer, I want to set up the project infrastructure so that I have a working development environment | Agreed | [View Details](./1/prd.md) |
| 2 | Developer | As a developer, I want to create the database schema with spatial indexing so that I can store and query property data efficiently | Proposed | [View Details](./2/prd.md) |
| 3 | Developer | As a developer, I want to build a data import pipeline so that I can load county parcel data into the database | Proposed | [View Details](./3/prd.md) |
| 4 | Developer | As a developer, I want to build the core Go API backend so that frontends can query property data | Proposed | [View Details](./4/prd.md) |
| 5 | User | As a user, I want to query property information by clicking on a map location so that I can see ownership and boundary details | Proposed | [View Details](./5/prd.md) |
| 6 | User | As a user, I want to interact with a Next.js-based map interface so that I can explore properties with a modern web UI | Proposed | [View Details](./6/prd.md) |
| 7 | User | As a user, I want to interact with a Vue.js-based map interface so that I can explore properties using an alternative frontend implementation | Proposed | [View Details](./7/prd.md) |
| 8 | User | As a user, I want to search for properties by various criteria so that I can find specific parcels quickly | Proposed | [View Details](./8/prd.md) |

## Backlog History

| Timestamp | PBI_ID | Event_Type | Details | User |
| :--- | :--- | :--- | :--- | :--- |
| 20251019-000000 | N/A | Backlog Created | Initial backlog created from PRD | AI_Agent |
| 20251019-000001 | 1 | PBI Created | Project Infrastructure Setup PBI created | AI_Agent |
| 20251019-000002 | 2 | PBI Created | Database Schema and Spatial Indexing PBI created | AI_Agent |
| 20251019-000003 | 3 | PBI Created | Data Import Pipeline PBI created | AI_Agent |
| 20251019-000004 | 4 | PBI Created | Core Go API Backend PBI created | AI_Agent |
| 20251019-000005 | 5 | PBI Created | Point-in-Polygon Query API PBI created | AI_Agent |
| 20251019-000006 | 6 | PBI Created | Next.js Frontend with Map Integration PBI created | AI_Agent |
| 20251019-000007 | 7 | PBI Created | Vue.js Frontend with Leaflet PBI created | AI_Agent |
| 20251019-000008 | 8 | PBI Created | Search and Discovery Features PBI created | AI_Agent |
| 20251019-000009 | ALL | PBI Updated | Updated all PBIs to use latest technology versions (Go 1.25, PostgreSQL 18, Next.js 15, etc.) | AI_Agent |
| 20251019-120000 | 1 | Status Change | PBI 1 moved from Proposed to Agreed | User |

## Notes

- PBIs 1-7 represent the MVP scope as defined in the PRD
- PBI 8 is post-MVP "Should Have" features
- All PBIs are in "Proposed" status awaiting User approval
- Priority order reflects the dependency chain: infrastructure → database → data import → API → frontends → advanced features

