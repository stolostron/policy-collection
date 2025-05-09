# Virtual Machine Policy YAML Definitions

This directory contains YAML definitions for creating virtual machines using Open Cluster Management policies.

## Prerequisites
Before applying the YAML, ensure the following:
1. You have access to an OpenShift cluster with Open Cluster Management installed.
2. The `openshift-virtualization` operator is installed and configured.

## Instructions

### 1. Update the YAML File
Make the following changes to the `fedora-rootdisk-and-datadisk.yaml` file:
- **Virtual Machine Name**: Update the `metadata.name` field to set a unique name for your virtual machine(s).
- **Password**: Set a secure password in the `userData` section under `cloudInitNoCloud`.
- **SSH Authorized Keys (Optional)**: Uncomment and add your SSH public key(s) in the `ssh_authorized_keys` section if SSH access is required.
- **Remediation Action**: Change `remediationAction: inform` to `remediationAction: enforce` to enable automatic creation of the virtual machine(s).

### 2. Apply the YAML
Run the following command to apply the YAML file:
```bash
oc apply -f fedora-rootdisk-and-datadisk.yaml
```

### 3. Label the ManagedCluster
To trigger the creation of the virtual machine(s), add the following label to the target ManagedCluster resource:
```bash
oc label managedcluster <cluster-name> acm/cnv-create-vm-sample=true
```
Alternatively, you can add this label through the OpenShift console.

### Notes
The policy contains two virtual machine template definitions:
* One with a root disk and a data disk.
* One with only a root disk.

### Additional Information
* Ensure that the namespace specified in the YAML matches the namespace where the resources will be deployed.
* Verify that the acm/cnv-create-vm-sample label is applied to the correct cluster(s) to avoid unintended resource creation.
* For more details, refer to the Open Cluster Management documentation.