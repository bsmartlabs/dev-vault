package cli

import "io"

type commandRoute struct {
	name  string
	usage func(io.Writer)
	run   func(commandContext, []string) int
}

var commandRoutes = []commandRoute{
	{name: "version", usage: printVersionUsage, run: runVersion},
	{name: "list", usage: printListUsage, run: runList},
	{name: "pull", usage: printPullUsage, run: runPull},
	{name: "push", usage: printPushUsage, run: runPush},
}

func commandNames() []string {
	names := make([]string, 0, len(commandRoutes))
	for _, route := range commandRoutes {
		names = append(names, route.name)
	}
	return names
}

func routeForCommand(name string) (commandRoute, bool) {
	for _, route := range commandRoutes {
		if route.name == name {
			return route, true
		}
	}
	return commandRoute{}, false
}

func usageForCommand(name string) (func(io.Writer), bool) {
	route, ok := routeForCommand(name)
	if !ok {
		return nil, false
	}
	return route.usage, true
}
