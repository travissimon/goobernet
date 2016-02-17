package data

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type GoobernetConfig struct {
	JenkinsUrl string `json:jenkinsUrl`
}

type Project struct {
	Id            uint   `json:"id"`
	Name          string `json:"name"`
	ShortName     string `json:"shortName"`
	Description   string `json:"description"`
	Email         string `json:"email"`
	ContactName   string `json:"contactName"`
	GithubUrl     string `json:"githubUrl"`
	BuildTemplate string `json:"buildTemplate"`
}

type Environment struct {
	Id           uint   `json:"id"`
	Name         string `json:"name"`
	GoobenetUrl  string `json:"goobernetUrl"`
	StartingPort uint   `json:"startingPort"`
}

type Deployment struct {
	Project     Project
	Environment Environment
	Location    string // url:port
}

// Used to serialise and deserialise deployments
type DeploymentJoin struct {
	EnvironmentId uint   `json:"environmentId"`
	ProjectId     uint   `json:"projectId"`
	Location      string `json:"location"`
}

type JenkinsTemplate struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

var config GoobernetConfig
var projects []Project
var environments []Environment
var deployments []Deployment
var templates []JenkinsTemplate

func init() {
	_, err := os.Stat(".goobernet")
	if err != nil {
		fmt.Printf("Creating new config")
		createConfigDirectory()
	} else {
		fmt.Printf("Loading config data\n")
		readConfig()
	}
	loadTemplates()
}

func GetConfig() GoobernetConfig {
	return config
}

func GetProjects() []Project {
	return projects
}

func GetEnvironments() []Environment {
	return environments
}

func GetEnvironmentById(id uint) (Environment, error) {
	for i := 0; i < len(environments); i++ {
		if environments[i].Id == id {
			return environments[i], nil
		}
	}
	return Environment{}, fmt.Errorf("Unable to find environment with id: %d", id)
}

func GetEnvironmentByName(name string) (Environment, error) {
	name = strings.ToLower(name)
	for i := 0; i < len(environments); i++ {
		if strings.ToLower(environments[i].Name) == name {
			return environments[i], nil
		}
	}
	return Environment{}, fmt.Errorf("Unable to find environment named '%s'", name)
}

func GetDeployments() []Deployment {
	return deployments
}

func GetDeploymentsByEnvironmentName(environmentName string) (map[string]string, error) {
	environment, err := GetEnvironmentByName(environmentName)
	if err != nil {
		return make(map[string]string), fmt.Errorf("Could not find environment with the name '%s'", environmentName)
	}
	return GetDeploymentsByEnvironmentId(environment.Id)
}

func GetDeploymentsByEnvironmentId(id uint) (map[string]string, error) {
	urlMap := make(map[string]string)

	for i := 0; i < len(deployments); i++ {
		d := deployments[i]
		if d.Environment.Id == id {
			urlMap[d.Project.ShortName] = d.Location
		}
	}

	return urlMap, nil
}

func GetTemplates() []JenkinsTemplate {
	return templates
}

// Serialisation methods

func createConfigDirectory() {
	// create directory
	os.Mkdir(".goobernet", 0755)

	// general config
	c := GoobernetConfig{"https://docker-server.dev.etd.nicta.com.au"}
	if err := serialise(config, "config.json"); err != nil {
		return
	}
	config = c

	// projects
	projects = make([]Project, 0, 5)
	if err := serialise(projects, "projects.json"); err != nil {
		return
	}

	// environments
	environments = make([]Environment, 0, 5)
	if err := serialise(environments, "environments.json"); err != nil {
		return
	}

	// deployments
	deployments = make([]Deployment, 0, 5)
	if err := serialise(deployments, "deployments.json"); err != nil {
		return
	}
}

func serialise(obj interface{}, filename string) error {
	bytes, err := prettyPrint(obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serialising object: %s\n", err.Error())
		return err
	}

	err = ioutil.WriteFile(".goobernet/"+filename, bytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err.Error())
		return err
	}

	return nil
}

func readConfig() {
	bytes, err := ioutil.ReadFile(".goobernet/config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %s\n", err.Error())
	} else {
		err = json.Unmarshal(bytes, &config)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config json: %s\n", err.Error())
	}

	bytes, err = ioutil.ReadFile(".goobernet/projects.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading project data: %s\n", err.Error())
	} else {
		err = json.Unmarshal(bytes, &projects)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling project json: %s\n", err.Error())
	}

	bytes, err = ioutil.ReadFile(".goobernet/environments.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading environment data: %s\n", err.Error())
	} else {
		err = json.Unmarshal(bytes, &environments)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling environment json: %s\n", err.Error())
	}

	// deployment joins refer to ids
	// but in memory we'll store actual objects
	var djs []DeploymentJoin
	bytes, err = ioutil.ReadFile(".goobernet/deployments.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading deployment data: %s\n", err.Error())
	} else {
		err = json.Unmarshal(bytes, &djs)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling deployment json: %s\n", err.Error())
	}

	var depls = make([]Deployment, 0, 10)
	for i := 0; i < len(djs); i++ {
		join := djs[i]

		// assume few projects/deployments, so looping is cheap
		var proj Project
		for j := 0; j < len(projects); j++ {
			if projects[j].Id == join.ProjectId {
				proj = projects[j]
				break
			}
		}
		if proj.Id == 0 {
			fmt.Fprintf(os.Stderr, "Project id %d not found during initialisation\n", join.ProjectId)
			continue
		}
		var env Environment
		for j := 0; j < len(environments); j++ {
			if environments[j].Id == join.EnvironmentId {
				env = environments[j]
				break
			}
		}
		if env.Id == 0 {
			fmt.Fprintf(os.Stderr, "Environment id %d not found during initialisation\n", join.EnvironmentId)
			continue
		}
		depls = append(depls, Deployment{proj, env, join.Location})
	}

	deployments = depls
}

func deserialise(obj interface{}, filename string) (interface{}, error) {
	bytes, err := ioutil.ReadFile(".goobernet/" + filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %s\n", filename, err.Error())
		return nil, err
	}
	err = json.Unmarshal(bytes, &obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling json: %s\n", err.Error())
		return nil, err
	}
	return obj, nil
}

func loadTemplates() {
	files, err := ioutil.ReadDir("templates")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %s\n", err.Error())
		templates = make([]JenkinsTemplate, 0, 10)
		return
	}
	for _, f := range files {
		t := JenkinsTemplate{}
		t.Name = f.Name()
		bytes, err := ioutil.ReadFile("templates/" + t.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading Jenkins template: %s\n", err.Error())
			continue
		}
		t.Content = string(bytes)
		templates = append(templates, t)
	}
}

func prettyPrint(obj interface{}) ([]byte, error) {
	return json.MarshalIndent(obj, "", "\t")
}
