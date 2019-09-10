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
)

const imageConfigName = "test"
const imageTag = "test123"

func TestShouldRebuild(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	if err != nil {
		log.Fatal(err)
	}

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
	if err != nil {
		log.Fatal(err)
	}

	imageConf.Build.Custom.OnChange = []*string{
		ptr.String("./**"),
	}

	cache := generated.NewCache()

	imageCache := cache.GetImageCache(imageConfigName)
	imageCache.Tag = imageTag

	shouldRebuild, err = NewBuilder(imageConfigName, imageConf, imageTag).ShouldRebuild(cache, false)
	if err != nil {
		log.Fatal(err)
	}
	if shouldRebuild == false {
		log.Fatal("1: Expected rebuild true, got false")
	}

	shouldRebuild, err = NewBuilder(imageConfigName, imageConf, imageTag).ShouldRebuild(cache, false)
	if err != nil {
		log.Fatal(err)
	}
	if shouldRebuild == true {
		log.Fatal("2: Expected rebuild false, got true")
	}

	imageConf.Image = "test-image-new"
	shouldRebuild, err = NewBuilder(imageConfigName, imageConf, imageTag).ShouldRebuild(cache, false)
	if err != nil {
		log.Fatal(err)
	}
	if shouldRebuild == false {
		log.Fatal("3: Expected rebuild true, got false")
	}
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
