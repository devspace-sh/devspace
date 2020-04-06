package util

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/restart"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type ContainerRestarter interface {
	RestartContainer() error
}
