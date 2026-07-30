package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/riverisagame/godeploy/internal/application"
)

type WebhookHandler struct {
	projectSvc   *application.ProjectService
	deploySvc    *application.DeployService
	deployEngine *application.DeployEngine
}

func NewWebhookHandler(projectSvc *application.ProjectService, deploySvc *application.DeployService, deployEngine *application.DeployEngine) *WebhookHandler {
	return &WebhookHandler{
		projectSvc:   projectSvc,
		deploySvc:    deploySvc,
		deployEngine: deployEngine,
	}
}

// @Ref: docs/sps/plans/20260721_v2.5_refactoring_ir.md Task 3.3 | @Date: 2026-07-22
func (h *WebhookHandler) HandleGitHubPush(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.projectSvc.GetProjectByID(uint(projectID))
	if err != nil || project == nil {
		w.WriteHeader(http.StatusNotFound)
		RespondJSON(w, map[string]string{"error": "project not found"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to read body")
		return
	}
	// @Ref: docs/sps/plans/20260730_bugfix_plan.md | @Date: 2026-07-30
	defer func() { _ = r.Body.Close() }()

	githubSig := r.Header.Get("X-Hub-Signature-256")
	gitlabToken := r.Header.Get("X-Gitlab-Token")
	giteeToken := r.Header.Get("X-Gitee-Token")

	if project.WebhookSecret == "" {
		RespondError(w, http.StatusForbidden, "missing secret")
		return
	}

	valid := false
	if githubSig != "" {
		mac := hmac.New(sha256.New, []byte(project.WebhookSecret))
		mac.Write(body)
		expectedMAC := mac.Sum(nil)
		expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
		if subtle.ConstantTimeCompare([]byte(githubSig), []byte(expectedSignature)) == 1 {
			valid = true
		}
	} else if gitlabToken != "" {
		if subtle.ConstantTimeCompare([]byte(gitlabToken), []byte(project.WebhookSecret)) == 1 {
			valid = true
		}
	} else if giteeToken != "" {
		if subtle.ConstantTimeCompare([]byte(giteeToken), []byte(project.WebhookSecret)) == 1 {
			valid = true
		}
	}

	if !valid {
		RespondError(w, http.StatusForbidden, "invalid signature or token")
		return
	}

	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	started := 0
	for _, env := range project.Environments {
		if env.Branch == branch || (env.Branch == "" && branch == "main") { // default to main if branch not set
			// Start deploy in background to not block the webhook response
			deployment, err := h.deploySvc.TriggerDeploy(env.ID, 0, "WEBHOOK_TRIGGER")
			if err == nil {
				go h.deployEngine.StartDeploy(deployment, project, env)
				started++
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	RespondJSON(w, map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("triggered %d environments", started),
	})
}
