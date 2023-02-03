# Configuration

There are three ways to configure the S3 GW:

1. CLI parameters
2. YAML file
3. Environment variables

Everything available as a CLI parameter can also be specified via environment variables and almost everything can be
specified via `.yaml` configuration file.

But **not vice versa**, some parameters can be configured only with environment variables/configuration file.
Most of these parameters have default values, therefore, these ways to configure the gateway are optional and
basic configuration can be completed with CLI parameters only.

1. [CLI parameters](#cli-parameters)
    1. [Nodes and weights](#nodes-and-weights)
    2. [Wallet](#wallet)
    3. [Binding and TLS](#listening-on-address-and-TLS)
    4. [RPC endpoint and resolving of bucket names](#rpc-endpoint-and-resolving-of-bucket-names)
    5. [Processing of requests](#processing-of-requests)
    6. [Connection to FrostFS](#connection-to-FrostFS)
    7. [Monitoring and metrics](#monitoring-and-metrics)
2. [YAML file and environment variables](#yaml-file-and-environment-variables)
    1. [Configuration file](#frostfs-s3-gateway-configuration-file)

## CLI parameters

### Nodes and weights

You can specify multiple `-p` options to add more FrostFS nodes; this will make
a gateway spread requests equally among them (using weight 1 for every node):

```shell
$ frostfs-s3-gw -p 192.168.130.72:8080 -p 192.168.130.71:8080
```

If you want some specific load distribution proportions, use weights and priorities, they
can only be specified via environment variables or a configuration file.

### Wallet

Wallet (`--wallet`) is a mandatory parameter. It is a path to a wallet file. You can provide a passphrase to decrypt
a wallet via env variable or conf file, or you will be asked to enter a password interactively.
You can also specify an account address to use from a wallet using the `--address` parameter.

### Listening on address and TLS

You can make the gateway listen on specific address using the `--listen_address` option.

It can also provide TLS interface for its users, just specify paths to the key and
certificate files via `--tls.key_file` and `--tls.cert_file` parameters. Note
that using these options makes gateway TLS-only. If you need to serve both TLS
and plain text, you either have to run two gateway instances or use some
external redirecting solution.

Example to bind to `192.168.130.130:443` and serve TLS there (keys and nodes are
omitted):

```shell
$ frostfs-s3-gw --listen_address 192.168.130.130:443 \
  --tls.key_file=key.pem --tls.cert_file=cert.pem
```

Using these flag you can configure only one address. To set multiple addresses use yaml config. 

### RPC endpoint and resolving of bucket names

To set RPC endpoint specify a value of parameter `-r` or `--rpc_endpoint`. The parameter is **required if** another
parameter's `--resolve_order` value contains `nns`.

```shell
$ frostfs-s3-gw --rpc_endpoint http://morph-chain.frostfs.devenv:30333/ --resolve_order nns,dns
```

### Processing of requests

Maximum number of clients whose requests can be handled by the gateway can be specified by the value of
`--max_clients_count` parameter.
`--max_clients_deadline` defines deadline after which the gate sends error `RequestTimeout` to a client.

```shell
$ frostfs-s3-gw --max_clients_count 150 --max_clients_deadline 1m
```

### Connection to FrostFS

Timeout to connect to FrostFS nodes can be set with `--connect_timeout`
and timeout to check node health during rebalance`--healthcheck_timeout`.

Also, interval to check node health can be specified by `--rebalance_interval` value.

```shell
$ frostfs-s3-gw --healthcheck_timeout 15s --connect_timeout 1m --rebalance_interval 1h
```

### Monitoring and metrics

Pprof and Prometheus are integrated into the gateway. To enable them, use `--pprof` and `--metrics` flags or
`S3_GW_PPROF_ENABLED`/`S3_GW_PROMETHEUS_ENABLED` environment variables.

## YAML file and environment variables

Example of a YAML configuration file: [yaml-example](/config/config.yaml)
Examples of environment variables: [env-example](/config/config.env).

A path to a configuration file can be specified with `--config` parameter:

```shell
$ frostfs-s3-gw --config your-config.yaml
```

### Multiple configs

You can use several config files when running application. It allows you to split configuration into parts.
For example, you can use separate yaml file for pprof and prometheus section in config (see [config examples](../config)).
You can either provide several files with repeating `--config` flag or provide path to the dir that contains all configs using `--config-dir` flag.
Also, you can combine these flags:

```shell
$ frostfs-s3-gw --config ./config/config.yaml --config /your/partial/config.yaml --config-dir ./config/dir
```

**Note:** next file in `--config` flag overwrites values from the previous one.
Files from `--config-dir` directory overwrite values from `--config` files.
So the command above run `frostfs-s3-gw` to listen on `0.0.0.0:8080` address (value from `./config/config.yaml`),
applies parameters from `/your/partial/config.yaml`,
enable pprof (value from `./config/dir/pprof.yaml`) and prometheus (value from `./config/dir/prometheus.yaml`).

### Reload on SIGHUP

Some config values can be reloaded on SIGHUP signal. 
Such parameters have special mark in tables below.

You can send SIGHUP signal to app using the following command:

```shell
$ kill -s SIGHUP <app_pid>
```

Example:

```shell
$ ./bin/frostfs-s3-gw --config config.yaml  &> s3.log &
[1] 998346

$ cat s3.log
# ...
2022-09-30T17:38:22.338+0300    info    s3-gw/app.go:371        application started     {"name": "frostfs-s3-gw", "version": "v0.24.0"}
# ...

$ kill -s SIGHUP 998346

$ cat s3.log
# ...
2022-09-30T17:38:40.909+0300    info    s3-gw/app.go:491        SIGHUP config reload completed
```

### FrostFS S3 Gateway configuration file

This section contains detailed FrostFS S3 Gateway configuration file description
including default config values and some tips to set up configurable values.

There are some custom types used for brevity:

* `duration` -- string consisting of a number and a suffix. Suffix examples include `s` (seconds), `m` (minutes), `ms` (
  milliseconds).

### Structure

| Section            | Description                                                 |
|--------------------|-------------------------------------------------------------|
| no section         | [General parameters](#general-section)                      |
| `wallet`           | [Wallet configuration](#wallet-section)                     |
| `peers`            | [Nodes configuration](#peers-section)                       |
| `placement_policy` | [Placement policy configuration](#placement_policy-section) |
| `server`           | [Server configuration](#server-section)                     |
| `logger`           | [Logger configuration](#logger-section)                     |
| `tree`             | [Tree configuration](#tree-section)                         |
| `cache`            | [Cache configuration](#cache-section)                       |
| `nats`             | [NATS configuration](#nats-section)                         |
| `cors`             | [CORS configuration](#cors-section)                         |
| `pprof`            | [Pprof configuration](#pprof-section)                       |
| `prometheus`       | [Prometheus configuration](#prometheus-section)             |
| `frostfs`          | [Parameters of requests to FrostFS](#frostfs-section)       |

### General section

```yaml
listen_domains:
   - s3dev.frostfs.devenv
   - s3dev2.frostfs.devenv

rpc_endpoint: http://morph-chain.frostfs.devenv:30333
resolve_order:
  - nns
  - dns

connect_timeout: 10s
stream_timeout: 10s
healthcheck_timeout: 15s
rebalance_interval: 60s
pool_error_threshold: 100

max_clients_count: 100
max_clients_deadline: 30s

allowed_access_key_id_prefixes: 
   - Ck9BHsgKcnwfCTUSFm6pxhoNS4cBqgN2NQ8zVgPjqZDX
   - 3stjWenX15YwYzczMr88gy3CQr4NYFBQ8P7keGzH5QFn
```

| Parameter                        | Type       | SIGHUP reload | Default value  | Description                                                                                                                                                                                                       |
|----------------------------------|------------|---------------|----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `listen_domains`                 | `[]string` |               |                | Domains to be able to use virtual-hosted-style access to bucket.                                                                                                                                                  |
| `rpc_endpoint`                   | `string`   | yes           |                | The address of the RPC host to which the gateway connects to resolve bucket names (required to use the `nns` resolver).                                                                                           |
| `resolve_order`                  | `[]string` | yes           | `[dns]`        | Order of bucket name resolvers to use. Available resolvers: `dns`, `nns`.                                                                                                                                         |                                                                                                                                                                           |
| `connect_timeout`                | `duration` |               | `10s`          | Timeout to connect to a node.                                                                                                                                                                                     |
| `stream_timeout`                 | `duration` |               | `10s`          | Timeout for individual operations in streaming RPC.                                                                                                                                                               |
| `healthcheck_timeout`            | `duration` |               | `15s`          | Timeout to check node health during rebalance.                                                                                                                                                                    |
| `rebalance_interval`             | `duration` |               | `60s`          | Interval to check node health.                                                                                                                                                                                    |
| `pool_error_threshold`           | `uint32`   |               | `100`          | The number of errors on connection after which node is considered as unhealthy.                                                                                                                                   |
| `max_clients_count`              | `int`      |               | `100`          | Limits for processing of clients' requests.                                                                                                                                                                       |
| `max_clients_deadline`           | `duration` |               | `30s`          | Deadline after which the gate sends error `RequestTimeout` to a client.                                                                                                                                           |
| `allowed_access_key_id_prefixes` | `[]string` |               |                | List of allowed `AccessKeyID` prefixes which S3 GW serve. If the parameter is omitted, all `AccessKeyID` will be accepted.                                                                                        |

### `wallet` section

```yaml
wallet:
   path: /path/to/wallet.json # Path to wallet
   passphrase: "" # Passphrase to decrypt wallet.
   address: NfgHwwTi3wHAS8aFAN243C5vGbkYDpqLHP
```

| Parameter    | Type     | Default value | Description                                                               |
|--------------|----------|---------------|---------------------------------------------------------------------------|
| `path`       | `string` |               | Path to wallet                                                            |
| `passphrase` | `string` |               | Passphrase to decrypt wallet.                                             |
| `address`    | `string` |               | Account address to get from wallet. If omitted default one will be used.  |

### `peers` section

```yaml
# Nodes configuration
# This configuration makes the gateway use the first node (node1.frostfs:8080)
# while it's healthy. Otherwise, gateway uses the second node (node2.frostfs:8080)
# for 10% of requests and the third node (node3.frostfs:8080) for 90% of requests.
# Until nodes with the same priority level are healthy
# nodes with other priority are not used.
# The lower the value, the higher the priority.
peers:
  0:
    address: node1.frostfs:8080
    priority: 1
    weight: 1
  1:
    address: node2.frostfs:8080
    priority: 2
    weight: 0.1
  2:
    address: node3.frostfs:8080
    priority: 2
    weight: 0.9
```

| Parameter  | Type     | Default value | Description                                                                                                                                             |
|------------|----------|---------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `address`  | `string` |               | Address of storage node.                                                                                                                                |
| `priority` | `int`    | `1`           | It allows to group nodes and don't switch group until all nodes with the same priority will be unhealthy. The lower the value, the higher the priority. |
| `weight`   | `float`  | `1`           | Weight of node in the group with the same priority. Distribute requests to nodes proportionally to these values.                                        |


### `placement_policy` section

```yaml
placement_policy:
  default: REP 3
  region_mapping: /path/to/mapping/rules.json
```

| Parameter        | Type     | SIGHUP reload | Default value | Description                                                                                                                                                                                                           |
|------------------|----------|---------------|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `default`        | `string` | yes           | `REP 3`       | Default policy of placing containers in FrostFS. If a user sends a request `CreateBucket` and doesn't define policy for placing of a container in FrostFS, the S3 Gateway will put the container with default policy. |
| `region_mapping` | `string` | yes           |               | Path to file that maps aws `LocationContraint` values to FrostFS placement policy. The similar to `--container-policy` flag in `frostfs-s3-authmate` util, see in [docs](./authmate.md#containers-policy)             |

File for `region_mapping` must contain something like this:

```json
{
   "rep-3": "REP 3",
   "complex": "REP 1 IN X CBF 1 SELECT 1 FROM * AS X",
   "example-json-policy": "{\"replicas\":[{\"count\":3,\"selector\":\"SelASD0\"}],\"container_backup_factor\":3,\"selectors\":[{\"name\":\"SelASD0\",\"count\":3,\"filter\":\"*\"}],\"filters\":[]}"
}
```

**Note:** on SIGHUP reload policies will be updated only if both parameters are valid. 
So if you change `default` to some valid value and set invalid path in `region_mapping` the `default` value won't be changed.

### `server` section

You can specify several listeners for server. For example, for `http` and `https`.

```yaml
server:
  - address: 0.0.0.0:8080
    tls:
      enabled: false
      cert_file: /path/to/cert
      key_file: /path/to/key
  - address: 0.0.0.0:8081
    tls:
      enabled: true
      cert_file: /path/to/another/cert
      key_file: /path/to/another/key
```

| Parameter       | Type     | SIGHUP reload | Default value  | Description                                   |
|-----------------|----------|---------------|----------------|-----------------------------------------------|
| `address`       | `string` |               | `0.0.0.0:8080` | The address that the gateway is listening on. |
| `tls.enabled`   | `bool`   |               | false          | Enable TLS or not.                            |
| `tls.cert_file` | `string` | yes           |                | Path to the TLS certificate.                  |
| `tls.key_file`  | `string` | yes           |                | Path to the key.                              |

### `logger` section

```yaml
logger:
  level: debug
```

| Parameter | Type     | SIGHUP reload | Default value | Description                                                                                        |
|-----------|----------|---------------|---------------|----------------------------------------------------------------------------------------------------|
| `level`   | `string` | yes           | `debug`       | Logging level.<br/>Possible values:  `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`. |

### `tree` section

```yaml
tree:
  service: s01.frostfs.devenv:8080
```

| Parameter | Type     | Default value | Description                                                                                                |
|-----------|----------|---------------|------------------------------------------------------------------------------------------------------------|
| `service` | `string` |               | Endpoint of the tree service. Must be provided. Can be one of the node address (from the `peers` section). |

### `cache` section

```yaml
cache:
  objects:
    lifetime: 300s
    size: 150
  list:
    lifetime: 1m
    size: 100
  names:
    lifetime: 1m
    size: 1000
  buckets:
    lifetime: 1m
    size: 500
  system:
    lifetime: 2m
    size: 1000
  accessbox:
    lifetime: 5m
    size: 10
  accesscontrol:
    lifetime: 1m
    size: 100000
```

| Parameter       | Type                              | Default value                     | Description                                                                            |
|-----------------|-----------------------------------|-----------------------------------|----------------------------------------------------------------------------------------|
| `objects`       | [Cache config](#cache-subsection) | `lifetime: 5m`<br>`size: 1000000` | Cache for objects (FrostFS headers).                                                   |
| `list`          | [Cache config](#cache-subsection) | `lifetime: 60s`<br>`size: 100000` | Cache which keeps lists of objects in buckets.                                         |
| `names`         | [Cache config](#cache-subsection) | `lifetime: 60s`<br>`size: 10000`  | Cache which contains mapping of nice name to object addresses.                         |
| `buckets`       | [Cache config](#cache-subsection) | `lifetime: 60s`<br>`size: 1000`   | Cache which contains mapping of bucket name to bucket info.                            |
| `system`        | [Cache config](#cache-subsection) | `lifetime: 5m`<br>`size: 10000`   | Cache for system objects in a bucket: bucket settings, notification configuration etc. |
| `accessbox`     | [Cache config](#cache-subsection) | `lifetime: 10m`<br>`size: 100`    | Cache which stores access box with tokens by its address.                              |
| `accesscontrol` | [Cache config](#cache-subsection) | `lifetime: 1m`<br>`size: 100000`  | Cache which stores owner to cache operation mapping.                                   |

#### `cache` subsection

```yaml
lifetime: 2m
size: 1000
```

| Parameter  | Type       | Default value    | Description                   |
|------------|------------|------------------|-------------------------------|
| `lifetime` | `duration` | depends on cache | Lifetime of entries in cache. |
| `size`     | `int`      | depends on cache | LRU cache size.               |

### `nats` section

This is an advanced section, use with caution.
You can turn on notifications about successful completions of basic operations, and the gateway will send notifications
via NATS JetStream.

1. to configure the NATS server with JetStream
2. to specify NATS parameters for the S3 GW. It's ***necessary*** to define a values of `nats.enable` or
   `S3_GW_NATS_ENABLED` as `True`
3. to configure notifications in a bucket

```yaml
nats:
  enabled: true
  endpoint: nats://localhost:4222
  timeout: 30s
  cert_file: /path/to/cert
  key_file: /path/to/key
  root_ca: /path/to/ca
```

| Parameter     | Type       | Default value | Description                                          |
|---------------|------------|---------------|------------------------------------------------------|
| `enabled`     | `bool`     | `false`       | Flag to enable the service.                          |
| `endpoint`    | `string`   |               | NATS endpoint to connect to.                         |
| `timeout`     | `duration` | `30s`         | Timeout for the object notification operation.       |
| `certificate` | `string`   |               | Path to the client certificate.                      |
| `key`         | `string`   |               | Path to the client key.                              |
| `ca`          | `string`   |               | Override root CA used to verify server certificates. |

### `cors` section

```yaml
cors:
  default_max_age: 600
```

| Parameter         | Type  | Default value | Description                                          |
|-------------------|-------|---------------|------------------------------------------------------|
| `default_max_age` | `int` | `600`         | Value of `Access-Control-Max-Age` header in seconds. |

# `pprof` section

Contains configuration for the `pprof` profiler.

```yaml
pprof:
  enabled: true
  address: localhost:8085
```

| Parameter | Type     | SIGHUP reload | Default value    | Description                             |
|-----------|----------|---------------|------------------|-----------------------------------------|
| `enabled` | `bool`   | yes           | `false`          | Flag to enable the service.             |
| `address` | `string` | yes           | `localhost:8085` | Address that service listener binds to. |

# `prometheus` section

Contains configuration for the `prometheus` metrics service.

```yaml
prometheus:
  enabled: true
  address: localhost:8086
```

| Parameter | Type     | SIGHUP reload | Default value    | Description                             |
|-----------|----------|---------------|------------------|-----------------------------------------|
| `enabled` | `bool`   | yes           | `false`          | Flag to enable the service.             |
| `address` | `string` | yes           | `localhost:8086` | Address that service listener binds to. |

# `frostfs` section

Contains parameters of requests to FrostFS. 
This value can be overridden with `X-Amz-Meta-Frostfs-Copies-Number` header for `PutObject`, `CopyObject`, `CreateMultipartUpload`.

```yaml
frostfs:
  set_copies_number: 0
```

| Parameter           | Type     | Default value | Description                                                                                                                                                                 |
|---------------------|----------|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `set_copies_number` | `uint32` | `0`           | Number of the object copies to consider PUT to FrostFS successful. <br/>Default value `0` means that object will be processed according to the container's placement policy |
