package pieButtonExecutionAdapter

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

// launchViaURI attempts to launch an application using its URI.
func launchViaURI(appNameKey string, uri string) error {
	if uri == "" {
		return fmt.Errorf("URI is empty for app '%s'", appNameKey) // Should not happen if called correctly
	}
	// Assumes Windows "start" command for URI handling.
	cmd := exec.Command("cmd", "/C", "start", "", uri)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start '%s' (URI: %s) via shell: %w", appNameKey, uri, err)
	}
	log.Printf("Attempted to start '%s' via URI handler: %s", appNameKey, uri)
	return nil
}

// buildExecCmd configures an *exec.Cmd for launching a traditional executable.
func buildExecCmd(actualExePath string, workingDir string, args string) (*exec.Cmd, error) {
	if actualExePath == "" {
		return nil, fmt.Errorf("executable path is empty")
	}

	cmd := exec.Command(actualExePath)

	if workingDir != "" {
		if !filepath.IsAbs(workingDir) {
			cmd.Dir = filepath.Join(filepath.Dir(actualExePath), workingDir)
		} else {
			cmd.Dir = workingDir
		}
	} else {
		cmd.Dir = filepath.Dir(actualExePath)
	}

	if args != "" {
		// For robust parsing of args with quotes, consider a dedicated library.
		parsedArgs := strings.Fields(args)
		cmd.Args = append([]string{actualExePath}, parsedArgs...)
	} else {
		cmd.Args = []string{actualExePath} // Ensure cmd.Args[0] is the command itself
	}
	return cmd, nil
}
