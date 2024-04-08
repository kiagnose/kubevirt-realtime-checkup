# kubevirt-realtime-checkup

An automated test checking the readiness of a KubeVirt cluster to run virtualized realtime workloads.

## Permissions

You need to be a namespace-admin in order to execute this checkup.
The checkup requires the following permissions:

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: realtime-checkup-sa
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kiagnose-configmap-access
rules:
  - apiGroups: [ "" ]
    resources: [ "configmaps" ]
    verbs: [ "get", "update" ]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kiagnose-configmap-access
subjects:
  - kind: ServiceAccount
    name: realtime-checkup-sa
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kiagnose-configmap-access
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kubevirt-realtime-checker
rules:
  - apiGroups: [ "kubevirt.io" ]
    resources: [ "virtualmachineinstances" ]
    verbs: [ "create", "get", "delete" ]
  - apiGroups: [ "subresources.kubevirt.io" ]
    resources: [ "virtualmachineinstances/console" ]
    verbs: [ "get" ]
  - apiGroups: [ "" ]
    resources: [ "configmaps" ]
    verbs: [ "create", "delete" ]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kubevirt-realtime-checker
subjects:
  - kind: ServiceAccount
    name: realtime-checkup-sa
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kubevirt-realtime-checker
```

## Configuration

| Key                                          | Description                                                     | Is Mandatory | Remarks                                                       |
|----------------------------------------------|-----------------------------------------------------------------|--------------|---------------------------------------------------------------|
| spec.timeout                                 | How much time before the checkup will try to close itself       | True         |                                                               |
| spec.param.vmUnderTestContainerDiskImage     | VM under test container disk image                              | True         |                                                               |
| spec.param.vmUnderTestTargetNodeName         | Node Name on which the VM under test will be scheduled to       | False        | Assumed to be configured to nodes that allow realtime traffic |
| spec.param.oslatDuration                     | How much time will the oslat program run                        | False        | Defaults to TBD                                               |
| spec.param.oslatLatencyThresholdMicroSeconds | A latency higher than this value will cause the checkup to fail | False        | Defaults to TBD                                               |

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: realtime-checkup-config
data:
  spec.timeout: 10m
  spec.param.vmUnderTestContainerDiskImage: quay.io/kiagnose/kubevirt-realtime-checkup-vm:main
  spec.param.oslatDuration: 1h
```

## Execution

In order to execute the checkup, fill in the required data and apply this manifest:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: realtime-checkup
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccountName: realtime-checkup-sa
      restartPolicy: Never
      containers:
        - name: realtime-checkup
          image: quay.io/kiagnose/kubevirt-realtime-checkup:main
          imagePullPolicy: Always
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop: [ "ALL" ]
            runAsNonRoot: true
            seccompProfile:
              type: "RuntimeDefault"
          env:
            - name: CONFIGMAP_NAMESPACE
              value: <target-namespace>
            - name: CONFIGMAP_NAME
              value: realtime-checkup-config
            - name: POD_UID
              valueFrom:
                fieldRef:
                  fieldPath: metadata.uid
```

## Checkup Results Retrieval

After the checkup Job had completed, the results are made available at the user-supplied ConfigMap object:

```bash
kubectl get configmap realtime-checkup-config -n <target-namespace> -o yaml
```

| Key                                       | Description                                                      | Remarks  |
|-------------------------------------------|------------------------------------------------------------------|----------|
| status.succeeded                          | Specifies if the checkup is successful (`true`) or not (`false`) |          |
| status.failureReason                      | The reason for failure if the checkup fails                      |          |
| status.startTimestamp                     | The time when the checkup started                                | RFC 3339 |
| status.completionTimestamp                | The time when the checkup has completed                          | RFC 3339 |
| status.result.vmUnderTestActualNodeName   | The node on which the VM under test was scheduled                |          |
| status.result.oslatMaxLatencyMicroSeconds | Actual oslat maximum measured latency                            |          |
