package tanka

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/log"
)

func TestNewTankaEnvironment(t *testing.T) {
	type args struct {
		config *latest.TankaConfig
	}
	tests := []struct {
		name string
		args args
		want TankaEnvironment
	}{
		{
			name: "HydrateArguments",
			args: args{
				config: &latest.TankaConfig{
					TankaBinaryPath: "",
					Path:            "my",
					EnvironmentPath: "my/tanka/environment",
					EnvironmentName: "my-env-name",
					ExternalCodeVariables: map[string]string{
						"MY_CODE_ARG": "true",
					},
					ExternalStringVariables: map[string]string{
						"MY_STR_ARG":    "my-ext-var-string",
						ExtVarName:      "my-devspace-name",
						ExtVarNamespace: "my-devspace-namespace",
					},
					TopLevelCode:   []string{"true"},
					TopLevelString: []string{"my-tla-string"},
					Target:         "*",
				},
			},
			want: &tankaEnvironmentImpl{
				name:         "my-devspace-name",
				namespace:    "my-devspace-namespace",
				tkBinaryPath: tkDefaultCommand,
				jbBinaryPath: jbDefaultCommand,
				rootDir:      "my",
				args: []string{
					"my/tanka/environment",
				},
				flags: []string{
					"--name=my-env-name",
					"--ext-code=MY_CODE_ARG=true",
					"--ext-str=MY_STR_ARG=my-ext-var-string",
					"--ext-str=DEVSPACE_NAME=my-devspace-name",
					"--ext-str=DEVSPACE_NAMESPACE=my-devspace-namespace",
					"--tla-code=true",
					"--tla-str=my-tla-string",
					"--target=*",
				},
				stdout: os.Stdout,
				stderr: os.Stderr,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTankaEnvironment(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					cmp.Diff(
						got,
						tt.want,
						// os.File requires unexported fields for comparing on Windows
						cmp.AllowUnexported(tankaEnvironmentImpl{}), cmp.AllowUnexported(os.File{}),
					),
				)
			}
		})
	}
}

func getCtx() devspacecontext.Context {
	return devspacecontext.NewContext(context.Background(), nil, &log.DiscardLogger{})
}

func newEchoTankaEnv() (*tankaEnvironmentImpl, *bytes.Buffer) {
	out := new(bytes.Buffer)
	return &tankaEnvironmentImpl{
		args:         []string{"--this-is-a-tanka-flag"},
		tkBinaryPath: "echo",
		jbBinaryPath: "echo",
		stdout:       out,
		stderr:       out,
	}, out
}

func Test_tankaEnvironmentImpl_Apply(t *testing.T) {

	want := "apply --this-is-a-tanka-flag --auto-approve=always\n"

	tkEnv, out := newEchoTankaEnv()
	err := tkEnv.Apply(getCtx())
	if err != nil {
		t.Error(err)
	}
	got := out.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Diff(t *testing.T) {

	want := "diff --this-is-a-tanka-flag --exit-zero --summarize\n"

	tkEnv, _ := newEchoTankaEnv()
	got, err := tkEnv.Diff(getCtx())
	if err != nil {
		t.Error(err)
	}

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Show(t *testing.T) {

	want := "show --this-is-a-tanka-flag --dangerous-allow-redirect\n"

	tkEnv, _ := newEchoTankaEnv()
	buf := new(bytes.Buffer)
	err := tkEnv.Show(getCtx(), buf)
	if err != nil {
		t.Error(err)
	}
	got := buf.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Prune(t *testing.T) {

	want := "prune --this-is-a-tanka-flag --auto-approve=always\n"

	tkEnv, out := newEchoTankaEnv()
	err := tkEnv.Prune(getCtx())
	if err != nil {
		t.Error(err)
	}
	got := out.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Delete(t *testing.T) {

	want := "delete --this-is-a-tanka-flag --auto-approve=always\n"

	tkEnv, out := newEchoTankaEnv()
	err := tkEnv.Delete(getCtx())
	if err != nil {
		t.Error(err)
	}
	got := out.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Install(t *testing.T) {

	want := "install\n"

	tkEnv, out := newEchoTankaEnv()
	err := tkEnv.Install(getCtx())
	if err != nil {
		t.Error(err)
	}
	got := out.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}

func Test_tankaEnvironmentImpl_Update(t *testing.T) {

	want := "update\n"

	tkEnv, out := newEchoTankaEnv()
	err := tkEnv.Update(getCtx())
	if err != nil {
		t.Error(err)
	}
	got := out.String()

	if got != want {
		t.Errorf(cmp.Diff(got, want))
	}
}
