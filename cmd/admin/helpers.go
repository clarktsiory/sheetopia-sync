package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func input(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func inputPassword(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// without a terminal there is no echo to disable, e.g. when running `docker compose exec` without -t
		return input(prompt)
	}

	fmt.Printf("%s: ", prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}

func inputNewPassword(prompt string) (string, error) {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	for {
		password, err := inputPassword(prompt)
		if err != nil {
			return "", err
		}
		if password == "" {
			if !interactive {
				return "", errors.New("password must not be empty")
			}
			fmt.Println("Password must not be empty. Try again.")
			continue
		}

		// piped input cannot be mistyped, so only ask for confirmation when a terminal is attached
		if !interactive {
			return password, nil
		}

		repeat, err := inputPassword("Repeat password")
		if err != nil {
			return "", err
		}
		if password != repeat {
			fmt.Println("Passwords don't match. Try again.")
			continue
		}
		return password, nil
	}
}
