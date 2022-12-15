# AWS CLI basic usage

## Configuration

### Credentials

To configure basic settings that the AWS CLI uses to interact with the Gateway, follow the steps below:

1. issue a secret with frostfs-s3-authmate tool (see [FrostFS S3 Authmate](./authmate.md))
2. execute the command
```
$ aws configure
```
after you enter this command, the AWS CLI will prompt you for four pieces of information, like in this example
(replace with your own values):
```
AWS Access Key ID [None]: 5g933dyLEkXbbAspouhPPTiyLZRg4axBW1axSPD87eVT0AiXsH4AjYy1iTJ4C1WExzjBrSobJsQFWEyKLREe5sQYM
AWS Secret Access Key [None]: 438bbd8243060e1e1c9dd4821756914a6e872ce29bf203b68f81b140ac91231c
Default region name [None]: ru 
Default output format [none]: json 
```

## Basic usage

> **_NOTE:_** To specify the IP and the port of the gate, append `--endpoint-url https://%IP:%PORT` to your commands.

### Bucket

#### Obtainment of a list of buckets 

To view the list of the buckets in the FrostFS node, to which the gateway is connected, enter the following command:
```
$ aws s3 ls 
```

#### Creation of a bucket

At this moment, the gateway supports only canned ACL and doesn't support the setting of location constraints.

To create a bucket, run the following command:
```
$ aws s3api create-bucket --bucket %BUCKET_NAME --acl %ACL
```
where `%ACL` can be represented by a hex encoded value or by keywords `public-read-write`, `private`, `public-read`. 
If the parameter is not set, the default value is `private`.

> **_NOTE:_**  Bucket creation uses async-poll approach. `BucketAlreadyOwnedByYou`
> status can occur if the AWS CLI makes multiple attempts (see details
> in [docs](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-retries.html)).
> This status means successful bucket creation after several tries. When the
> operation has not been completed in the entire wait time, timeout status outputs.
> Timeout does not always mean failure of the operation: success can be checked
> later by bucket getting/listing commands, or by repeating the creating command
> (previous success will result in the status mentioned above). It is worth clarifying
> that the final success of the bucket creation is not guaranteed after a timeout error.

#### Deletion of a bucket 

To delete a bucket, execute the following command:
```
$ aws s3api delete-bucket --bucket %BUCKET_NAME
```

### Object

#### Obtainment of a list of objects

To view the list of the objects in a bucket, run:
```
$ aws s3api list-objects --bucket %BUCKET_NAME 
```

#### Upload of a file

To upload a file into a bucket in the FrostFS network, run the following command:
```
$ aws s3api put-object --bucket %BUCKET_NAME --key %OBJECT_KEY --body  %FILEPATH
```
where %OBJECT_KEY is the filepath of an object in FrostFS

#### Upload of a dir

To upload a dir into a bucket in the FrostFS network, run the following command:

```
$ aws s3 sync %DIRPATH s3://%BUCKET_NAME 
```

#### Download of a file

To download a file from a bucket in the FrostFS Network, execute:
```
$ aws s3api get-object --bucket  %BUCKET_NAME --key %OBJECT_KEY %OUTFILE
```

where %OUTFILE is the file to store object content.

#### Deletion of a file
To delete a file:
```
$ aws s3api delete-object --bucket %BUCKET_NAME --key %FILE_NAME
```
