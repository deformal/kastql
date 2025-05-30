package types

type Command struct{}
type Flag string

const (
	Serve Command = "serve"
)

var CommandList = []Command{
	Serve,
}

var ServeFlagsList = []Flag{}
