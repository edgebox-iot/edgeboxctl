package diagnostics

type ReleaseVersion string

var Version string
var Commit string
var BuildDate string

const (
	DEV_VERSION   ReleaseVersion = "dev"
	PROD_VERSION  ReleaseVersion = "prod"
	CLOUD_VERSION ReleaseVersion = "cloud"
	OTHER_VERSION ReleaseVersion = "other"
)

func GetReleaseVersion() ReleaseVersion {
	return ReleaseVersion(Version)
}
