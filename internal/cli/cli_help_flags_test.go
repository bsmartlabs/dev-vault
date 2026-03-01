package cli

import "testing"

func TestHasPreCommandHelpFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "direct", args: []string{"-h"}, want: true},
		{name: "singleDashHelp", args: []string{"-help"}, want: true},
		{name: "withGlobalBefore", args: []string{"--config", "x", "-h"}, want: true},
		{name: "withGlobalEquals", args: []string{"--config=x", "--help"}, want: true},
		{name: "subcommandHelpNotGlobal", args: []string{"pull", "-h"}, want: false},
		{name: "endOfFlags", args: []string{"--", "-h"}, want: false},
	}
	for _, tc := range cases {
		if got := hasPreCommandHelpFlag(tc.args); got != tc.want {
			t.Fatalf("%s: expected %v got %v", tc.name, tc.want, got)
		}
	}
}
