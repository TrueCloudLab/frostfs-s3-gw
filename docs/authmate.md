# FrostFS S3 AuthMate

Authmate is a tool to create gateway AWS credentials. AWS users
are authenticated with access key IDs and secrets, while FrostFS users are
authenticated with key pairs. To complicate things further, we have S3 gateway
that usually acts on behalf of some user, but the user doesn't necessarily want to
give their keys to the gateway.

To solve this, we use FrostFS bearer tokens that are signed by the owner (FrostFS
"user") and that can implement any kind of policy for FrostFS requests allowed
to use this token. However, tokens can't be used as AWS credentials directly. Thus,
they're stored on FrostFS as regular objects, and an access key ID is just an
address of this object while a secret is generated randomly.

Tokens are not stored on FrostFS in plaintext, they're encrypted with a set of
gateway keys. So, in order for a gateway to be able to successfully extract bearer
token, the object needs to be stored in a container available for the gateway
to read, and it needs to be encrypted with this gateway's key (among others
potentially).

1. [Generation of wallet](#generation-of-wallet)
2. [Issuance of a secret](#issuance-of-a-secret)
   1. [CLI parameters](#cli-parameters)
   2. [Bearer tokens](#bearer-tokens)
   3. [Session tokens](#session-tokens)
   4. [Containers policy](#containers-policy)
3. [Obtainment of a secret](#obtainment-of-a-secret-access-key)
4. [Generate presigned url](#generate-presigned-url)

## Generation of wallet

To generate a wallet for a gateway, run the following command:

```shell
$ ./neo-go wallet init -a -w wallet.json

Enter the name of the account > AccountTestName
Enter passphrase > 
Confirm passphrase > 

{
 	"version": "3.0",
 	"accounts": [
 		{
 			"address": "NhLQpDnerpviUWDF77j5qyjFgavCmasJ4p",
 			"key": "6PYUFyYpJ1JGyMrYV8NqeUFLKfpEVHsGGjCYtTDkjnKaSgYizRBZxVerte",
 			"label": "AccountTestName",
 			"contract": {
 				"script": "DCECXCsUZPwUyKHs6nAyyCvJ5s/vLwZkkVtWNC0zWzH8a9dBVuezJw==",
 				"parameters": [
 					{
 						"name": "parameter0",
 						"type": "Signature"
 					}
 				],
 				"deployed": false
 			},
 			"lock": false,
 			"isDefault": false
 		}
 	],
 	"scrypt": {
 		"n": 16384,
 		"r": 8,
 		"p": 8
 	},
 	"extra": {
 		"Tokens": null
 	}
 }

wallet is successfully created, the file location is wallet.json
```

To get the public key from the wallet:
```shell
$ ./bin/neo-go wallet dump-keys -w wallet.json

NhLQpDnerpviUWDF77j5qyjFgavCmasJ4p (simple signature contract):
025c2b1464fc14c8a1ecea7032c82bc9e6cfef2f0664915b56342d335b31fc6bd7
```

## Issuance of a secret

To issue a secret means to create Bearer and, optionally, Session tokens and
put them as an object into a container on the FrostFS network.

### CLI parameters

**Required parameters:**
* `--wallet` is a path to a wallet `.json` file. You can provide a passphrase to decrypt
a wallet via environment variable `AUTHMATE_WALLET_PASSPHRASE`, or you will be asked to enter a passphrase 
interactively. You can also specify an account address to use from a wallet using the `--address` parameter.
* `--peer` is an address of a FrostFS peer to connect to
* `--gate-public-key` is a public `secp256r1` 33-byte short key of a gate (use flags repeatedly for multiple gates). The tokens are encrypted
by a set of gateway keys, so you need to pass them as well.

You can issue a secret using the parameters above only. The tool will 
1. create a new container  
   1. without a friendly name
   2. with ACL `0x3c8c8cce` -- all operations are forbidden for `OTHERS` and `BEARER` user groups, except for `GET` 
   3. with policy `REP 2 IN X CBF 3 SELECT 2 FROM * AS X` 
2. put bearer and session tokens with default rules (details in [Bearer tokens](#Bearer tokens) and 
[Session tokens](#Session tokens))

E.g.:
```shell
$ frostfs-s3-authmate issue-secret --wallet wallet.json \
 --peer 192.168.130.71:8080 \
 --gate-public-key 0313b1ac3a8076e155a7e797b24f0b650cccad5941ea59d7cfd51a024a8b2a06bf\
 --gate-public-key 0317585fa8274f7afdf1fc5f2a2e7bece549d5175c4e5182e37924f30229aef967
 
 Enter password for wallet.json > 
 
{
  "access_key_id": "5g933dyLEkXbbAspouhPPTiyLZRg4axBW1axSPD87eVT0AiXsH4AjYy1iTJ4C1WExzjBrSobJsQFWEyKLREe5sQYM",
  "secret_access_key": "438bbd8243060e1e1c9dd4821756914a6e872ce29bf203b68f81b140ac91231c",
  "owner_private_key": "274fdd6e71fc6a6b8fe77bec500254115d66d6d17347d7db0880d2eb80afc72a",
  "container_id":"5g933dyLEkXbbAspouhPPTiyLZRg4axBW1axSPD87eVT"
}
```

`access_key_id` and `secret_access_key` are AWS credentials that you can use with any S3 client.

`access_key_id` consists of Base58 encoded containerID(cid) and objectID(oid) stored on the FrostFS network and containing
the secret. Format of `access_key_id`: `%cid0%oid`, where 0(zero) is a delimiter.

**Optional parameters:**
* `--container-id` - you can put the tokens into an existing container, but this way is ***not recommended***.
* `--container-friendly-name` -- name of a container with tokens, by default container will not have a friendly name
* `--container-placement-policy` -  placement policy of auth container to put the secret into. Default value is
`REP 2 IN X CBF 3 SELECT 2 FROM * AS X`
* `--lifetime`-- lifetime of tokens.  For example 50h30m (note: max time unit is an hour so to set a day you should use 
24h). Default value is `720h` (30 days). It will be ceil rounded to the nearest amount of epoch
* `--aws-cli-credentials` - path to the aws cli credentials file, where authmate will write `access_key_id` and 
`secret_access_key` to

### Bearer tokens

Creation of bearer tokens is mandatory.

Rules for a bearer token can be set via parameter `--bearer-rules` (json-string and file path allowed):
```shell
$ frostfs-s3-authmate issue-secret --wallet wallet.json \
--peer 192.168.130.71:8080 \
--gate-public-key 0313b1ac3a8076e155a7e797b24f0b650cccad5941ea59d7cfd51a024a8b2a06bf \
--bearer-rules bearer-rules.json  
```
where content of `bearer-rules.json`:
```json
{
  "records": [
    {"operation": "PUT", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "GET", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "HEAD", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "DELETE", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "SEARCH", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "GETRANGE", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
    {"operation": "GETRANGEHASH", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]}
  ]
}
```

**Note:** such rules allow all operations for all users (the same behavior when records are empty). 
To restrict access you MUST provide records with `DENY` action. That's why we recommend always place such records
at the end of records (see default rules below) to prevent undesirable access violation.
Since the rules are applied from top to bottom, they do not override what was previously allowed.

If bearer rules are not set, a token will be auto-generated with a value:
```json
{
    "version": {
       "major": 2,
       "minor": 11
    },
    "containerID": {
       "value": null
    },
    "records": [
       {"operation": "GET", "action": "ALLOW", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},

       {"operation": "GET", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "HEAD", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "PUT", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "DELETE", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "SEARCH", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "GETRANGE", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]},
       {"operation": "GETRANGEHASH", "action": "DENY", "filters": [], "targets": [{"role": "OTHERS", "keys": []}]}
    ]
}
```

### Session tokens

With a session token, there are 3 options: 
1. append `--session-tokens` parameter with your custom rules in json format (as a string or file path). E.g.:
```shell
$ frostfs-s3-authmate issue-secret --wallet wallet.json \
--peer 192.168.130.71:8080 \
--gate-public-key 0313b1ac3a8076e155a7e797b24f0b650cccad5941ea59d7cfd51a024a8b2a06bf \
--session-tokens session.json
```
where content of `session.json`:
```json
[
  {
    "verb": "PUT",
    "containerID": null
  },
  {
    "verb": "DELETE",
    "containerID": null
  },
  {
    "verb": "SETEACL",
    "containerID": null
  }
]
```

Available `verb` values: `PUT`, `DELETE`, `SETEACL`.

If `containerID` is `null` or omitted, then session token rule will be applied
to all containers. Otherwise, specify `containerID` value in human-redabale
format (base58 encoded string).

> **_NB!_** To create buckets in FrostFS it's necessary to have session tokens with `PUT` and `SETEACL` permissions, that's why 
the authmate creates a `SETEACL` session token automatically in case when a user specified the token rule with `PUT` and 
forgot about the rule with `SETEACL`.

2. append `--session-tokens` parameter with the value `none` -- no session token will be created
3. skip the parameter, and `authmate` will create session tokens with default rules (the same as in `session.json`
in example above)

### Containers policy

Rules for mapping of `LocationConstraint` ([aws spec](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html#API_CreateBucket_RequestBody)) 
to `PlacementPolicy` ([frostfs spec](https://github.com/TrueCloudLab/frostfs-spec/blob/master/01-arch/02-policy.md)) 
can be set via parameter `--container-policy` (json-string and file path allowed):
```json
{
  "rep-3": "REP 3",
  "complex": "REP 1 IN X CBF 1 SELECT 1 FROM * AS X",
  "example-json-policy": "{\"replicas\":[{\"count\":3,\"selector\":\"SelASD0\"}],\"container_backup_factor\":3,\"selectors\":[{\"name\":\"SelASD0\",\"count\":3,\"filter\":\"*\"}],\"filters\":[]}"
}
```

## Obtainment of a secret access key

You can get a secret access key associated with an access key ID by obtaining a
secret stored on the FrostFS network. Here is an example of providing one password (for `wallet.json`) via env variable 
and the other (for `gate-wallet.json`) interactively:

```shell
$ AUTHMATE_WALLET_PASSPHRASE=some-pwd \
frostfs-s3-authmate obtain-secret --wallet wallet.json \
--peer 192.168.130.71:8080 \
--gate-wallet gate-wallet.json \
--access-key-id 5g933dyLEkXbbAspouhPPTiyLZRg4axBW1axSPD87eVT0AiXsH4AjYy1iTJ4C1WExzjBrSobJsQFWEyKLREe5sQYM

Enter password for gate-wallet.json >
{
  "secret_access_key": "438bbd8243060e1e1c9dd4821756914a6e872ce29bf203b68f81b140ac91231c"
}
```


## Generate presigned URL

You can generate [presigned url](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html) 
using AWS credentials from `~/.aws/credentials` (you can specify profile using the `--profile` flag) 
with the following command:

```shell
$ frostfs-s3-authmate generate-presigned-url --endpoint http://localhost:8084 \
  --method get --bucket presigned --object obj --lifetime 30s
  
{
  "URL": "http://localhost:8084/presigned/obj?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=6UpmiuYspPLMWfyhEKYmZQSsTGkFLS5MhQVdsda3fhz908Hw9eo9urTmaJtfvHMHUpY8SWAptk61bns2Js8f1M5tZ%2F20220615%2Fus-east-2%2Fs3%2Faws4_request&X-Amz-Date=20220615T084203Z&X-Amz-Expires=30&X-Amz-SignedHeaders=host&X-Amz-Signature=47f74d4b84566708a17dded05cce732690745f141235215104ad051e265e3c59"
}
```

You can also provide credential explicitly:

```shell
$ frostfs-s3-authmate generate-presigned-url --endpoint http://localhost:8084 \
  --method put --bucket presigned --object obj --lifetime 12h \
  --region ru --aws-secret-access-key c2d65ef2980f03f4f495bdebedeeae760496697880d61d106bb9a4e5cd2e0607 \
  --aws-access-key-id ETaA2CadPcA7bAkLsML2PbTudXY8uRt2PDjCCwkvRv9s0FDCxWDXYc1SA1vKv8KbyCNsLY2AmAjJ92Vz5rgvsFCy

{
  "URL": "http://localhost:8084/presigned/obj?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ETaA2CadPcA7bAkLsML2PbTudXY8uRt2PDjCCwkvRv9s0FDCxWDXYc1SA1vKv8KbyCNsLY2AmAjJ92Vz5rgvsFCy%2F20220615%2Fru%2Fs3%2Faws4_request&X-Amz-Date=20220615T084455Z&X-Amz-Expires=43200&X-Amz-SignedHeaders=host&X-Amz-Signature=ac92aecc3787eeed03922ea82af082091c806f1bb58a7101602ed21369815d04"
}
```

### AWS CLI
You can also can get the presigned URL (only for GET) using aws cli v2:

```shell
$ aws s3 --endpoint http://localhost:8084 presign s3://pregigned/obj 

http://localhost:8084/pregigned/obj?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=6UpmiuYspPLMWfyhEKYmZQSsTGkFLS5MhQVdsda3fhz908Hw9eo9urTmaJtfvHMHUpY8SWAptk61bns2Js8f1M5tZ%2F20220615%2Fus-east-2%2Fs3%2Faws4_request&X-Amz-Date=20220615T072348Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=b82c13952534b1bba699a718f2d42d135c2833a1e64030d4ce0e198af46551d4
```
