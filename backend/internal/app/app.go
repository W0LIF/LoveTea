package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/W0LIF/LoveTea/backend/internal/config"
	"github.com/W0LIF/LoveTea/backend/internal/modules/account"
	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/W0LIF/LoveTea/backend/internal/modules/chat"
	"github.com/W0LIF/LoveTea/backend/internal/modules/content"
	"github.com/W0LIF/LoveTea/backend/internal/modules/couples"
	"github.com/W0LIF/LoveTea/backend/internal/modules/dates"
	"github.com/W0LIF/LoveTea/backend/internal/modules/games"
	"github.com/W0LIF/LoveTea/backend/internal/modules/habits"
	"github.com/W0LIF/LoveTea/backend/internal/modules/health"
	"github.com/W0LIF/LoveTea/backend/internal/modules/home"
	"github.com/W0LIF/LoveTea/backend/internal/modules/letters"
	"github.com/W0LIF/LoveTea/backend/internal/modules/location"
	"github.com/W0LIF/LoveTea/backend/internal/modules/memories"
	"github.com/W0LIF/LoveTea/backend/internal/modules/notifications"
	"github.com/W0LIF/LoveTea/backend/internal/modules/onboarding"
	"github.com/W0LIF/LoveTea/backend/internal/modules/pet"
	"github.com/W0LIF/LoveTea/backend/internal/modules/planning"
	"github.com/W0LIF/LoveTea/backend/internal/modules/reminders"
	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
	"github.com/W0LIF/LoveTea/backend/internal/modules/users"
	"github.com/W0LIF/LoveTea/backend/internal/modules/wellness"
	"github.com/W0LIF/LoveTea/backend/internal/modules/wheel"
	"github.com/W0LIF/LoveTea/backend/internal/platform/mail"
	"github.com/W0LIF/LoveTea/backend/internal/platform/push"
	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/W0LIF/LoveTea/backend/internal/platform/storage"
)

type App struct {
	Server    *http.Server
	Outbox    *syncmodule.OutboxDispatcher
	Reminders *reminders.Worker
}

