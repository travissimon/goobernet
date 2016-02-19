package model

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/travissimon/goobernet/data"
	"github.com/travissimon/goobernet/jenkins"
)

type Easement struct {
	Project     data.Project              `json:"project"`
	Build       jenkins.JenkinsJobDetails `json:"build"`
	Deployments []EaseDeployment          `json:"deployments"`
}

type EaseDeployment struct {
	Environment data.Environment `json:"environment"`
	Container   docker.Container `json:"container"`
}
