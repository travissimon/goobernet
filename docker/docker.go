package docker

import (
	"fmt"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

type Container struct {
	Command    string  `json:"command"`
	Created    int64   `json:"created"`
	Id         string  `json:"id"`
	Image      string  `json:"image"`
	Name       string  `json:"name"` // note this is an array from docker, might have to change
	Ports      []Port  `json:"ports"`
	Labels     []Label `json:"labels"`
	RootFsSize int64   `json:"rootFsSize"`
	RwSize     int64   `json:"sizeRw"`
	Status     string  `json:"status"`
}

type Port struct {
	Private int64  `json:"private"`
	Public  int64  `json:"publice"`
	Type    string `json:"type"`
	IP      string `json:"IP"`
}

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func init() {
	var err error
	client, err = docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to docker: %s\n", err.Error())
	}
}

var client *docker.Client

func GetContainers() ([]Container, error) {
	listOpts := docker.ListContainersOptions{}
	listOpts.All = true
	listOpts.Limit = 1000
	listOpts.Size = true

	apiContainers, err := client.ListContainers(listOpts)

	if err != nil {
		return nil, err
	}

	containers := make([]Container, 0, len(apiContainers))
	for _, c := range apiContainers {
		ports := make([]Port, 0, 5)
		for _, p := range c.Ports {
			port := Port{
				Private: p.PrivatePort,
				Public:  p.PublicPort,
				Type:    p.Type,
				IP:      p.IP,
			}
			ports = append(ports, port)
		}

		labels := make([]Label, 0, 5)
		for k, v := range c.Labels {
			labels = append(labels, Label{k, v})
		}

		c := Container{
			Command:    c.Command,
			Id:         c.ID,
			Image:      c.Image,
			Name:       c.Names[0],
			Ports:      ports,
			Labels:     labels,
			RootFsSize: c.SizeRootFs,
			RwSize:     c.SizeRw,
			Status:     c.Status,
		}

		containers = append(containers, c)
	}

	return containers, nil
}

func CreateContainer() {
	// docker.Container, err := client.CreateContainer(opts)
}