func New(
	cfg *config.Config,
	logger *slog.Logger,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	tokens *security.TokenManager,
) (*App, error) {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	timeout := middleware.Timeout(15 * time.Second)
	router.Use(func(next http.Handler) http.Handler {
		timed := timeout(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/sync/ws" {
				next.ServeHTTP(w, r)
				return
			}
			timed.ServeHTTP(w, r)
		})
	})

	// Push notification manager
	pushManager := push.NewManager(logger)
	if cfg.FCMAccessToken != "" && cfg.FCMAccessToken != "change_me" {
		pushManager.Register(push.NewFCMProvider(cfg.FCMAccessToken, cfg.FCMAccessToken, logger))
	}
	if cfg.APNSTopic != "" {
		pushManager.Register(push.NewAPNsProvider(cfg.APNSTopic, logger))
	}
	logger.Info("push manager initialized", "platforms", pushManager.Platforms())

	healthHandler := health.NewHandler(db, redisClient)

	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)

	// Auth module
	authRepository := auth.NewRepository(db)
	var mailSender *mail.Sender
	if cfg.MailHost != "" {
		mailSender = mail.NewSender(
			cfg.MailHost, cfg.MailPort,
			cfg.MailUser, cfg.MailPass,
			cfg.MailFrom, cfg.MailUseTLS,
		)
	}
	authService := auth.NewService(authRepository, tokens, mailSender)
	authHandler := auth.NewHandler(authService)

	rateLimitLogin := auth.RateLimit(redisClient, "login", 10, time.Minute)
	rateLimitRegister := auth.RateLimit(redisClient, "register", 10, time.Minute)
	rateLimitRefresh := auth.RateLimit(redisClient, "refresh", 30, time.Minute)

	// Users module
	usersRepository := users.NewRepository(db)
	usersService := users.NewService(usersRepository)
	usersHandler := users.NewHandler(usersService)

	// Couples module
	couplesRepo := couples.NewRepository(db)
	couplesService := couples.NewService(
		couplesRepo,
		cfg.JWTSecret,
		redisClient,
	)
	couplesHandler := couples.NewHandler(couplesService)

	homeRepository := home.NewRepository(db)
	homeService := home.NewService(usersService, couplesService, homeRepository)
	homeHandler := home.NewHandler(homeService, logger)

	onboardingRepository := onboarding.NewRepository(db)
	onboardingService := onboarding.NewService(onboardingRepository)
	onboardingHandler := onboarding.NewHandler(onboardingService, logger)

	syncRepository := syncmodule.NewRepository(db)
	syncService := syncmodule.NewService(syncRepository)
	syncHandler := syncmodule.NewHandler(syncService, logger)
	realtimeHub := syncmodule.NewHub()
	realtimeHandler := syncmodule.NewRealtimeHandler(realtimeHub, logger)
	outboxDispatcher := syncmodule.NewOutboxDispatcher(db, realtimeHub, logger)

	planningRepository := planning.NewRepository(db)
	planningService := planning.NewService(planningRepository)
	planningHandler := planning.NewHandler(planningService, logger)

	wellnessRepository := wellness.NewRepository(db)
	wellnessService := wellness.NewService(wellnessRepository)
	wellnessHandler := wellness.NewHandler(wellnessService, logger)

	habitsRepository := habits.NewRepository(db)
	habitsService := habits.NewService(habitsRepository)
	habitsHandler := habits.NewHandler(habitsService, logger)

	objectStore, err := storage.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("configure object storage: %w", err)
	}
	contentRepository := content.NewRepository(db)
	contentService := content.NewService(contentRepository, objectStore)
	contentHandler := content.NewHandler(contentService, logger)

	// Chat module
	chatRepository := chat.NewRepository(db)
	chatService := chat.NewService(chatRepository)
	chatHandler := chat.NewHandler(chatService, logger)

	// Letters module
	lettersRepository := letters.NewRepository(db)
	lettersService := letters.NewService(lettersRepository)
	lettersHandler := letters.NewHandler(lettersService, logger)

	// Notifications module
	notificationsRepository := notifications.NewRepository(db)
	notificationsService := notifications.NewService(notificationsRepository, pushManager, logger)
	notificationsHandler := notifications.NewHandler(notificationsService, logger)

	// Games module
	gamesRepository := games.NewRepository(db)
	gamesService := games.NewService(gamesRepository)
	gamesHandler := games.NewHandler(gamesService, logger)

	// Wheel module
	wheelRepository := wheel.NewRepository(db)
	wheelService := wheel.NewService(wheelRepository)
	wheelHandler := wheel.NewHandler(wheelService, logger)

	// Location module
	locationRepository := location.NewRepository(db)
	locationService := location.NewService(locationRepository)
	locationHandler := location.NewHandler(locationService, logger)

	// Dates module
	datesRepository := dates.NewRepository(db)
	datesService := dates.NewService(datesRepository, db)
	datesHandler := dates.NewHandler(datesService)

	// Memories module
	memoriesRepository := memories.NewRepository(db)
	memoriesService := memories.NewService(memoriesRepository)
	memoriesHandler := memories.NewHandler(memoriesService)

	// Account module (export + delete)
	accountRepository := account.NewRepository(db)
	accountService := account.NewService(accountRepository)
	accountHandler := account.NewHandler(accountService)

	// Pet module
	petRepository := pet.NewRepository(db)
	petService := pet.NewService(petRepository)
	petHandler := pet.NewHandler(petService, logger)

	// Reminders worker
	remindersRepository := reminders.NewRepository(db)
	reminderWorker := reminders.NewWorker(remindersRepository, logger, 60*time.Second, pushManager)

	router.Route("/api/v1", func(r chi.Router) {
		// Authentication
		r.Route("/auth", func(r chi.Router) {
			r.With(rateLimitRegister).Post("/register", authHandler.Register)
			r.With(rateLimitLogin).Post("/login", authHandler.Login)
			r.With(rateLimitRefresh).Post("/refresh", authHandler.Refresh)
			r.With(rateLimitLogin).Post("/forgot-password", authHandler.ForgotPassword)
			r.With(rateLimitLogin).Post("/reset-password", authHandler.ResetPassword)
			r.Group(func(r chi.Router) {
				r.Use(auth.AuthMiddleware(tokens))

				r.Post("/logout", authHandler.Logout)
				r.Get("/me", authHandler.Me)
			})
		})

		// User profiles
		r.Route("/users", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))

			r.Get("/me", usersHandler.GetMe)
			r.Patch("/me", usersHandler.UpdateMe)
			r.Post("/me/devices", usersHandler.RegisterDevice)
			r.Get("/me/devices", usersHandler.ListDevices)
			r.Delete("/me/devices/{deviceID}", usersHandler.DeleteDevice)
		})

		// Couples
		r.Route("/couples", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))

			r.Post("/{coupleID}/invitations", couplesHandler.CreateInvitation)
			r.Post("/join", couplesHandler.AcceptInvitation)

			r.Get("/me", couplesHandler.GetMyCouple)
			r.Patch("/me/profile", couplesHandler.UpdateMyProfile)
			r.Post("/me/detach/confirm", couplesHandler.ConfirmDetach)
		})
		r.With(auth.AuthMiddleware(tokens)).Post("/couples", couplesHandler.CreateCouple)

		r.Route("/onboarding", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))

			r.Get("/tests", onboardingHandler.ListTests)
			r.Get("/answers", onboardingHandler.ListMyResponses)
			r.Post("/answers", onboardingHandler.SubmitAnswers)
		})

		r.Route("/sync", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))

			r.Get("/pull", syncHandler.Pull)
			r.Post("/push", planningHandler.PushSyncChanges)
			r.Get("/ws", realtimeHandler.ServeHTTP)
		})

		r.With(auth.AuthMiddleware(tokens)).Get("/tasks", planningHandler.ListTasks)
		r.With(auth.AuthMiddleware(tokens)).Post("/tasks", planningHandler.CreateTask)
		r.With(auth.AuthMiddleware(tokens)).Patch("/tasks/{taskID}", planningHandler.UpdateTask)
		r.With(auth.AuthMiddleware(tokens)).Delete("/tasks/{taskID}", planningHandler.DeleteTask)
		r.With(auth.AuthMiddleware(tokens)).Get("/calendar/events", planningHandler.ListCalendarEvents)
		r.With(auth.AuthMiddleware(tokens)).Post("/calendar/events", planningHandler.CreateCalendarEvent)
		r.With(auth.AuthMiddleware(tokens)).Put("/calendar/events/{eventID}", planningHandler.UpdateCalendarEvent)
		r.With(auth.AuthMiddleware(tokens)).Delete("/calendar/events/{eventID}", planningHandler.DeleteCalendarEvent)
		r.With(auth.AuthMiddleware(tokens)).Get("/calendar/dates", planningHandler.ListImportantDates)
		r.With(auth.AuthMiddleware(tokens)).Post("/calendar/dates", planningHandler.CreateImportantDate)
		r.With(auth.AuthMiddleware(tokens)).Put("/calendar/dates/{dateID}", planningHandler.UpdateImportantDate)
		r.With(auth.AuthMiddleware(tokens)).Delete("/calendar/dates/{dateID}", planningHandler.DeleteImportantDate)

		r.With(auth.AuthMiddleware(tokens)).Get("/wellness/mood", wellnessHandler.ListMood)
		r.With(auth.AuthMiddleware(tokens)).Post("/wellness/mood", wellnessHandler.SaveMood)
		r.With(auth.AuthMiddleware(tokens)).Get("/wellness/sleep", wellnessHandler.ListSleep)
		r.With(auth.AuthMiddleware(tokens)).Post("/wellness/sleep", wellnessHandler.SaveSleep)

		r.With(auth.AuthMiddleware(tokens)).Get("/habits", habitsHandler.List)
		r.With(auth.AuthMiddleware(tokens)).Post("/habits", habitsHandler.Create)
		r.With(auth.AuthMiddleware(tokens)).Put("/habits/{habitID}", habitsHandler.Update)
		r.With(auth.AuthMiddleware(tokens)).Delete("/habits/{habitID}", habitsHandler.Delete)
		r.With(auth.AuthMiddleware(tokens)).Get("/habits/{habitID}/completions", habitsHandler.ListCompletions)
		r.With(auth.AuthMiddleware(tokens)).Post("/habits/{habitID}/completions", habitsHandler.Complete)
		r.With(auth.AuthMiddleware(tokens)).Delete("/habits/{habitID}/completions", habitsHandler.Uncomplete)
		r.With(auth.AuthMiddleware(tokens)).Get("/habits/{habitID}/stats", habitsHandler.Stats)

		r.With(auth.AuthMiddleware(tokens)).Get("/home", homeHandler.Get)

		r.With(auth.AuthMiddleware(tokens)).Get("/media", contentHandler.ListMedia)
		r.Route("/media", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))

			r.Post("/uploads", contentHandler.CreateUpload)
			r.Post("/{mediaID}/complete", contentHandler.CompleteUpload)
			r.Get("/{mediaID}/download", contentHandler.CreateDownloadURL)
			r.Delete("/{mediaID}", contentHandler.DeleteMedia)
		})
		r.With(auth.AuthMiddleware(tokens)).Get("/moments", contentHandler.ListMoments)
		r.With(auth.AuthMiddleware(tokens)).Post("/moments", contentHandler.CreateMoment)

		// Chat
		r.Route("/chat", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/conversation", chatHandler.GetConversation)
			r.Get("/messages", chatHandler.ListMessages)
			r.Post("/messages", chatHandler.SendMessage)
			r.Delete("/messages/{id}", chatHandler.DeleteMessage)
			r.Post("/messages/{id}/reaction", chatHandler.AddReaction)
			r.Delete("/messages/{id}/reaction", chatHandler.RemoveReaction)
			r.Get("/messages/{id}/reactions", chatHandler.ListReactions)
		})

		// Letters (Гневное письмо)
		r.Route("/letters", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/", lettersHandler.ListMyLetters)
			r.Get("/for-me", lettersHandler.ListForMe)
			r.Post("/", lettersHandler.CreateLetter)
			r.Get("/{id}", lettersHandler.GetLetter)
			r.Post("/{id}/read", lettersHandler.MarkRead)
			r.Post("/{id}/archive", lettersHandler.Archive)
			r.Get("/cooldown", lettersHandler.CooldownRemaining)
		})

		// Notifications
		r.Route("/notifications", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/", notificationsHandler.List)
			r.Get("/settings", notificationsHandler.GetSettings)
			r.Patch("/settings", notificationsHandler.UpdateSettings)
			r.Post("/{id}/read", notificationsHandler.MarkRead)
			r.Post("/read-all", notificationsHandler.MarkAllRead)
		})

		// Games
		r.Route("/games", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/sessions", gamesHandler.ListSessions)
			r.Post("/sessions", gamesHandler.StartSession)
			r.Post("/sessions/{id}/answer", gamesHandler.SubmitAnswer)
			r.Post("/sessions/{id}/finish", gamesHandler.FinishSession)
		})

		// Wheel
		r.Route("/wheel", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/sectors", wheelHandler.ListSectors)
			r.Post("/sectors", wheelHandler.CreateSector)
			r.Delete("/sectors/{id}", wheelHandler.DeleteSector)
			r.Post("/spin", wheelHandler.Spin)
			r.Post("/spin/{id}/confirm", wheelHandler.Confirm)
			r.Get("/history", wheelHandler.History)
		})

		// Location
		r.Route("/location", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/sharing", locationHandler.GetSharing)
			r.Post("/sharing", locationHandler.ToggleSharing)
			r.Post("/position", locationHandler.UpdatePosition)
			r.Get("/partner-position", locationHandler.GetPartnerPosition)
			r.Get("/geofences", locationHandler.ListGeofences)
			r.Post("/geofences", locationHandler.CreateGeofence)
		})

		// Dates
		r.Route("/dates", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Post("/ideas", datesHandler.CreateIdea)
			r.Get("/ideas", datesHandler.ListIdeas)
			r.Get("/ideas/{id}", datesHandler.GetIdea)
			r.Patch("/ideas/{id}/status", datesHandler.UpdateIdeaStatus)
			r.Get("/recommendations", datesHandler.ListRecommendations)
		})

		// Memories
		r.Route("/memories", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			memoriesHandler.RegisterRoutes(r)
		})

		// Pet
		r.Route("/pet", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			r.Get("/", petHandler.GetPetState)
			r.Patch("/appearance", petHandler.UpdatePetAppearance)
		})

		// Account export + delete
		r.Route("/account", func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokens))
			accountHandler.RegisterRoutes(r)
		})
	})

	logger.Info("HTTP routes registered")

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{Server: server, Outbox: outboxDispatcher, Reminders: reminderWorker}, nil
}
