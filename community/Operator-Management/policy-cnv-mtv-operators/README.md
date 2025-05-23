# Install OpenShift Virtualization operator in the Fleet

This Policy for Red Hat Advanced Cluster Management for Kubernetes(ACM) includes an Operator policy. This policy deploys Red Hat operators on the clusters managed by ACM (ManagedClusters). The included placement uses labels to determine which ManagedCluster to apply the Operator policy to. The Operator Policy deploys the OpenShift Virtualization operator & the Migration Toolkit for Virtualization operator and include a Configuration Policy to validate core controllers are online.

Two policies will appear in Advanced Cluster Managements **Governance** console (Polcies):
* openshift-virtualization `policy-cnv.yaml`
* openshift-virtualization-mtv  `policy-mtv.yaml`

* To choose a ManagedCluster as a target for this Container Native Virtualization Operator Policy by applying the label `acm/cnv-operator-install: "true"` to a ManagedCluster in ACM
* The openshift-virtualization-mtv policy just targets the Advanced Cluster Management's hub cluster, via the `local-cluster: true` label

### Deploy the Policy
Apply the policies to the ACM Hub, then add the label `acm/cnv-operator-install=true` to the ManagedClusters where you want to deploy Container Native Virtualization operator (use the console/api/oc-cli)
```
oc apply -f ./
```

Then bind the `default` **ManagedClusterSet** to the `default` namespace, so that placement can resolve your clusters

You may customize the hyperconverged settings in the `policy-cnv.yaml` file. Where you will find:
  1. Subscription resource details
  2. ClusterServiceVersion resource details
  3. HyperConverged resource details
  4. HostPathProvisioner resource details
  5. Other status checks that validate the OpenShift Virtualization operator deployed correctly.
  6. NEW resources can be added to policy to customize the deployment of OpenShift Virtualization

To customize any of these resources in the policy, refer to the [OpenShift Virtualization documentation](https://docs.openshift.com/container-platform/4.17/virt/install/installing-virt.html#virt-subscribing-cli_installing-virt).

