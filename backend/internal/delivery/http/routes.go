// Package http assembles the Gin engine, base middleware, and route registration.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/middleware"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	"github.com/skryfon-org/employee-360/backend/shared"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"strings"
)

// SetupRouter builds and returns a fully configured Gin engine with middleware and routes.
func SetupRouter(cfg *config.Config, log zerolog.Logger, ctr *container.AppContainer) *gin.Engine {
	engine := gin.New()

	// Trust no proxy unless configured: otherwise any client could spoof
	// X-Forwarded-For to dodge the per-IP rate limiter (gin trusts all by default).
	if err := engine.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Fatal().Err(err).Msg("invalid server.trusted_proxies")
	}

	// Base middleware chain: request_id -> logger -> cors -> recovery.
	engine.Use(
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.CORS(cfg.CORS.AllowedOrigins),
		middleware.Recovery(log),
	)

	// One resolver shared by both per-IP limiters (see middleware.NewClientIPResolver).
	clientIP := middleware.NewClientIPResolver(cfg.Server.TrustedProxies, log)

	// Global rate limiting (after recovery so 429s still carry request-id/CORS headers).
	if cfg.RateLimit.Enabled {
		limiter, _ := middleware.RateLimit(middleware.RateLimitConfig{
			RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
			Burst:             cfg.RateLimit.Burst,
			CleanupInterval:   cfg.RateLimit.CleanupInterval,
			IdleTTL:           cfg.RateLimit.IdleTTL,
			ClientIP:          clientIP,
			ExemptPaths:       []string{"/health", "/healthz", shared.APIVersionPrefix + "/health"},
		})
		engine.Use(limiter)
	}

	registerRoutes(engine, ctr, clientIP)

	return engine
}

