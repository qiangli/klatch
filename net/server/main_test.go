package main

import "testing"

func TestListenAddress(t *testing.T) {
	for input, want := range map[string]string{"7100": ":7100", "127.0.0.1:7100": "127.0.0.1:7100", "[::1]:7100": "[::1]:7100"} {
		if got := listenAddress(input); got != want {
			t.Errorf("%q: got %q, want %q", input, got, want)
		}
	}
}
