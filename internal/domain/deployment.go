package domain

import "errors"

type Deployment struct {
	ID         uint   `json:"id"`
	EnvID      uint   `json:"env_id"`
	UserID     uint   `json:"user_id"`
	CommitHash string `json:"commit_hash"`
	Status     string `json:"status"` // pending, running, success, failed
	Phase      string `json:"phase"`
	Log        string `json:"log"`
}

func NewDeployment(envID, userID uint, commitHash string) (*Deployment, error) {
	if envID <= 0 {
		return nil, errors.New("environment ID must be > 0")
	}
	if userID <= 0 {
		return nil, errors.New("user ID must be > 0")
	}

	return &Deployment{
		EnvID:      envID,
		UserID:     userID,
		CommitHash: commitHash,
		Status:     "pending",
		Phase:      "init",
	}, nil
}

func (d *Deployment) SetPhase(phase string) {
	d.Phase = phase
}

func (d *Deployment) MarkSuccess(log string) {
	d.Status = "success"
	d.Log = log
}

func (d *Deployment) MarkFailed(log string) {
	d.Status = "failed"
	d.Log = log
}
