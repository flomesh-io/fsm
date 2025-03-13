package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

var (
	refreshFrequency = time.Second * 2
)

type watcher struct {
	fsmApp            string
	podName           string
	spinner           *spinner.Spinner
	containerCnt      int
	readyContainerCnt int
	restartCnt        int
	ready             bool
}

func (w *watcher) refresh(clientSet kubernetes.Interface, fsmNamespace string) {
	if w.ready {
		return
	}
	fieldSelector := fields.OneTermEqualSelector("metadata.name", w.podName).String()
	pods, err := clientSet.CoreV1().Pods(fsmNamespace).List(context.Background(), metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		return
	}
	if len(pods.Items) == 0 {
		w.ready = true
		w.spinner.Stop()
		return
	}
	pod := pods.Items[0]
	phase := pod.Status.Phase
	containers := pod.Status.ContainerStatuses
	w.containerCnt = len(containers)
	w.readyContainerCnt = 0
	w.restartCnt = 0
	for _, c := range containers {
		if c.Ready {
			w.readyContainerCnt++
		}
		w.restartCnt = w.restartCnt + int(c.RestartCount)
	}
	if w.containerCnt == w.readyContainerCnt || corev1.PodSucceeded == phase {
		w.ready = true
		w.spinner.Stop()
	} else {
		w.spinner.Suffix = fmt.Sprintf("%s[%s] READY:%d/%d STATUS:%s RESTARTS:%d",
			w.podName, fsmNamespace, w.readyContainerCnt, w.containerCnt, phase, w.restartCnt)
	}
}

// Spinner indicator to fsm install
type Spinner struct {
	fsmNamespace string
	clientSet    kubernetes.Interface
	watchers     map[string]*watcher
	err          error
	quit         chan bool

	deployPrometheus bool
	deployGrafana    bool
	deployJaeger     bool
}

// Init instance of Spinner with the supplied options
func (s *Spinner) Init(clientSet kubernetes.Interface, fsmNamespace string, vals map[string]interface{}) {
	s.clientSet = clientSet
	s.fsmNamespace = fsmNamespace
	s.quit = make(chan bool, 1)
	s.watchers = make(map[string]*watcher)
	if fsm, exists := vals["fsm"]; exists {
		fsmVals := fsm.(map[string]interface{})
		if v, has := fsmVals["deployPrometheus"]; has {
			s.deployPrometheus = v.(bool)
		}
		if v, has := fsmVals["deployGrafana"]; has {
			s.deployGrafana = v.(bool)
		}
		if v, has := fsmVals["deployJaeger"]; has {
			s.deployJaeger = v.(bool)
		}
	}
}

func (s *Spinner) done() bool {
	if len(s.watchers) >= 3 {
		doneApps := map[string]bool{
			"fsm-bootstrap":  false,
			"fsm-injector":   false,
			"fsm-controller": false,
		}
		if s.deployPrometheus {
			doneApps["fsm-prometheus"] = false
		}
		if s.deployGrafana {
			doneApps["fsm-grafana"] = false
		}
		if s.deployJaeger {
			doneApps["fsm-jaeger"] = false
		}
		for _, w := range s.watchers {
			if !w.ready {
				return false
			}
			doneApps[w.fsmApp] = true
		}
		for _, done := range doneApps {
			if !done {
				return false
			}
		}
		return true
	}
	return false
}

func (s *Spinner) refreshFsmApps() {
	apps, err := s.clientSet.CoreV1().Pods(s.fsmNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		s.err = err
		s.quit <- true
		return
	}
	if len(apps.Items) == 0 {
		return
	}
	for _, app := range apps.Items {
		_, exists := s.watchers[app.Name]
		if !exists {
			w := new(watcher)
			w.podName = app.Name
			w.fsmApp = app.Labels["app"]
			if len(w.fsmApp) == 0 {
				parts := strings.Split(w.podName, "-")
				if len(parts) > 1 {
					w.fsmApp = fmt.Sprintf("%s-%s", parts[0], parts[1])
				}
			}
			w.spinner = spinner.New(spinner.CharSets[35], time.Millisecond*500)
			_ = w.spinner.Color("bgBlue", "bold", "fgGreen")

			if len(w.fsmApp) > 0 {
				w.spinner.Suffix = fmt.Sprintf("%s[%s] is being deployed ...", w.fsmApp, w.podName)
				w.spinner.FinalMSG = fmt.Sprintf("%s[%s] Done\n", w.fsmApp, w.podName)
			} else {
				w.spinner.Suffix = w.podName
				w.spinner.FinalMSG = fmt.Sprintf("%s Done\n", w.podName)
			}
			w.spinner.Start()
			s.watchers[app.Name] = w
		}
	}
	for _, w := range s.watchers {
		w.refresh(s.clientSet, s.fsmNamespace)
	}
}

// Run starts spinner indicator
func (s *Spinner) Run(job func() error) error {
	updateChan := make(chan interface{}, 1)

	slidingTimer := time.NewTimer(refreshFrequency)
	defer slidingTimer.Stop()

	go func() {
		if err := job(); err != nil {
			s.err = err
			s.quit <- true
		}
	}()

	for {
		select {
		case <-s.quit:
			return s.err
		case <-updateChan:
			slidingTimer.Reset(refreshFrequency)
		case <-slidingTimer.C:
			s.refreshFsmApps()
			if !s.done() {
				updateChan <- new(interface{})
			} else {
				s.quit <- true
			}
		}
	}
}
