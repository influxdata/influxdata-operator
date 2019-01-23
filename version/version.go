package version

var (
	Version      = "0.0.2"
	BuildVersion = ""
)

func GetBuildVersion() string {
	return BuildVersion
}
