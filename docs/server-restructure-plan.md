# Server Restructuring Plan - Clean Architecture Implementation

## Overview
Restructure the current server daemon (handles CRUD operations for CLI agent) following the [go-clean-template](https://github.com/evrone/go-clean-template) pattern to implement Clean Architecture principles.

## Current Structure Analysis

```
server/
├── data/
│   ├── conversation/
│   │   ├── conversation.go    # Mixed: entity + repo implementation
│   │   ├── types.go            # ConversationMetadata entity
│   │   ├── schema.sql
│   │   └── conversation_test.go
│   └── plan/
│       ├── plan.go             # Mixed: entity + repo implementation
│       └── schema.sql
├── db/
│   └── db.go                   # Database initialization
├── models.go                   # Model registry (factory pattern)
└── server.go                   # HTTP handlers + routing + DI
```

### Current Issues
1. **Mixing concerns**: Entity definitions mixed with repository implementations in same files
2. **No use case layer**: Business logic scattered in HTTP handlers and model methods
3. **Tight coupling**: HTTP handlers directly call data models
4. **No interfaces**: Cannot swap implementations or mock easily
5. **Hard to test**: Business logic embedded in handlers
6. **No configuration abstraction**: Hardcoded paths and configs

## Target Structure (Clean Architecture)

```
server/
├── internal/                           # Private application code
│   ├── app/                           # Application initialization & DI container
│   │   └── app.go                     # Wire all dependencies together
│   │
│   ├── entity/                        # Domain models (innermost layer)
│   │   ├── conversation.go            # Conversation entity + core methods
│   │   └── plan.go                    # Plan entity + core methods
│   │
│   ├── usecase/                       # Business logic layer
│   │   ├── conversation/
│   │   │   ├── conversation.go        # Conversation use cases
│   │   │   └── conversation_test.go
│   │   ├── plan/
│   │   │   ├── plan.go                # Plan use cases
│   │   │   └── plan_test.go
│   │   └── contracts.go               # Use case interfaces
│   │
│   ├── repo/                          # Repository layer (outer)
│   │   ├── sqlite/                    # SQLite implementations
│   │   │   ├── conversation.go
│   │   │   └── plan.go
│   │   └── contracts.go               # Repository interfaces
│   │
│   └── controller/                    # Delivery/Transport layer
│       └── http/
│           ├── v1/
│           │   ├── conversation.go    # Conversation HTTP handlers
│           │   ├── plan.go            # Plan HTTP handlers
│           │   ├── router.go          # v1 route registration
│           │   └── types.go           # Request/response DTOs
│           └── router.go              # Main HTTP router setup
│
├── pkg/                               # Reusable packages
│   ├── sqlite/                        # SQLite connection wrapper
│   │   └── sqlite.go
│   ├── httpserver/                    # HTTP server wrapper
│   │   └── server.go
│   └── logger/                        # Logger interface (future)
│       └── logger.go
│
├── config/                            # Configuration
│   └── config.go                      # Config struct + env loading
│
├── migrations/                        # Database migrations
│   ├── 001_conversations.sql
│   └── 002_plans.sql
│
└── cmd/                               # Server entry point (if separate from main CLI)
    └── server/
        └── main.go
```

## Migration Steps

### Phase 1: Extract Entities (Domain Layer)
**Goal**: Pure domain models with no external dependencies

- [ ] Create `server/internal/entity/conversation.go`
  - Move `Conversation` struct from `data/conversation/conversation.go`
  - Move `ConversationMetadata` from `data/conversation/types.go`
  - Keep `New()` factory and `Append()` methods (pure business logic)
  - Remove database dependencies

- [ ] Create `server/internal/entity/plan.go`
  - Move `Plan`, `PlanInfo`, `Step` structs from `data/plan/plan.go`
  - Keep pure business methods only
  - Remove database dependencies

**Success Criteria**: Entity package imports only standard library + `message` package

### Phase 2: Define Repository Interfaces
**Goal**: Interface contracts for data access

- [ ] Create `server/internal/repo/contracts.go`
  ```go
  type ConversationRepo interface {
      Save(ctx context.Context, conv *entity.Conversation) error
      Load(ctx context.Context, id string) (*entity.Conversation, error)
      List(ctx context.Context) ([]entity.ConversationMetadata, error)
      LatestID(ctx context.Context) (string, error)
  }
  
  type PlanRepo interface {
      // Define plan repository methods
  }
  ```

### Phase 3: Implement SQLite Repositories
**Goal**: Concrete repository implementations

- [ ] Create `server/internal/repo/sqlite/conversation.go`
  - Move `ConversationModel` logic from `data/conversation/conversation.go`
  - Implement `ConversationRepo` interface
  - Inject `*sql.DB` via constructor
  - Add context support to all methods

- [ ] Create `server/internal/repo/sqlite/plan.go`
  - Move `PlanModel` logic from `data/plan/plan.go`
  - Implement `PlanRepo` interface

### Phase 4: Create Use Case Layer
**Goal**: Encapsulate business logic independent of delivery mechanism

- [ ] Create `server/internal/usecase/contracts.go`
  ```go
  type ConversationUseCase interface {
      CreateConversation(ctx context.Context) (*entity.Conversation, error)
      GetConversation(ctx context.Context, id string) (*entity.Conversation, error)
      ListConversations(ctx context.Context) ([]entity.ConversationMetadata, error)
      SaveConversation(ctx context.Context, conv *entity.Conversation) error
  }
  ```

- [ ] Create `server/internal/usecase/conversation/conversation.go`
  ```go
  type UseCase struct {
      repo repo.ConversationRepo
  }
  
  func New(r repo.ConversationRepo) *UseCase {
      return &UseCase{repo: r}
  }
  
  func (uc *UseCase) CreateConversation(ctx context.Context) (*entity.Conversation, error) {
      conv, err := entity.NewConversation()
      if err != nil {
          return nil, fmt.Errorf("usecase - CreateConversation: %w", err)
      }
      
      if err := uc.repo.Save(ctx, conv); err != nil {
          return nil, fmt.Errorf("usecase - CreateConversation - repo.Save: %w", err)
      }
      
      return conv, nil
  }
  // ... other methods
  ```

- [ ] Create `server/internal/usecase/plan/plan.go`
  - Implement plan business logic

### Phase 5: Create HTTP Controller Layer
**Goal**: HTTP transport layer that uses use cases

- [ ] Create `server/internal/controller/http/v1/types.go`
  ```go
  // Request/Response DTOs
  type CreateConversationResponse struct {
      ID string `json:"id"`
  }
  
  type SaveConversationRequest struct {
      ID       string              `json:"id"`
      Messages []*message.Message  `json:"messages"`
  }
  ```

- [ ] Create `server/internal/controller/http/v1/conversation.go`
  ```go
  type Handler struct {
      conversationUC usecase.ConversationUseCase
  }
  
  func NewConversationHandler(uc usecase.ConversationUseCase) *Handler {
      return &Handler{conversationUC: uc}
  }
  
  func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
      ctx := r.Context()
      
      conv, err := h.conversationUC.CreateConversation(ctx)
      if err != nil {
          // Error handling
          return
      }
      
      // Response
      json.NewEncoder(w).Encode(CreateConversationResponse{ID: conv.ID})
  }
  ```

- [ ] Create `server/internal/controller/http/v1/router.go`
  - Register v1 routes

- [ ] Create `server/internal/controller/http/router.go`
  - Setup main HTTP router
  - Mount v1 routes at `/api/v1`

### Phase 6: Configuration Management
**Goal**: Externalized configuration

- [ ] Create `server/config/config.go`
  ```go
  type Config struct {
      Server ServerConfig
      DB     DBConfig
  }
  
  type ServerConfig struct {
      Port string `env:"SERVER_PORT" envDefault:"11435"`
  }
  
  type DBConfig struct {
      DSN string `env:"DB_DSN"`
  }
  
  func New() (*Config, error) {
      // Parse from env or use defaults
  }
  ```

### Phase 7: Infrastructure Packages
**Goal**: Reusable infrastructure components

- [ ] Create `server/pkg/sqlite/sqlite.go`
  - Move database initialization from `db/db.go`
  - Connection pooling wrapper

- [ ] Create `server/pkg/httpserver/server.go`
  - HTTP server wrapper with graceful shutdown

### Phase 8: Dependency Injection & App Initialization
**Goal**: Wire everything together

- [ ] Create `server/internal/app/app.go`
  ```go
  func Run(cfg *config.Config) error {
      // 1. Initialize infrastructure
      db, err := sqlite.New(cfg.DB.DSN)
      if err != nil {
          return fmt.Errorf("app - Run - sqlite.New: %w", err)
      }
      defer db.Close()
      
      // Run migrations
      if err := db.Migrate(); err != nil {
          return fmt.Errorf("app - Run - db.Migrate: %w", err)
      }
      
      // 2. Initialize repositories
      conversationRepo := sqliteRepo.NewConversationRepo(db)
      planRepo := sqliteRepo.NewPlanRepo(db)
      
      // 3. Initialize use cases
      conversationUC := conversationUC.New(conversationRepo)
      planUC := planUC.New(planRepo)
      
      // 4. Initialize HTTP handlers
      httpServer := httpserver.New(cfg.Server.Port)
      httpController.NewRouter(httpServer.Mux(), conversationUC, planUC)
      
      // 5. Start server
      return httpServer.Start()
  }
  ```

- [ ] Update `server/server.go` or create new entry point
  ```go
  func Serve(ln net.Listener) error {
      cfg, err := config.New()
      if err != nil {
          return err
      }
      
      return app.Run(cfg)
  }
  ```

### Phase 9: Testing
**Goal**: Comprehensive test coverage

- [ ] Unit tests for use cases with mocked repositories
- [ ] Integration tests for repositories with test database
- [ ] HTTP handler tests with mocked use cases

### Phase 10: Migration & Cleanup
**Goal**: Remove old code

- [ ] Migrate all SQL schemas to `migrations/` folder
- [ ] Delete old `server/data/` directory
- [ ] Delete old `server/db/` directory
- [ ] Delete old `server/models.go`
- [ ] Update all imports throughout the project

## Benefits of New Structure

1. **Testability**: Each layer can be tested independently with mocks
2. **Maintainability**: Clear separation of concerns
3. **Flexibility**: Easy to add new transport layers (gRPC, CLI commands)
4. **Scalability**: Can extract use cases to microservices later
5. **Clean dependencies**: Inner layers don't depend on outer layers
6. **Version management**: Can add v2 API without breaking v1

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│          HTTP Controller (v1)                   │
│   conversation.go │ plan.go │ router.go         │
└─────────────────┬───────────────────────────────┘
                  │ calls via interface
┌─────────────────▼───────────────────────────────┐
│            Use Case Layer                       │
│   ConversationUseCase │ PlanUseCase             │
└─────────────────┬───────────────────────────────┘
                  │ uses entities
┌─────────────────▼───────────────────────────────┐
│            Entity Layer                         │
│   Conversation │ Plan │ ConversationMetadata    │
└─────────────────▲───────────────────────────────┘
                  │ returns entities
┌─────────────────┴───────────────────────────────┐
│         Repository Layer (SQLite)               │
│   ConversationRepo │ PlanRepo                   │
└─────────────────────────────────────────────────┘
```

## Notes

- Keep backward compatibility during migration
- Use feature flags if needed for gradual rollout
- Update AGENT.md with new structure once complete
- Consider adding middleware for logging, auth, metrics later
- Context should flow through all layers for cancellation/timeout
