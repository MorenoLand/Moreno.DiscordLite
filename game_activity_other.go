//go:build !windows

package main

func enumerateGameProcesses(_ ...string) ([]gameProcess, error) {
	return []gameProcess{}, nil
}
