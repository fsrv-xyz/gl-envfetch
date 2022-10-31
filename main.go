package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/xanzy/go-gitlab"
	"gopkg.in/alecthomas/kingpin.v2"
)

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func main() {
	app := kingpin.New(os.Args[0], "A command line tool to start a shell with gitlab environment variables set")
	projectId := app.Flag("project", "project id").Required().String()
	gitlabToken := app.Flag("token", "gitlab token").Envar("GL_TOKEN").Required().String()
	gitlabServer := app.Flag("server", "gitlab server e.g. gitlab.com").Envar("GL_SERVER").Required().String()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	gitlabServerURL, gitlabURLParseError := url.Parse(*gitlabServer)
	if gitlabURLParseError != nil {
		log.Fatal(gitlabURLParseError)
	}

	if gitlabServerURL.Host == "" {
		log.Fatal("Invalid gitlab server url")
	}
	fmt.Printf("Connecting to %#v\n", gitlabServerURL.String())

	client, clientInstantiationError := gitlab.NewClient(*gitlabToken, gitlab.WithBaseURL(gitlabServerURL.JoinPath("/api/v4").String()))
	if clientInstantiationError != nil {
		log.Fatal(clientInstantiationError)
	}

	vars, _, varsFetchError := client.ProjectVariables.ListVariables(*projectId, nil)
	if varsFetchError != nil {
		log.Fatal(varsFetchError)
	}

	if len(vars) == 0 {
		fmt.Println(color.RedString("No variables found"))
		os.Exit(1)
	}

	color.New(color.FgHiGreen, color.Bold).Printf("Found %d variables:\n", len(vars))
	alreadyDefinedVars := 0
	for _, v := range vars {
		if _, found := os.LookupEnv(v.Key); found {
			fmt.Println("  ", color.YellowString(v.Key))
			alreadyDefinedVars++
		} else {
			fmt.Println("  ", color.GreenString(v.Key))
		}
	}

	if alreadyDefinedVars > 0 {
		fmt.Println(color.RedString("Some variables are already defined in local environment"))
		if !askForConfirmation("Do you want to override them?") {
			os.Exit(1)
		}
	}

	if !askForConfirmation("Do you want to continue?") {
		os.Exit(0)
	}

	cmd := exec.Command(os.Getenv("SHELL"))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	for _, v := range vars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", v.Key, v.Value))
	}
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
