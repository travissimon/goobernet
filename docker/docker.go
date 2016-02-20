package docker

import (
	"fmt"
	"os"
	"strconv"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/travissimon/goobernet/data"
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
	Private uint64 `json:"private"`
	Public  uint64 `json:"publice"`
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
				Private: uint64(p.PrivatePort),
				Public:  uint64(p.PublicPort),
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
			Created:    c.Created,
			Image:      c.Image,
			Name:       c.Names[0][1:],
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

// remove me soon
func startExample() {
	ports := make([]docker.Port, 0, 1)
	ports = append(ports, docker.Port{Private: 8080, Public: 8080, Type: "tcp"})

	vars := make(map[string]string)
	vars["env-var"] = "env-val"

	registry := data.GetConfig().Registry

	containerName := "bpc-core-service"
	img := registry + "/" + containerName
	fmt.Printf("Creating from img: %s\n", img)
	ctr, err := docker.CreateContainer(containerName, img, "dev", ports, vars)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating container: %s\n", err.Error())
	}

	err = docker.StartContainer(ctr.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting container: %s\n", err.Error())
	}
}

func CreateContainer(containerName, image, domain string, portsToMap []Port, environmentVars map[string]string) (*docker.Container, error) {
	cfg := &docker.Config{}
	cfg.Image = image
	cfg.Hostname = containerName
	cfg.Domainname = domain

	var e struct{}
	exposedPorts := make(map[docker.Port]struct{})
	for _, p := range portsToMap {
		prt := strconv.FormatUint(p.Private, 10) + "/" + p.Type
		exposedPorts[docker.Port(prt)] = e
		fmt.Printf("Exposing %s\n", prt)
	}
	cfg.ExposedPorts = exposedPorts

	envStrs := make([]string, 0, 10)
	for k, v := range environmentVars {
		envStrs = append(envStrs, k+"="+v)
	}
	cfg.Env = envStrs

	hostCfg := &docker.HostConfig{}
	hostCfg.PublishAllPorts = false
	hostCfg.Privileged = false

	hostPorts := make(map[docker.Port][]docker.PortBinding)
	for _, p := range portsToMap {
		prt := strconv.FormatUint(p.Private, 10) + "/" + p.Type
		bindings := make([]docker.PortBinding, 0, 4)
		bindings = append(bindings, docker.PortBinding{HostIP: "", HostPort: strconv.FormatUint(p.Public, 10)})
		fmt.Printf("Binding %s to %s\n", prt, bindings[0])
		hostPorts[docker.Port(prt)] = bindings
	}
	hostCfg.PortBindings = hostPorts

	opts := docker.CreateContainerOptions{}
	opts.Config = cfg
	opts.Name = containerName
	opts.HostConfig = hostCfg

	json, _ := data.PrettyPrint(opts)
	fmt.Printf("create options: %s\n", string(json))

	container, err := client.CreateContainer(opts)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating container: %s\n", err.Error())
	} else {
		fmt.Printf("Container created: %s\n", container.ID)
	}

	return container, err
}

func StartContainer(id string) error {
	err := client.StartContainer(id, nil)
	return err
}
