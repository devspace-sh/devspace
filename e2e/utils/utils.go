package utils

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/portforward"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/port"

	"github.com/sirupsen/logrus"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	latestSpace "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	logger "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// BaseCustomFactory is a factory override for testing
type BaseCustomFactory struct {
	*factory.DefaultFactoryImpl

	CacheLogger logger.Logger
	Buff        *bytes.Buffer
	Verbose     bool
	Timeout     int
	Namespace   string
	Pwd         string
	Client      kubectl.Client
	DirPath     string
	DirName     string
}

func (b *BaseCustomFactory) GetLogContents() string {
	if b.Buff != nil {
		return b.Buff.String()
	}

	return ""
}

// ResetLog resets the log
func (b *BaseCustomFactory) ResetLog() {
	b.Buff = nil
	b.CacheLogger = nil
}

// GetLog implements interface
func (b *BaseCustomFactory) GetLog() logger.Logger {
	if b.Verbose {
		return logger.GetInstance()
	} else if b.CacheLogger == nil {
		b.Buff = &bytes.Buffer{}
		b.CacheLogger = logger.NewStreamLogger(b.Buff, logrus.InfoLevel)
	}

	return b.CacheLogger
}

// ChangeWorkingDir changes the working directory
func ChangeWorkingDir(pwd string, cachedLogger logger.Logger) error {
	wd, err := filepath.Abs(pwd)
	if err != nil {
		return err
	}
	// fmt.Println("WD:", wd)
	// Change working directory
	err = os.Chdir(wd)
	if err != nil {
		return err
	}

	// Notify user that we are not using the current working directory
	cachedLogger.Infof("Using devspace config in %s", filepath.ToSlash(wd))

	return nil
}

// PrintTestResult prints a test result with a specific formatting
func PrintTestResult(testName string, subTestName string, err error, log logger.Logger) {
	if err == nil {
		successIcon := html.UnescapeString("&#" + strconv.Itoa(128513) + ";")
		log.Donef("%v  Test '%v' of group test '%v' successfully passed!\n", successIcon, subTestName, testName)
	} else {
		failureIcon := html.UnescapeString("&#" + strconv.Itoa(128545) + ";")
		log.Warnf("%v  Test '%v' of group test '%v' failed!\n", failureIcon, subTestName, testName)
	}
}

// DeleteNamespace deletes a given namespace and waits for the process to finish
func DeleteNamespace(client kubectl.Client, namespace string) {
	err := client.KubeClient().CoreV1().Namespaces().Delete(namespace, nil)
	if err != nil {
		fmt.Println(err)
	}
}

// PurgeNamespacesByPrefixes deletes the namespaces that were created during testing process
func PurgeNamespacesByPrefixes(nsPrefixes []string) error {
	type customFactory struct {
		*factory.DefaultFactoryImpl
		ctrl build.Controller
	}

	f := &customFactory{}

	client, err := f.NewKubeDefaultClient()
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	nsList, err := client.KubeClient().CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range nsList.Items {
		name := ns.ObjectMeta.Name
		for _, p := range nsPrefixes {
			if strings.HasPrefix(name, p) {
				fmt.Println("Delete namespace:", name)
				DeleteNamespace(client, name)
			}
		}
	}

	return nil
}

// AnalyzePods waits for the pods to be running (if possible) and healthcheck them
func AnalyzePods(client kubectl.Client, namespace string, cachedLogger logger.Logger) error {
	err := analyze.NewAnalyzer(client, cachedLogger).Analyze(namespace, false)
	if err != nil {
		return err
	}

	return nil
}

func startPortForwarding(config *latest.Config, generatedConfig *generated.Config, client kubectl.Client, log logger.Logger) ([]*portforward.PortForwarder, error) {
	if config.Dev == nil {
		return nil, nil
	}

	var pf []*portforward.PortForwarder

	for _, portForwarding := range config.Dev.Ports {
		p, err := startForwarding(portForwarding, generatedConfig, config, client, log)
		if err != nil {
			return nil, err
		}

		pf = append(pf, p)
	}

	return pf, nil
}

