package custom

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"gotest.tools/assert"
)

const imageConfigName = "test"
const imageTag = "test123"

func TestShouldRebuild(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.NilError(t, err, "Create temporary dir")
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	assert.NilError(t, err, "Change dir")

	imageConf := &latest.ImageConfig{
		Image: "test-image",
		Build: &latest.BuildConfig{
			Custom: &latest.CustomConfig{},
		},
	}

	shouldRebuild, err := NewBuilder(imageConfigName, imageConf, imageTag).ShouldRebuild(nil, false)
	if shouldRebuild == false {
		t.Fatal("Expected rebuild true, got false")
	}

	// More complex test
	err = ioutil.WriteFile("test", []byte("test123"), 0644)
	assert.NilError(t, err, "Write File")

	imageConf.Build.Custom.OnChange = []*string{
		ptr.String("./**"),
	}

	cache := generated.NewCache()

	imageCache := cache.GetImageCache(imageConfigName)
	imageCache.Tag = imageTag

	shouldRebuild, err = NewBuilder(imageConfigName, imageConf, imageTag).ShouldRebuild(cache, false)
	assert.NilError(t, err, "ShouldRebuild")
	assert.Equal(t, shouldRebuild, true, "Unexpected shouldRebuild")
}

func TestBuild(t *testing.T) {
	imageConf := &latest.ImageConfig{
		Image: "test-image",
		Build: &latest.BuildConfig{
			Custom: &latest.CustomConfig{
				Command: "my-command",
				Args: []*string{
					ptr.String("flag1"),
					ptr.String("flag2"),
				},
				ImageFlag: "--imageflag",
			},
		},
	}

	builder := NewBuilder(imageConfigName, imageConf, imageTag)
	builder.cmd = &command.FakeCommand{}

	err := builder.Build(log.GetInstance())
	if err != nil {
		t.Fatal(err)
	}
}
