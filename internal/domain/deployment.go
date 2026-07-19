package domain

import "errors"

type Deployment struct {
	ID         uint
	EnvID      uint
	UserID     uint
	CommitHash string
	Status     string
	Phase      string
	Log        string
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
