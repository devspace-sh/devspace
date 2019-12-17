package utils

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/portforward"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/port"

	"html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	logger "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// ChangeWorkingDir changes the working directory
func ChangeWorkingDir(pwd string) error {
	log := logger.GetInstance()

	wd, err := filepath.Abs(pwd)
	if err != nil {
		return err
	}
	fmt.Println("WD:", wd)
	// Change working directory
	err = os.Chdir(wd)
	if err != nil {
		return err
	}

	// Notify user that we are not using the current working directory
	log.Infof("Using devspace config in %s", filepath.ToSlash(wd))

	return nil
}

// PrintTestResult prints a test result with a specific formatting
func PrintTestResult(testName string, subTestName string, err error) {
	if err == nil {
		successIcon := html.UnescapeString("&#" + strconv.Itoa(128513) + ";")
		fmt.Printf("%v  Test '%v' of group test '%v' successfully passed!\n", successIcon, subTestName, testName)
	} else {
		failureIcon := html.UnescapeString("&#" + strconv.Itoa(128545) + ";")
		fmt.Printf("%v  Test '%v' of group test '%v' failed!\n", failureIcon, subTestName, testName)
	}
}

// DeleteNamespaceAndWait deletes a given namespace and waits for the process to finish
func DeleteNamespaceAndWait(client kubectl.Client, namespace string) {
	log := logger.GetInstance()

	log.StartWait("Deleting namespace '" + namespace + "'")
	err := client.KubeClient().CoreV1().Namespaces().Delete(namespace, nil)
	if err != nil {
		log.Fatal(err)
	}

	isExists := true
	for isExists {
		_, err = client.KubeClient().CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			isExists = false
		}
	}

	defer log.StopWait()
}

// AnalyzePods waits for the pods to be running (if possible) and healthcheck them
func AnalyzePods(client kubectl.Client, namespace string) error {
	err := analyze.NewAnalyzer(client, logger.GetInstance()).Analyze(namespace, false)
	if err != nil {
		return err
	}

	return nil
}

func startPortForwarding(config *latest.Config, generatedConfig *generated.Config, client kubectl.Client) ([]*portforward.PortForwarder, error) {
	if config.Dev == nil {
		return nil, nil
	}

	var pf []*portforward.PortForwarder

	for _, portForwarding := range config.Dev.Ports {
		p, err := startForwarding(portForwarding, generatedConfig, config, client)
		if err != nil {
			return nil, err
		}

		pf = append(pf, p)
	}

	return pf, nil
}

func startForwarding(portForwarding *latest.PortForwardingConfig, generatedConfig *generated.Config, config *latest.Config, client kubectl.Client) (*portforward.PortForwarder, error) {
	log := logger.GetInstance()

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
func PortForwardAndPing(config *latest.Config, generatedConfig *generated.Config, client kubectl.Client) error {
	log := logger.GetInstance()

	portForwarder, err := startPortForwarding(config, generatedConfig, client)
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
			resp, err := http.Get(url)
			if err != nil {
				log.Fatal(err)
			}

			if resp.StatusCode == 200 {
				log.Donef("Pinging %v: status code 200", url)
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
	dirPath, err = ioutil.TempDir("", "init")
	dirName = filepath.Base(dirPath)
	if err != nil {
		return
	}
	fmt.Println("tempDir created:", dirPath)
	return
}

// DeleteTempDir deletes temp directory
func DeleteTempDir(dirPath string) {
	// log := logger.GetInstance()

	// //Delete temp folder
	// err := os.RemoveAll(dirPath)
	// if err != nil {
	// 	log.Fatalf("Error removing dir: %v", err)
	// }
	fmt.Println("Fake deleting temp")
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

// DeleteTempAndResetWorkingDir deletes /tmp dir and reinitialize the working dir
func DeleteTempAndResetWorkingDir(tmpDir string, pwd string) {
	DeleteTempDir(tmpDir)
	_ = ChangeWorkingDir(pwd)
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
