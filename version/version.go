package version

var (
	Version = "0.0.1"
	BuildVersion = ""
)

func GetBuildVersion() string {
	return BuildVersion
}