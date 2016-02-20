package ci

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/travissimon/goobernet/data"
)

type BuildTask struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	IsGood bool   `json:"color"`
}

type TaskDetails struct {
	Name        string      `json:"name"`
	Url         string      `json:"url"`
	Description string      `json:"description"`
	LastBuild   Build       `json:"lastBuild"`
	Downstream  []BuildTask `json:"downstreamBuilds"`
}

type Build struct {
	Number    int64     `json:"buildNumber"`
	Duration  int64     `json:"duration"`
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
	Url       string    `json:"url"`
	IsGood    bool      `json:"isGood"`
}

type BuildServerProxy interface {
	GetTasks() ([]BuildTask, error)
	GetTaskDetails(taskName string) (*TaskDetails, error)
	CreateTask(newProject data.Project) error
}

var Proxy BuildServerProxy

func init() {
	cfg := data.GetConfig()

	var err error
	Proxy, err = newJenkinsProxy(cfg.JenkinsUrl, cfg.JenkinsUsername, cfg.JenkinsPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to Jenkins build server: %s\n", err.Error())
		fmt.Fprintf(os.Stderr, "Please check your connection and configuration settings\n")
		periodicallyRecheckConnection(cfg)
	}
}

// Proxies calls to Jenkins - allows system to run
// when Jenkins is unavailable
type JenkinsProxy struct {
	client *gojenkins.Jenkins
}

func newJenkinsProxy(url, username, password string) (BuildServerProxy, error) {
	fmt.Printf("Connecting to Jenkins instance: %s\n", url)
	proxy := &JenkinsProxy{}
	j, err := gojenkins.CreateJenkins(url, username, password).Init()
	if err != nil {
		return proxy, err
	}

	proxy.client = j
	return proxy, nil
}

func periodicallyRecheckConnection(cfg data.GoobernetConfig) {
	go func() {
		for {
			select {
			case <-time.After(1 * time.Minute):
				fmt.Fprintf(os.Stderr, "Retrying Jenkins connection\n")
				j, err := gojenkins.CreateJenkins(cfg.JenkinsUrl, cfg.JenkinsUsername, cfg.JenkinsPassword).Init()
				if err == nil {
					fmt.Printf("Connected to Jenkins. Small miracles.\n")
					jenkinsProxy, ok := Proxy.(*JenkinsProxy)
					if ok {
						jenkinsProxy.client = j
					}
					return
				}
			}
		}
	}()
}

func (jp *JenkinsProxy) GetTasks() ([]BuildTask, error) {
	if jp.client == nil {
		return nil, errors.New("No connection to build server\n")
	}

	jobNames, err := jp.client.GetAllJobNames()
	if err != nil {
		return nil, err
	}

	tasks := make([]BuildTask, 0, len(jobNames))

	for _, j := range jobNames {
		tasks = append(tasks, BuildTask{j.Name, j.Url, j.Color == "blue"})
	}

	return tasks, nil
}

func (jp *JenkinsProxy) GetTaskDetails(taskName string) (*TaskDetails, error) {
	if jp.client == nil {
		return nil, errors.New("No connection to build server\n")
	}

	job, err := jp.client.GetJob(taskName)
	if err != nil {
		return nil, err
	}

	downStr, _ := job.GetDownstreamJobs()
	tasks := make([]BuildTask, 0, len(downStr))
	for _, j := range downStr {
		tasks = append(tasks, BuildTask{j.GetName(), j.Raw.URL, j.Raw.Color == "blue"})
	}

	lb, _ := job.GetLastBuild()
	lastBuild := Build{
		Number:    lb.GetBuildNumber(),
		Duration:  lb.GetDuration(),
		Result:    lb.GetResult(),
		Timestamp: lb.GetTimestamp(),
		Url:       lb.GetUrl(),
		IsGood:    lb.IsGood(),
	}

	return &TaskDetails{
		Name:        job.GetName(),
		Url:         job.Raw.URL,
		Description: job.GetDescription(),
		LastBuild:   lastBuild,
		Downstream:  tasks,
	}, nil
}

func (jp *JenkinsProxy) CreateTask(newProject data.Project) error {
	if jp.client == nil {
		return errors.New("No connection to build server\n")
	}

	var templ data.JenkinsTemplate
	var err error

	templ = newProject.BuildTemplate

	var t *template.Template
	if t, err = template.New("Build template").Parse(templ.Content); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, newProject)
	if err != nil {
		return err
	}

	xml := buf.String()
	if _, err := jp.client.CreateJob(xml, newProject.ShortName); err != nil {
		return err
	}

	// save our proj
	if err := data.AddProject(newProject); err != nil {
		return err
	}

	return nil
}
