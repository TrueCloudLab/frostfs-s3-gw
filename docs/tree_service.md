# Tree service

To get objects' metadata and system information, the S3 GW makes requests to the Tree service. 
This is a service in FrostFS storage that keeps different information as a tree structure. 

Each node keeps one of the types of data as a set of **key-value pairs**:
* Bucket settings: lock configuration and versioning mode 
* Bucket tagging
* Object tagging
* Object metadata: OID, name, creation time, system metadata
* Object locking settings
* Active multipart upload info

Some data takes up a lot of memory, so we store it in FrostFS nodes as an object with payload. 
But we keep these objects' metadata in the Tree service too:
* Notification configuration
* CORS
* Metadata of parts of active multipart uploads
