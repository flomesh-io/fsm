package bugreport

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	successIndicator = "\u2713" // check mark, used for success
	failureIndicator = "x"      // cross mark, used for failure
	commandsDirName  = "commands"
)

// Run generates a bug report from the given config
func (c *Config) Run() error {
	// Check prerequisites
	if err := checkPrereq(); err != nil {
		return err
	}

	// Create staging directory
	stagingDir, err := ioutil.TempDir("", "fsm_bug_report_")
	if err != nil {
		return fmt.Errorf("Error creating temp directory needed for creating bug report. Aborting: %w", err)
	}
	c.stagingDir = stagingDir
	fmt.Fprintf(c.Stdout, "[+] Created staging dir %s to generate bug report\n", stagingDir)
	c.endSection()

	// Generate report for mesh namespaces
	fmt.Fprintf(c.Stdout, "[+] Collecting information about meshed namespaces\n")
	if err := c.initRootNamespaceDir(); err != nil {
		c.completionFailure("Error initializing root namespaces dir. Aborting")
	}
	c.collectMeshedNamespaceReport()
	c.completionSuccess("Finished generating report for meshed namespaces")
	c.endSection()

	// Generate report for each app namespace
	fmt.Fprintf(c.Stdout, "[+] Collecting information from individual app namespaces\n")
	c.collectPerNamespaceReport()
	c.completionSuccess("Finished generating report for individual app namespaces")
	c.endSection()

	// Generate report for each app pod
	c.AppPods = c.getUniquePods()
	fmt.Fprintf(c.Stdout, "[+] Collecting information from individual app pods\n")
	c.collectPerPodReport()
	c.completionSuccess("Finished generating report for individual app pods")
	c.endSection()

	// Generate report for control plane pods
	fmt.Fprintf(c.Stdout, "[+] Collecting information from control plane\n")
	c.collectControlPlaneLogs()
	c.completionSuccess("Finished generating report for control plane")
	c.endSection()

	// Generate report for ingress
	if c.CollectIngress {
		fmt.Fprintf(c.Stdout, "[+] Collecting ingress information\n")
		c.collectIngressReport()
		c.collectIngressControllerReport()
		c.completionSuccess("Finished generating report for ingress")
		c.endSection()
	}

	// Generate output file if not provided
	if c.OutFile == "" {
		outFd, err := ioutil.TempFile("", "*_fsm-bug-report.tar.gz")
		if err != nil {
			c.completionFailure("Error creating temp file for bug report")
			return fmt.Errorf("Error creating bug report: %w", err)
		}
		c.OutFile = outFd.Name()
	}

	// Archive it
	fmt.Fprintf(c.Stdout, "[+] Collecting information from individual app namespaces\n")
	if err := c.archive(stagingDir, c.OutFile); err != nil {
		c.completionFailure("Error archiving bug report")
		return fmt.Errorf("Error creating bug report: %w", err)
	}
	// Remove staging dir
	if err := os.RemoveAll(c.stagingDir); err != nil {
		c.completionFailure("Error removing staging dir %s, err: %s", c.stagingDir, err)
	}
	c.endSection()

	return nil
}

func checkPrereq() error {
	requiredTools := []string{"fsm", "kubectl"}
	for _, tool := range requiredTools {
		if !pathExists(tool) {
			return fmt.Errorf("Prerequisite not met: %s not found", tool)
		}
	}
	return nil
}

func pathExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (c *Config) endSection() {
	fmt.Fprint(c.Stdout, "\n\n")
}

func (c *Config) completionSuccess(format string, a ...interface{}) {
	fmt.Fprintf(c.Stdout, "%s %s\n", successIndicator, fmt.Sprintf(format, a...))
}

func (c *Config) completionFailure(format string, a ...interface{}) {
	fmt.Fprintf(c.Stderr, "%s %s\n", failureIndicator, fmt.Sprintf(format, a...))
}

func runCmdAndWriteToFile(cmdList []string, outFile string) error {
	if len(cmdList) == 0 {
		return fmt.Errorf("Atleast 1 command must be provided, none provided")
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outFile), 0700); err != nil {
		return fmt.Errorf("Error creating parent directory for path: %s: %w", outFile, err)
	}

	cmd := exec.Command(cmdList[0], cmdList[1:]...) //#nosec G204

	// open the out file for writing
	outfile, err := os.Create(filepath.Clean(outFile))
	if err != nil {
		return err
	}
	//nolint: errcheck
	//#nosec G307
	defer outfile.Close()
	cmd.Stdout = outfile
	cmd.Stderr = outfile

	return cmd.Run()
}
