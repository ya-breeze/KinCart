# KinCart Implementation Plan - [READY]

This document provides a detailed technical roadmap for building the KinCart web application.

## 1. Technology Stack - [READY]

| Component | Technology | Description | Status |
| :--- | :--- | :--- | :--- |
| **Backend** | Go (Golang) | High-performance compiled language for the API. | **[READY]** |
| **Web Framework** | Gin | Lightweight and fast HTTP web framework. | **[READY]** |
| **Database** | SQLite | Embedded, file-based database. | **[READY]** |
| **ORM** | GORM | Objective-Relational Mapping for Go. | **[READY]** |
| **Frontend** | React + Vite | Modern, fast reactive UI. | **[READY]** |
| **Styling** | Vanilla CSS | Custom, premium design. | **[READY]** |
| **Deployment** | Docker | Containerized environment. | **[READY]** |
| **Linting** | golangci-lint | Go linter runner. | **[READY]** |
| **Storage** | Local FS | For local photo storage. | **[NOT IMPLEMENTED]** |
| **AI** | OpenAI/Gemini | For receipt parsing (Future). | **[NOT IMPLEMENTED]** |

## 2. Database Schema (GORM Models) - [PARTIAL]

### `Family` - **[READY]**
- `ID` (Primary Key)
- `Name` (String)
- `Currency` (String, Default: "₽") - **[NOT IMPLEMENTED]**

### `Shop` - **[NOT IMPLEMENTED]**
- `ID` (Primary Key)
- `Name` (String)
- `FamilyID` (Foreign Key)

### `ShopCategoryOrder` - **[NOT IMPLEMENTED]**
- `ShopID` (Foreign Key)
- `CategoryID` (Foreign Key)
- `SortOrder` (Integer)

### `Category` - **[READY]**
- `ID` (Primary Key)
- `Name` (String)
- `SortOrder` (Integer) - Default order

### `User` - **[READY]**
- `ID` (Primary Key)
- `Username` (String, Unique)
- `PasswordHash` (String)
- `FamilyID` (Foreign Key)

### `ShoppingList` - **[READY]**
- `ID` (Primary Key)
- `Title` (String)
- `FamilyID` (Foreign Key)
- `Status` (Enum: Planning, In-Progress, Completed)
- `EstimatedAmount` (Float)
- `ActualAmount` (Float)

### `Item` - **[PARTIAL]**
- `ID` (Primary Key) - **[READY]**
- `Name` (String) - **[READY]**
- `Description` (String) - **[READY]**
- `ListID` (Foreign Key) - **[READY]**
- `CategoryID` (Foreign Key) - **[READY]**
- `IsBought` (Boolean) - **[READY]**
- `LocalPhotoPath` (String) - **[NOT IMPLEMENTED]** (Replaces PhotoURL)
- `Price` (Float) - **[READY]**
- `IsUrgent` (Boolean) - **[READY]**

### `ItemFrequency` - **[NOT IMPLEMENTED]**
- `FamilyID` (Foreign Key)
- `ItemName` (String)
- `Frequency` (Integer)

## 3. API Design (REST) - [PARTIAL]

### Auth - **[READY]**
- `POST /api/auth/login`
- `GET /api/auth/me`

### Lists - **[PARTIAL]**
- `GET /api/lists`
- `POST /api/lists`
- `PATCH /api/lists/:id`
- `POST /api/lists/:id/duplicate` - **[NOT IMPLEMENTED]**

### Items - **[PARTIAL]**
- `POST /api/lists/:id/items`
- `PATCH /api/items/:id`
- `DELETE /api/items/:id`
- `GET /api/items/frequent` - **[NOT IMPLEMENTED]**
- `POST /api/items/:id/photo` - **[NOT IMPLEMENTED]**

### Categories & Shops - **[PARTIAL]**
- `GET /api/categories` - **[READY]**
- `POST /api/categories` - **[NOT IMPLEMENTED]**
- `PATCH /api/categories/:id` - **[NOT IMPLEMENTED]**
- `GET /api/shops` - **[NOT IMPLEMENTED]**
- `POST /api/shops` - **[NOT IMPLEMENTED]**
- `PATCH /api/shops/:id/order` - **[NOT IMPLEMENTED]**

### Config
- `GET /api/config` - Get family currency, etc. - **[NOT IMPLEMENTED]**
- `PATCH /api/config` - Update currency, etc. - **[NOT IMPLEMENTED]**

## 4. Development Phases

### Phase 1: Core Backend & DB - **[READY]**
- Setup Go environment.
- Initial GORM models.
- Auth system.

### Phase 2: Fundamental API - **[READY]**
- CRUD for Lists and Items.
- Basic Category support.

### Phase 3: Advanced Frontend & UX - **[PARTIAL]**
- **[READY]** Dashboard & Mode Switching.
- **[READY]** List Details & Grouping.
- **[NOT IMPLEMENTED]** List Template/Reuse (Duplication).
- **[NOT IMPLEMENTED]** Frequently Used Items panel.

### Phase 4: Media & Configuration - **[NOT IMPLEMENTED]**
- **[NOT IMPLEMENTED]** Local photo storage (server-side).
- **[NOT IMPLEMENTED]** Configurable Currencies.
- **[NOT IMPLEMENTED]** Category Management UI.

### Phase 5: Shop Optimization & AI (Future) - **[NOT IMPLEMENTED]**
- **[NOT IMPLEMENTED]** Shop-specific category ordering.
- **[NOT IMPLEMENTED]** Receipt parsing with AI.

