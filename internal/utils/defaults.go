package utils

const DefaultPort = 9000
const ConfigFilePathAndName = "config.yaml"

var (
	CurrentPort               int
	AcceptedConfigFileFormats = []string{"yaml", "yml"}
	Version                   = "v1.0.0"
)