func startForwarding(portForwarding *latest.PortForwardingConfig, generatedConfig *generated.Config, config *latest.Config, client kubectl.Client, log logger.Logger) (*portforward.PortForwarder, error) {
	var imageSelector []string
	if portForwarding.ImageName != "" && generatedConfig != nil {
		imageConfigCache := generatedConfig.GetActive().GetImageCache(portForwarding.ImageName)
		if imageConfigCache.ImageName != "" {
			imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
		}
	}

	selector, err := targetselector.NewTargetSelector(config, client, &targetselector.SelectorParameter{
		ConfigParameter: targetselector.ConfigParameter{
			Namespace:     portForwarding.Namespace,
			LabelSelector: portForwarding.LabelSelector,
		},
	}, false, imageSelector)
	if err != nil {
		return nil, errors.Errorf("Error creating target selector: %v", err)
	}

	log.StartWait("Port-Forwarding: Waiting for containers to start...")
	pod, err := selector.GetPod(log)
	log.StopWait()
	if err != nil {
		return nil, errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	} else if pod == nil {
		return nil, nil
	}

	ports := make([]string, len(portForwarding.PortMappings))
	addresses := make([]string, len(portForwarding.PortMappings))

	for index, value := range portForwarding.PortMappings {
		if value.LocalPort == nil {
			return nil, errors.Errorf("port is not defined in portmapping %d", index)
		}

		localPort := strconv.Itoa(*value.LocalPort)
		remotePort := localPort
		if value.RemotePort != nil {
			remotePort = strconv.Itoa(*value.RemotePort)
		}

		open, _ := port.Check(*value.LocalPort)
		if open == false {
			log.Warnf("Seems like port %d is already in use. Is another application using that port?", *value.LocalPort)
		}

		ports[index] = localPort + ":" + remotePort
		if value.BindAddress == "" {
			addresses[index] = "localhost"
		} else {
			addresses[index] = value.BindAddress
		}
	}

	readyChan := make(chan struct{})
	errorChan := make(chan error)

	pf, err := client.NewPortForwarder(pod, ports, addresses, make(chan struct{}), readyChan, errorChan)
	if err != nil {
		return nil, errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts()
		if err != nil {
			log.Fatalf("Error forwarding ports: %v", err)
		}
	}()

	// Wait till forwarding is ready
	select {
	case <-readyChan:
		log.Donef("Port forwarding started on %s", strings.Join(ports, ", "))
	case <-time.After(20 * time.Second):
		return nil, errors.Errorf("Timeout waiting for port forwarding to start")
	}

	return pf, nil
}

// PortForwardAndPing creates port-forwardings and ping them for a 200 status code
func PortForwardAndPing(config *latest.Config, generatedConfig *generated.Config, client kubectl.Client, cachedLogger logger.Logger) error {
	portForwarder, err := startPortForwarding(config, generatedConfig, client, cachedLogger)
	if err != nil {
		return err
	}

	for _, pf := range portForwarder {
		ports, err := pf.GetPorts()
		if err != nil {
			return err
		}

		for _, p := range ports {
			url := fmt.Sprintf("http://localhost:%v/", p.Local)

			fmt.Println("Pinging url:", url)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				cachedLogger.Donef("Pinging %v: status code 200", url)
			} else {
				return fmt.Errorf("pinging %v: status code %v", url, resp.StatusCode)
			}
		}
	}

	// We close all the port-forwardings
	defer func() {
		for _, v := range portForwarder {
			v.Close()
		}
	}()

	return nil
}

// Equal tells whether a and b contain the same elements.
func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

