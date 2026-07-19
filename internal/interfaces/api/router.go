package api

import (
	"embed"
	"io/fs"
	"net/http"
	"pdeploy/internal/application"
	"pdeploy/internal/domain"
)

func NewRouter(
	projectSvc *application.ProjectService, 
	serverRepo domain.ServerRepository, 
	deploySvc *application.DeployService, 
	deployEngine *application.DeployEngine, 
	staticFS embed.FS,
) *http.ServeMux {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(projectSvc)
	serverHandler := NewServerHandler(serverRepo)
	deployHandler := NewDeployHandler(deploySvc, deployEngine, projectSvc)

	// Project Routes
	mux.HandleFunc("GET /api/projects", projectHandler.List)
	mux.HandleFunc("POST /api/projects", projectHandler.Create)

	// Environment Routes
	mux.HandleFunc("POST /api/projects/{id}/environments", projectHandler.AddEnvironment)
	mux.HandleFunc("PUT /api/projects/{id}/environments/{name}", projectHandler.UpdateEnvironment)

	// Server Routes
	mux.HandleFunc("GET /api/servers", serverHandler.List)
	mux.HandleFunc("POST /api/servers", serverHandler.Create)

	// Deploy & Log Stream Route
	mux.HandleFunc("POST /api/deployments", deployHandler.StartDeploy)
	mux.HandleFunc("GET /api/deployments/{id}/logs", deployHandler.StreamLogs)

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

	return mux
}
