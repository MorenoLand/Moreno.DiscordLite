//go:build !windows

package main

func enumerateGameProcesses(_ gameCandidateIndex, _ map[string]string) ([]gameProcess, error) {
	return []gameProcess{}, nil
}
