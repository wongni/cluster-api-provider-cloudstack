# Testing

This document is to help developers understand how to test CAPC.

## Code Origin

The most of the code under test/e2e is from CAPD (Cluster API for Docker) e2e testing (https://github.com/kubernetes-sigs/cluster-api/tree/main/test/e2e)
The ACS specific things are under test/e2e/config and test/e2e/data/infrastructure-cloudstack. 

## e2e

This section describes how to run end-to-end (e2e) testing with CAPC.

### Requirements

* Admin access to a Apache CloudStack (ACS) server
* The testing must occur on a host that can access the ACS server
* Docker ([download](https://www.docker.com/get-started))
* Kind ([download](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-with-a-package-manager))

### Environment variables

The first step to running the e2e tests is setting up the required environment variables:

| Environment variable                        | Description                                                                      | Default Value               |
|---------------------------------------------|----------------------------------------------------------------------------------|-----------------------------|
| `CLOUDSTACK_ZONE_NAME`                      | The zone name                                                                    | `zone1`                     |
| `CLOUDSTACK_NETWORK_NAME`                   | The network name. If not exisiting an isolated network with the name is created. | `Shared1`                   |
| `CLUSTER_ENDPOINT_IP`                       | The cluster endpoint IP                                                          | `172.16.2.199`              |
| `CLUSTER_ENDPOINT_IP_2`                     | The cluster endpoint IP for a second cluster                                     | `172.16.2.199`              |
| `CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING` | The machine offering for the control plane VM instances                          | `Large Instance`            |
| `CLOUDSTACK_WORKER_MACHINE_OFFERING`        | The machine offering for the worker node VM instances                            | `Medium Instance`           |
| `CLOUDSTACK_TEMPLATE_NAME`                  | The machine template for both control plane and worke node VM instances          | `kube-v1.20.10/ubuntu-2004` |
| `CLOUDSTACK_SSH_KEY_NAME`                   | The name of SSH key added to the VM instances                                    | `CAPCKeyPair6`              |

You also have to create `cloud-config.yaml` file with the following format to create K8S secrets.

```
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: default
type: Opaque
stringData:
  api-key: xxx
  secret-key: xxx
  api-url: http://10.11.0.1:8080/client/api
  verify-ssl: "false"
---
apiVersion: v1
kind: Secret
metadata:
  name: secret2
  namespace: default
type: Opaque
stringData:
  api-key: xxx
  secret-key: xxx
  api-url: http://10.11.0.2:8080/client/api
  verify-ssl: "false"
---
```

The api-key and secret-key can be found or generated at Home > Accounts > admin > Users > admin of the ACS management UI. `verify-ssl` is an optional flag and its default value is true. CAPC skips verifying the host SSL certificates when the flag is set to false.

### Running the e2e tests

Run the following command to execute the CAPC e2e tests:

```shell
make run-e2e
```
This command runs all e2e test cases.

You can specify JOB environment variable which value is a regular expression to select test cases to execute. 
For example, 

```shell
JOB=PR-Blocking make run-e2e
```
This command runs the e2e tests that contains `PR-Blocking` in their spec names. 

### Debugging the e2e tests
The E2E tests can be debugged by attaching a debugger to the e2e process after it is launched (*i.e., make run-e2e*).
To facilitate this, the E2E tests can be run with environment variable PAUSE_FOR_DEBUGGER_ATTACH=true.
(This is only strictly needed when you want the debugger to break early in the test process, i.e., in SynchronizedBeforeSuite.
There's usually quite enough time to attach if you're not breaking until your actual test code runs.)

When this environment variable is set to *true* a 15s pause is inserted at the beginning of the test process
(i.e., in the SynchronizedBeforeSuite).  The workflow is:
- Launch the e2e test: *PAUSE_FOR_DEBUGGER_ATTACH=true JOB=MyTest make run-e2e*
- Wait for console message: *Pausing 15s so you have a chance to attach a debugger to this process...*
- Quickly attach your debugger to the e2e process (i.e., e2e.test)
