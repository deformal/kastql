package types

type WelcomeMessageResponse struct {
	Command string
	Flags   []string
}

type CommandFlags struct {
	Name        string
	Description string
}

type Command struct {
	Name        string
	Description string
	Flags       *map[string]CommandFlags
}

const (
	Serve  = "serve"
	Status = "status"
)

const (
	ServeCommandPortFlag   = "port"
	ServeCommandConfigFlag = "config"
	ServeCommandHelpFlag   = "help"
)