// registerRoutes attaches all route groups to the engine.
func registerRoutes(engine *gin.Engine, c *container.Container, clientIP middleware.ClientIPFunc) {
	// Unversioned operational health checks for load balancers / Kubernetes / monitoring.
	if c.Health != nil && c.Health.Handler != nil {
		engine.GET("/health", c.Health.Handler.Health)
		engine.GET("/healthz", c.Health.Handler.Health)
	}

	// Strict limiter for unauthenticated credential/token endpoints (F-4). Counters
	// live in Redis when configured and fall back to memory otherwise.
	var throttle []gin.HandlerFunc
	if c.Config != nil && c.Config.AuthRateLimit.Enabled {
		a := c.Config.AuthRateLimit
		throttle = append(throttle, middleware.AuthThrottle(middleware.AuthThrottleConfig{
			PerAccountMax: a.PerAccountMax,
			PerIPMax:      a.PerIPMax,
			Window:        a.Window,
			Cache:         c.Cache,
			ClientIP:      clientIP,
		}))
	}
	limited := func(h gin.HandlerFunc) []gin.HandlerFunc { return append(append([]gin.HandlerFunc{}, throttle...), h) }

	v1 := engine.Group(shared.APIVersionPrefix)
	{
		// Versioned health check for client SDKs / smoke tests.
		if c.Health != nil && c.Health.Handler != nil {
			v1.GET("/health", c.Health.Handler.Health)
		}

		if c.Auth != nil && c.Auth.Handler != nil {
			authGroup := v1.Group("/auth")
			{
				authGroup.POST("/login", limited(c.Auth.Handler.Login)...)
				authGroup.POST("/refresh", c.Auth.Handler.Refresh)
				authGroup.POST("/forgot-password", limited(c.Auth.Handler.ForgotPassword)...)
				authGroup.POST("/reset-password", limited(c.Auth.Handler.ResetPassword)...)

				authProtected := authGroup.Group("", middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier), middleware.Tenant())
				{
					authProtected.POST("/logout", c.Auth.Handler.Logout)
				}
			}
		}

		// Admin-only role lookup (feeds the invitation form's role select).
		if c.Auth != nil && c.Auth.RoleHandler != nil {
			v1.GET("/roles",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
				c.Auth.RoleHandler.List,
			)
		}

		// Dashboards (read-only). Admin: tenant-scoped from the auth context.
		// Super admin: same tenant-scoped data plus tenant info/users by role, strictly super_admin.
		if c.Auth != nil && c.Auth.DashboardHandler != nil {
			dh := c.Auth.DashboardHandler
			v1.GET("/dashboard/admin",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin),
				dh.Admin,
			)
			v1.GET("/dashboard/super-admin",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleSuperAdmin),
				dh.SuperAdmin,
			)
			// Employee home: caller's own data; needs the complete-profile gate (admins pass it).
			if !dh.HasEmployee() {
				panic("routes: dashboard handler must have the employee usecase (WithEmployee) when the dashboard is configured")
			}
			if c.Employee == nil {
				panic("routes: Employee container (profile gate) is required for the employee dashboard route")
			}
			v1.GET("/dashboard/employee",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleEmployee, entity.RoleAdmin, entity.RoleSuperAdmin),
				middleware.RequireCompleteProfile(c.Employee.ProfileGate),
				dh.Employee,
			)
		}

		// Current-tenant settings (self-hosted: the caller's own tenant only), strictly
		// super_admin (Auth -> Tenant -> role guard). No tenant id in URL or body.
		if c.Auth != nil && c.Auth.TenantHandler != nil {
			th := c.Auth.TenantHandler
			tg := v1.Group("/tenant",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleSuperAdmin),
			)
			{
				tg.GET("", th.Get)
				tg.PATCH("", th.Update)
				tg.GET("/domains", th.ListDomains)
				tg.POST("/domains", th.AddDomain)
				tg.PATCH("/domains/:domainId", th.UpdateDomain)
				tg.DELETE("/domains/:domainId", th.RemoveDomain)
			}
		}

		if c.Auth != nil && c.Auth.InvitationHandler != nil {
			ih := c.Auth.InvitationHandler

			// Unauthenticated: the invitee proves possession of the emailed token.
			v1.POST("/invitations/accept", limited(ih.Accept)...)
			v1.GET("/invitations/validate", limited(ih.Validate)...)

			// Admin-only management routes: auth -> tenant -> role check.
			invGroup := v1.Group("/users/invitations",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				invGroup.POST("", ih.Invite)
				invGroup.GET("", ih.List)
				invGroup.POST("/:id/resend", ih.Resend)
				invGroup.DELETE("/:id", ih.Revoke)
			}
		}

		// Departments: tenant-scoped department CRUD (admin/super_admin only).
		if c.Department != nil && c.Department.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			dh := c.Department.Handler
			deptGroup := v1.Group("/departments",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				deptGroup.POST("", dh.Create)
				deptGroup.GET("", dh.List)
				deptGroup.GET("/:id", dh.GetByID)
				deptGroup.PUT("/:id", dh.Update)
				deptGroup.DELETE("/:id", dh.Delete)
			}
		}

		// Positions: tenant-scoped position CRUD (admin/super_admin only).
		if c.Position != nil && c.Position.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			ph := c.Position.Handler
			posGroup := v1.Group("/positions",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				posGroup.POST("", ph.Create)
				posGroup.GET("", ph.List)
				posGroup.GET("/:id", ph.GetByID)
				posGroup.PUT("/:id", ph.Update)
				posGroup.DELETE("/:id", ph.Delete)
			}
		}

		// Employees (admin) and /me (self-service profile, first login).
		// /me and its profile/emergency-contacts routes are deliberately NOT behind
		// the profile gate so an incomplete profile can be completed.
		if c.Employee != nil && c.Employee.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			eh := c.Employee.Handler
			auth := middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier)

			empGroup := v1.Group("/employees", auth, middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin))
			{
				empGroup.GET("", eh.List)
				empGroup.GET("/:id", eh.Get)
				empGroup.PUT("/:id", eh.Update)
				empGroup.PATCH("/:id", eh.Update)
				empGroup.GET("/:id/assignments", eh.ListAssignments)
				if c.Project != nil && c.Project.AllocationHandler != nil {
					empGroup.GET("/:id/allocations", c.Project.AllocationHandler.ListEmployeeAllocations)
				}
				if c.StatusLog != nil && c.StatusLog.Handler != nil {
					empGroup.GET("/:id/status-logs", c.StatusLog.Handler.ListEmployeeStatusLogs)
				}
				empGroup.POST("/:id/deactivate", eh.Deactivate)
				empGroup.POST("/:id/activate", eh.Activate)
				empGroup.GET("/:id/emergency-contacts", eh.ListContacts)
				empGroup.POST("/:id/emergency-contacts", eh.CreateContact)
				empGroup.PUT("/:id/emergency-contacts/:contactId", eh.UpdateContact)
				empGroup.DELETE("/:id/emergency-contacts/:contactId", eh.DeleteContact)
				// Granting/revoking admin is super_admin only.
				empGroup.PUT("/:id/role", middleware.RequireRole(entity.RoleSuperAdmin), eh.ChangeRole)
			}

			meGroup := v1.Group("/me", auth, middleware.Tenant())
			{
				meGroup.GET("", eh.Me)
				meGroup.GET("/profile", eh.GetMyProfile)
				if c.Project != nil && c.Project.AllocationHandler != nil {
					meGroup.GET("/projects", c.Project.AllocationHandler.ListMyProjects)
				}
				if c.StatusLog != nil && c.StatusLog.Handler != nil {
					meGroup.GET("/status-logs", c.StatusLog.Handler.ListMyStatusLogs)
					meGroup.GET("/status-logs/day-statuses", c.StatusLog.Handler.ListMyDayStatuses)
					meGroup.POST("/status-logs", c.StatusLog.Handler.CreateMyStatusLog)
					meGroup.PATCH("/status-logs/:id", c.StatusLog.Handler.UpdateMyStatusLog)
					meGroup.DELETE("/status-logs/:id", c.StatusLog.Handler.DeleteMyStatusLog)
				}
				meGroup.PATCH("/profile", eh.UpdateMyProfile)
				meGroup.GET("/emergency-contacts", eh.ListMyContacts)
				meGroup.POST("/emergency-contacts", eh.CreateMyContact)
				meGroup.PUT("/emergency-contacts/:contactId", eh.UpdateMyContact)
				meGroup.DELETE("/emergency-contacts/:contactId", eh.DeleteMyContact)
			}
		}

		// Projects and allocations (admin/super_admin only). The employee-facing
		// reads (/employees/:id/allocations, /me/projects) are registered above.
		if c.Project != nil && c.Project.Handler != nil && c.Project.AllocationHandler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			prh := c.Project.Handler
			alh := c.Project.AllocationHandler
			adminOnly := []gin.HandlerFunc{
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			}
			prjGroup := v1.Group("/projects", adminOnly...)
			{
				prjGroup.POST("", prh.CreateProject)
				prjGroup.GET("", prh.ListProjects)
				prjGroup.GET("/code-suggestion", prh.SuggestProjectCode)
				prjGroup.GET("/:id", prh.GetProject)
				prjGroup.PATCH("/:id", prh.UpdateProject)
				prjGroup.GET("/:id/allocations", alh.ListProjectAllocations)
				prjGroup.POST("/:id/allocations", alh.CreateAllocation)
			}
			allocGroup := v1.Group("/allocations", adminOnly...)
			{
				allocGroup.PATCH("/:id", alh.UpdateAllocation)
				allocGroup.DELETE("/:id", alh.DeleteAllocation)
			}
		}

		// Status logs: daily summary (admin/super_admin only).
		if c.StatusLog != nil && c.StatusLog.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			slh := c.StatusLog.Handler
			slGroup := v1.Group("/status-logs",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				slGroup.GET("/daily", slh.GetDailyStatusLogs)
				slGroup.GET("/monthly", slh.GetMonthlyStatusLogs)
			}
		}

		// Leave: types (read by any user; write admin only), self-service requests
		// under /me, approver inbox and admin listing. Decision authority (chosen
		// approver or tenant admin, never the applicant) is enforced in the usecase.
		if c.Leave != nil && c.Leave.Handler != nil && c.Leave.RequestHandler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			lth := c.Leave.Handler
			lrh := c.Leave.RequestHandler
			auth := middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier)
			anyUser := middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin, entity.RoleEmployee)
			adminOnly := middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin)

			ltGroup := v1.Group("/leave-types", auth, middleware.Tenant())
			{
				ltGroup.GET("", anyUser, lth.ListLeaveTypes)
				ltGroup.POST("", adminOnly, lth.CreateLeaveType)
				ltGroup.PATCH("/:id", adminOnly, lth.UpdateLeaveType)
			}

			leaveMe := v1.Group("/me", auth, middleware.Tenant(), anyUser)
			{
				leaveMe.GET("/leave-approvers", lrh.ListMyLeaveApprovers)
				leaveMe.GET("/leave-requests", lrh.ListMyLeaveRequests)
				leaveMe.POST("/leave-requests", lrh.ApplyLeave)
				leaveMe.POST("/leave-requests/:id/cancel", lrh.CancelMyLeaveRequest)
			}

			lrGroup := v1.Group("/leave-requests", auth, middleware.Tenant(), anyUser)
			{
				lrGroup.GET("/team", lrh.ListTeamLeaveRequests)
				lrGroup.GET("", adminOnly, lrh.ListAllLeaveRequests)
				lrGroup.POST("/:id/approve", lrh.ApproveLeaveRequest)
				lrGroup.POST("/:id/reject", lrh.RejectLeaveRequest)
			}
		}

		// Organization settings (employee code format).
		if c.Organization != nil && c.Organization.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			oh := c.Organization.Handler
			orgGroup := v1.Group("/organization/settings",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				orgGroup.GET("", oh.GetSettings)
				orgGroup.PUT("", oh.UpdateSettings)
			}

			// Employee-readable slice: weekly off days only, for the caller's tenant.
			if c.Organization.WeeklyOffDays != nil {
				offGroup := v1.Group("/organization/weekly-off-days",
					middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
					middleware.Tenant(),
					middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin, entity.RoleEmployee),
				)
				offGroup.GET("", c.Organization.WeeklyOffDays.GetWeeklyOffDays)
			}
		}

		// Audit log viewer: read-only, tenant-scoped (admin/super_admin only).
		if c.AuditLog != nil && c.AuditLog.Handler != nil && c.Auth != nil && c.Auth.TokenService != nil {
			ah := c.AuditLog.Handler
			auditGroup := v1.Group("/audit-logs",
				middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
				middleware.Tenant(),
				middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin),
			)
			{
				auditGroup.GET("", ah.List)
				auditGroup.GET("/:id", ah.GetByID)
			}
		}

		// Holiday Calendar: holidays, categories, settings
		if c.Holiday != nil {
			if c.Holiday.Handler == nil {
				panic("routes: holiday handler must not be nil when holiday container is configured")
			}
			if c.Auth != nil && c.Auth.TokenService != nil {
				hh := c.Holiday.Handler
				if c.Employee == nil {
					panic("routes: Employee container (profile gate) is required for holiday routes")
				}
				gate := middleware.RequireCompleteProfile(c.Employee.ProfileGate)

				holidayGroup := v1.Group("/holidays",
					middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
					middleware.Tenant(),
				)
				{
					holidayRead := holidayGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin, entity.RoleEmployee), gate)
					{
						holidayRead.GET("", hh.ListHolidays)
						holidayRead.GET("/:id", hh.GetHolidayByID)
					}
					holidayWrite := holidayGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin))
					{
						holidayWrite.POST("", hh.CreateHoliday)
						holidayWrite.PUT("/:id", hh.UpdateHoliday)
						holidayWrite.DELETE("/:id", hh.DeleteHoliday)
					}
				}

				catGroup := v1.Group("/holiday-categories",
					middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
					middleware.Tenant(),
				)
				{
					catRead := catGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin, entity.RoleEmployee), gate)
					{
						catRead.GET("", hh.ListCategories)
						catRead.GET("/:id", hh.GetCategoryByID)
					}
					catWrite := catGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin))
					{
						catWrite.POST("", hh.CreateCategory)
						catWrite.PUT("/:id", hh.UpdateCategory)
						catWrite.DELETE("/:id", hh.DeleteCategory)
					}
				}

				settingsGroup := v1.Group("/holiday-calendar/settings",
					middleware.Auth(c.Auth.TokenService, c.Auth.IdentityVerifier),
					middleware.Tenant(),
				)
				{
					settingsRead := settingsGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin, entity.RoleEmployee), gate)
					{
						settingsRead.GET("", hh.GetSettings)
					}
					settingsWrite := settingsGroup.Group("", middleware.RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin))
					{
						settingsWrite.PUT("", hh.UpdateSettings)
					}
				}
			}
		}
	}

	// Swagger UI: interactive API docs. Fail closed: served only in an explicit
	// development/test environment, so production never exposes the API map.
	if c.Config != nil && swaggerEnabled(c.Config.App.Environment) {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}

func swaggerEnabled(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "test":
		return true
	}
	return false
}
