# Clean Architecture Invariant

## Strict Boundary Rules

1. **Dependency Inversion**: Dependencies point strictly inwards:
   `Infrastructure → Delivery → Usecase → Domain`.
2. **Domain Layer (`backend/internal/domain/`)**:
   - Contains pure Go entities (`User`, `Role`, `UserRole`, `Department`, `Position`, `Holiday`, `HolidayCategory`, `AuditLog`, `Tenant`), repository interfaces (ports), and domain errors.
   - **Never import Gin, GORM, database/sql, or any external framework in `domain`.**
3. **Usecase Layer (`backend/internal/usecase/`)**:
   - Contains application business rules.
   - Coordinates domain entities, repositories, and domain services.
   - Transport-agnostic (no Gin `*gin.Context` or HTTP concepts).
4. **Delivery Layer (`backend/internal/delivery/http/`)**:
   - Thin Gin handlers that bind & validate input DTOs, invoke usecases, and write responses using `response.Success/Created/Paginated/Error` helpers.
   - **Handlers must NEVER call GORM or database queries directly.**
5. **Infrastructure Layer (`backend/internal/infrastructure/`)**:
   - Implements domain repository interfaces using GORM (`persistence/`).
   - Handles external services (JWT, bcrypt, mailer) and dependency injection wiring (`container/`).
