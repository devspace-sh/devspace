package tanka

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func Test_validateConfig(t *testing.T) {
	type args struct {
		cfg *latest.DeploymentConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "tanka not defined",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: nil},
			},
			wantErr: true,
		},
		{
			name: "path and environmentPath not defined",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					EnvironmentName: "test",
				}},
			},
			wantErr: true,
		},
		{
			name: "environmentName and environmentPath missing",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					Path: "./kubernetes/",
				}},
			},
			wantErr: true,
		},
		{
			name: "welldefined with path and environmentName",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					Path:            "./kubernetes/",
					EnvironmentName: "test",
				}},
			},
			wantErr: false,
		},
		{
			name: "welldefined with path and environmentPath",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					Path:            "./kubernetes/",
					EnvironmentPath: "environments/devspace",
				}},
			},
			wantErr: false,
		},
		{
			name: "welldefined with path, environmentPath and name",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					Path:            "./kubernetes/",
					EnvironmentPath: "environments/production",
					EnvironmentName: "devspace/my-demo-app",
				}},
			},
			wantErr: false,
		},
		{
			name: "welldefined with environmentPath and name and no path",
			args: args{
				cfg: &latest.DeploymentConfig{Tanka: &latest.TankaConfig{
					EnvironmentPath: "./kubernetes/environments/production",
					EnvironmentName: "devspace/my-demo-app",
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateConfig(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
