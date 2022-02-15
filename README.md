# GCP Fetch Redis Certs

Utility to fetch the certificate authority certificate for GCP cloud provider managed redis instances. GCP uses a self-signed CA for signing managed redis instance server certificates. This allows GCP to offer redis instances that have private dns names. However, this poses an issue for client connections as they do not trust the presented server certificate.

This utility addresses that issue by fetching the CA certificate for the managed redis instance from the gcp api. It uses existing gcloud or workload identity credentials to authenticate to the gcp api. The CA certificates are then written to a file.

This has been designed to run as an init pod to fetch the certificates and write them to a shared volume for the application.

## Usage

```shell
Usage of ./gcp-fetch-redis-certs:
  -caFile string
        File to write redis CA certificates to. (default "/certs/redis_ca.crt")
  -project string
        GCP project name of redis instance. (default "metadata")
  -redisInstance string
        Redis instance name. (default "redis-test-01")
  -region string
        GCP region of redis instance. (default "metadata")
```

- This requires permissions to read the workload identity, pod metadata, and describe a redis instance.


### Example Init Pod Template

When using this with kubernetes, a suggested pattern is to run it as an init container. This will:

- Create a memory backed empty volume, `/certs`, shared between the init container and any running containers.
- Use the gcp workload identity assigned to the `serviceAccountName` to read the instance details of `redis-test-01`
- Write all of the gcp self-signed redis CA certificates to `/certs/redis_ca.crt` and exit 0.
- When the primary containers start up they can then read the CA certificates from `/certs/redis_ca.crt` to validate connections to `redis-test-01`.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  labels:
    app: example
spec:
  containers:
  - name: example-container
    image: busybox:1.28
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
    volumeMounts:
    - mountPath: /certs
      name: redis-certs
      readOnly: true
  initContainers:
  - name: fetch-redis-certs
    image: derektamsen/gcp-fetch-redis-certs:0.0.1
    args: ['-redisInstance', 'redis-test-01']
    volumeMounts:
    - mountPath: /certs
      name: redis-certs
  serviceAccountName: k8s-redis-service-account
  volumes:
  - emptyDir:
      medium: Memory
    name: redis-certs
```
