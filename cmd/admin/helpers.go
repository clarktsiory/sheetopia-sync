package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func input(prompt string) string {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func inputPassword(prompt string) string {
	fmt.Printf("%s: ", prompt)
	password, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	return string(password)
}
