package api

import (
	"embed"
	"io/fs"
	"net/http"
	"pdeploy/internal/application"
)

func NewRouter(
	projectSvc *application.ProjectService, 
	serverSvc *application.ServerService, 
	deploySvc *application.DeployService, 
	deployEngine *application.DeployEngine, 
	authSvc *application.AuthService,
	staticFS embed.FS,
) http.Handler {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(projectSvc)
	serverHandler := NewServerHandler(serverSvc)
	deployHandler := NewDeployHandler(deploySvc, deployEngine, projectSvc)
	authHandler := NewAuthHandler(authSvc)
	authMiddleware := NewAuthMiddleware("secret-key")

	// Public Routes
	mux.HandleFunc("POST /api/login", authHandler.Login)

	// Helper to wrap handlers
	protect := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Wrap(h).ServeHTTP
	}

	// Project Routes
	mux.HandleFunc("GET /api/projects", protect(projectHandler.List))
	mux.HandleFunc("POST /api/projects", protect(projectHandler.Create))

	// Environment Routes
	mux.HandleFunc("POST /api/projects/{id}/environments", protect(projectHandler.AddEnvironment))
	mux.HandleFunc("PUT /api/projects/{id}/environments/{name}", protect(projectHandler.UpdateEnvironment))
	mux.HandleFunc("GET /api/projects/{id}/environments/{name}/diff", protect(deployHandler.GetDiff))

	// Server Routes
	mux.HandleFunc("GET /api/servers", protect(serverHandler.List))
	mux.HandleFunc("POST /api/servers", protect(serverHandler.Create))
	mux.HandleFunc("DELETE /api/servers/{id}", protect(serverHandler.Delete))

	// Deploy & Log Stream Route
	mux.HandleFunc("GET /api/deployments", protect(deployHandler.ListDeployments))
	mux.HandleFunc("POST /api/deployments", protect(deployHandler.StartDeploy))
	mux.HandleFunc("POST /api/deployments/{id}/rollback", protect(deployHandler.Rollback))
	mux.HandleFunc("GET /api/deployments/{id}/logs", protect(deployHandler.StreamLogs))

	// Setup embedded static files
	distFS, err := fs.Sub(staticFS, "web/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve from root if the request is not an API call
		fileServer.ServeHTTP(w, r)
	})

	recoveryMiddleware := NewRecoveryMiddleware()
	return recoveryMiddleware.Wrap(mux)
}
