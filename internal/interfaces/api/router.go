package api

import (
	"embed"
	"io/fs"
	"net/http"
	"github.com/riverisagame/godeploy/internal/application"
	"strings"
)

func NewRouter(
	projectSvc *application.ProjectService, 
	serverSvc *application.ServerService, 
	deploySvc *application.DeployService, 
	deployEngine *application.DeployEngine, 
	authSvc *application.AuthService,
	staticFS embed.FS,
	jwtSecret string,
) http.Handler {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(projectSvc)
	serverHandler := NewServerHandler(serverSvc)
	deployHandler := NewDeployHandler(deploySvc, deployEngine, projectSvc)
	authHandler := NewAuthHandler(authSvc)
	authMiddleware := NewAuthMiddleware(jwtSecret)

	// Public Routes
	mux.HandleFunc("POST /api/login", authHandler.Login)

	// Helper to wrap handlers
	protect := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Wrap(h).ServeHTTP
	}

	adminProtect := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Wrap(RequireAdmin(h)).ServeHTTP
	}

	// Project Routes
	mux.HandleFunc("GET /api/projects", protect(projectHandler.List))
	mux.HandleFunc("POST /api/projects", adminProtect(projectHandler.Create))
	mux.HandleFunc("PUT /api/projects/{id}", adminProtect(projectHandler.Update))
	mux.HandleFunc("DELETE /api/projects/{id}", adminProtect(projectHandler.Delete))

	// Environment Routes
	mux.HandleFunc("POST /api/projects/{id}/environments", adminProtect(projectHandler.AddEnvironment))
	mux.HandleFunc("PUT /api/projects/{id}/environments/{name}", adminProtect(projectHandler.UpdateEnvironment))
	mux.HandleFunc("GET /api/projects/{id}/environments/{name}/diff", protect(deployHandler.GetDiff))

	// Server Routes
	mux.HandleFunc("GET /api/servers", protect(serverHandler.List))
	mux.HandleFunc("POST /api/servers", adminProtect(serverHandler.Create))
	mux.HandleFunc("PUT /api/servers/{id}", adminProtect(serverHandler.Update))
	mux.HandleFunc("DELETE /api/servers/{id}", adminProtect(serverHandler.Delete))

	webhookHandler := NewWebhookHandler(projectSvc, deploySvc, deployEngine)

	// Deploy & Log Stream Route
	mux.HandleFunc("GET /api/deployments", protect(deployHandler.ListDeployments))
	mux.HandleFunc("POST /api/deployments", protect(deployHandler.StartDeploy))
	mux.HandleFunc("POST /api/deployments/{id}/rollback", adminProtect(deployHandler.Rollback)) // 仅管理员可回滚
	mux.HandleFunc("POST /api/deployments/{id}/cancel", protect(deployHandler.Cancel))
	mux.HandleFunc("GET /api/deployments/{id}/logs", protect(deployHandler.StreamLogs))

	// Webhooks
	mux.HandleFunc("POST /api/webhook/github", webhookHandler.HandleGitHubPush)

	// Setup embedded static files
	distFS, err := fs.Sub(staticFS, "web/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && !strings.HasPrefix(r.URL.Path, "/api/") {
			// Check if file exists in embedded FS
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			_, err := distFS.Open(path)
			if err != nil {
				// Fallback to index.html for SPA routing
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		} else {
			// If not GET or not static, maybe a 404 for API
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	recoveryMiddleware := NewRecoveryMiddleware()
	loggingMiddleware := NewLoggingMiddleware()
	
	var handler http.Handler = mux
	handler = CORS(handler)
	handler = loggingMiddleware.Wrap(handler)
	handler = recoveryMiddleware.Wrap(handler)
	return handler
}
