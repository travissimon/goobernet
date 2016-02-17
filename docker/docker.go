package docker

import (
	"fmt"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

func init() {
	var err error
	client, err = docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to docker: %s\n", err.Error())
	}
}

var client *docker.Client

func GetContainers() ([]docker.APIContainers, error) {
	listOpts := docker.ListContainersOptions{}
	listOpts.All = true
	listOpts.Limit = 1000
	listOpts.Size = true
	return client.ListContainers(listOpts)
}
