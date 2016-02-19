package docker

import (
	"fmt"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

type Container struct {
	Command    string `json:"command"`
	Created    uint   `json:"created"`
	Id         string `json:"id"`
	Image      string `json:"image"`
	Name       string `json:"name"` // note this is an array from docker, might have to change
	RootFsSize uint   `json:"rootFsSize"`
	RwSize     uint   `json:"sizeRw"`
	Status     string `json:"status"`
}

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
