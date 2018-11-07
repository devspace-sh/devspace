package configure

const defaultStableRepoURL = "https://kubernetes-charts.storage.googleapis.com"

const packageComment = `
# Here you can specify the subcharts values (for more information see: https://github.com/helm/helm/blob/master/docs/chart_template_guide/subcharts_and_globals.md#overriding-values-from-a-parent-chart)
`

const defaultPackageResourceReset = `
  resources:
    limit:
      cpu: 0
      memory: 0
    requests:
      cpu: 0
      memory: 0`

type packageDefault struct {
	serviceSelectors map[string]string
	values           string
}

var packageDefaultMap = map[string]packageDefault{
	"mysql": packageDefault{
		values: `` + defaultPackageResourceReset,
	},
	"mariadb": packageDefault{
		serviceSelectors: map[string]string{
			"app": "mariadb",
		},
		values: `` + defaultPackageResourceReset,
	},
	"influxdb": packageDefault{
		values: `
  setDefaultUser:
    enabled: true
    user:
      username: "admin"
      password: "password"` + defaultPackageResourceReset,
	},
}
