# Applying RBAC to ACM managed CNV clusters
OpenShift Virtualization has three predefined roles that you can bind to users or groups.
  * kubevirt.io:view
  * kubevirt.io:edit
  * kubevirt.io:admin

When OIDC is configured, supplying a `RoleBinding` or `ClusterRoleBinding`, with the **subject** and **clusterRole / role** allows the subject access to virtual machines in the namespace or on the cluster. The included policy creates users and assigns those users as **subjects** in a `clusterRoleBinding` (cluster wide) with the **ClusterRole** permissions of **kubevirt.io:edit**.

Any cluster that used the OpenShift Virtualization operator policy provided here, by adding the label `acm/cnv-operator-install: "true"` will also receive the `clusterRoleBinding` and **Users** defined in this policy if it is applied to ACM

### Deploythe policy
```
  oc apply -f ./acm-kubevirt-edit-fleet.yaml
```

### Notes
* This will be a ClusterPermission in Advanced Cluster Managemenet for Kubernetes v2.15