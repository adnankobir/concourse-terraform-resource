package terraform

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/adnankobir/concourse-terraform-resource/internal/ssh"
	"github.com/adnankobir/concourse-terraform-resource/internal/types"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func addAnsibleVar(args []string, name string, val string) []string {
	if name != "" {
		return append(args, "-e", fmt.Sprintf("%s=%s", name, val))
	}
	return append(args, "-e", fmt.Sprintf(`"%s"`, val))
}

func parseEnvironment(env *types.Environment) error {
	if err := envconfig.Process("", env); err != nil {
		return fmt.Errorf("failed to parse concourse environment: %v", err)
	}
	return nil
}

func setupLogging(stderr io.Writer) {
	logrus.SetOutput(stderr)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}

func setupSSH(key string) (*ssh.Agent, error) {
	agent, err := ssh.SpawnAgent()
	if err != nil {
		return agent, fmt.Errorf("failed to spawn ssh agent: %v", err)
	}
	err = agent.AddKey([]byte(key))
	if err != nil {
		log.Fatalf("Failed to add private key: %v", err)
	}
	err = os.Setenv("SSH_AUTH_SOCK", agent.SSHAuthSock())
	if err != nil {
		log.Fatalf("Failed to set agent forwarding: %v", err)
	}
	cmd := exec.Command("ssh-keyscan", "-H", "github.com", ">>", "~/.ssh/known_hosts")
	//cmd.Stdout = os.Stderr
	cmd.Stdout = nil
	cmd.Stderr = nil
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to add github.com to known hosts: %v", err)
	}
	return agent, err
}
