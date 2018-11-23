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
		values: `
  mysqlRootPassword: "YOUR_ROOT_PASSWORD"    # only set when first starting the mysql server
  mysqlDatabase: "YOUR_DATABASE_NAME"
  mysqlUser: "YOUR_USERNAME"                 # default user for the database
  mysqlPassword: "YOUR_PASSWORD"             # only set when first starting the mysql server
  persistence:
    enabled: true
    size: 3Gi` + defaultPackageResourceReset,
	},
	"mariadb": packageDefault{
		serviceSelectors: map[string]string{
			"app": "mariadb",
		},
		values: `
  rootUser:
    password: "YOUR_ROOT_PASSWORD"           # only set when first starting the mysql server
  db:
    name: "YOUR_DATABASE_NAME"
    user: "YOUR_USERNAME"
    password: "YOUR_PASSWORD"                # only set when first starting the mysql server
  replication:
    enabled: true
  master:
    persistence:
      enabled: true
      size: 3Gi
  slave:
    replicas: 1
    persistence:
      enabled: true
      size: 3Gi`,
	},
	"influxdb": packageDefault{
		values: `
  setDefaultUser:
    enabled: true
    user:
      username: "YOUR_USERNAME"
      password: "YOUR_PASSWORD"              # only set when first starting the mysql server
  persistence:
    enabled: true
    size: 3Gi` + defaultPackageResourceReset,
	},
	"mongodb": packageDefault{
		values: `
  mongodbRootPassword: "YOUR_ROOT_PASSWORD"  # only set when first starting the mysql server
  mongodbDatabase: "YOUR_DATABASE_NAME"
  mongodbUsername: "YOUR_USERNAME"
  mongodbPassword: "YOUR_PASSWORD"           # only set when first starting the mysql server
  mongodbDatabase:
    enabled: true
    user:
      username: "YOUR_USERNAME"
      password: "YOUR_PASSWORD"
  persistence:
    enabled: true
    size: 3Gi`,
	},
	"redis": packageDefault{
		values: `
  cluster:
    enabled: true
    slaveCount: 1`,
	},
}
