package framework

import (
	"fmt"
)

// HelmInstallFSM installs an fsm control plane using the fsm chart which lives in charts/fsm
func (td *FsmTestData) HelmInstallFSM(release, namespace string) error {
	if td.InstType == KindCluster {
		if err := td.LoadFSMImagesIntoKind(); err != nil {
			return err
		}
	}

	values := fmt.Sprintf("fsm.image.registry=%s,fsm.image.tag=%s,fsm.meshName=%s", td.CtrRegistryServer, td.FsmImageTag, release)
	args := []string{"install", release, "../../charts/fsm", "--set", values, "--namespace", namespace, "--create-namespace", "--wait"}
	stdout, stderr, err := td.RunLocal("helm", args...)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return fmt.Errorf("failed to run helm install with fsm chart: %s", stderr)
	}

	return nil
}

// DeleteHelmRelease uninstalls a particular helm release
func (td *FsmTestData) DeleteHelmRelease(name, namespace string) error {
	args := []string{"uninstall", name, "--namespace", namespace}
	_, _, err := td.RunLocal("helm", args...)
	if err != nil {
		td.T.Fatal(err)
	}
	return nil
}