/* The MIT License (MIT)

Copyright (c) 2018 otiai10

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Copy copies src to dest, doesn't matter if src is a directory or a file
func Copy(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	return copy(src, dest, info)
}

// copy dispatches copy-funcs according to the mode.
// Because this "copy" could be called recursively,
// "info" MUST be given here, NOT nil.
func copy(src, dest string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return lcopy(src, dest, info)
	}
	if info.IsDir() {
		return dcopy(src, dest, info)
	}
	return fcopy(src, dest, info)
}

// fcopy is for just a file,
// with considering existence of parent directory
// and file permission.
func fcopy(src, dest string, info os.FileInfo) error {

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

// dcopy is for a directory,
// with scanning contents inside the directory
// and pass everything to "copy" recursively.
func dcopy(srcdir, destdir string, info os.FileInfo) error {

	if err := os.MkdirAll(destdir, info.Mode()); err != nil {
		return err
	}

	contents, err := ioutil.ReadDir(srcdir)
	if err != nil {
		return err
	}

	for _, content := range contents {
		cs, cd := filepath.Join(srcdir, content.Name()), filepath.Join(destdir, content.Name())
		if err := copy(cs, cd, content); err != nil {
			// If any error, exit immediately
			return err
		}
	}
	return nil
}

// lcopy is for a symlink,
// with just creating a new symlink by replicating src symlink.
func lcopy(src, dest string, info os.FileInfo) error {
	src, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(src, dest)
}

// =====================================================================

// CreateTempDir creates a temp directory in /tmp
func CreateTempDir() (dirPath string, dirName string, err error) {
	// Create temp dir in /tmp/
	dirPath, err = ioutil.TempDir("", "test-e2e")
	dirName = filepath.Base(dirPath)
	if err != nil {
		return
	}
	// fmt.Println("tempDir created:", dirPath)
	return
}

// DeleteTempDir deletes temp directory
func DeleteTempDir(dirPath string, log logger.Logger) {
	// TODO: Needs to be implemented later on (but bugs on windows)
	// Delete temp folder
	err := os.RemoveAll(dirPath)
	if err != nil {
		log.Fatalf("Error removing dir: %v", err)
	}
}

// Capture replaces os.Stdout with a writer that buffers any data written
// to os.Stdout. Call the returned function to cleanup and get the data
// as a string.
func Capture() func() (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	done := make(chan error, 1)

	save := os.Stdout
	os.Stdout = w

	var buf strings.Builder

	go func() {
		_, err := io.Copy(&buf, r)
		r.Close()
		done <- err
	}()

	return func() (string, error) {
		os.Stdout = save
		w.Close()
		err := <-done
		return buf.String(), err
	}
}

// StringInSlice checks if a string is in a slice
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// SpaceExists checks if a string is in a slice
func SpaceExists(str string, list []*latestSpace.Space) bool {
	for _, v := range list {
		if v.Name == str {
			return true
		}
	}
	return false
}

// DeleteTempAndResetWorkingDir deletes /tmp dir and reinitialize the working dir
func DeleteTempAndResetWorkingDir(tmpDir string, pwd string, log logger.Logger) {
	DeleteTempDir(tmpDir, log)
	_ = ChangeWorkingDir(pwd, log)
}

// LookForDeployment search for a specific deployment name among the deployments, returns true if found
func LookForDeployment(client kubectl.Client, namespace string, expectedDeployment ...string) (bool, error) {
	s, err := client.KubeClient().CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	var deployments []string

	for _, x := range s.Items {
		deployments = append(deployments, x.Name)
	}

	for _, d := range expectedDeployment {
		exists := StringInSlice(d, deployments)
		if !exists {
			return false, nil
		}
	}

	return true, nil
}

// GenerateNamespaceName generates a new Namespace name with the given prefix and a random suffix
func GenerateNamespaceName(prefix string) string {
	// Seed the random number generator using the current time (nanoseconds since epoch):
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(1000)

	return fmt.Sprintf("%s-%v", prefix, r)
}
