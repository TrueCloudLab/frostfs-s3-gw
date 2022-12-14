package handler

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/creds/accessbox"
	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/TrueCloudLab/frostfs-sdk-go/object"
	oid "github.com/TrueCloudLab/frostfs-sdk-go/object/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/stretchr/testify/require"
)

func TestTableToAst(t *testing.T) {
	b := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, b)
	require.NoError(t, err)
	var id oid.ID
	id.SetSHA256(sha256.Sum256(b))

	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	table := new(eacl.Table)
	record := eacl.NewRecord()
	record.SetAction(eacl.ActionAllow)
	record.SetOperation(eacl.OperationGet)
	eacl.AddFormedTarget(record, eacl.RoleOthers)
	table.AddRecord(record)
	record2 := eacl.NewRecord()
	record2.SetAction(eacl.ActionDeny)
	record2.SetOperation(eacl.OperationPut)
	// Unknown role is used, because it is ignored when keys are set
	eacl.AddFormedTarget(record2, eacl.RoleUnknown, *(*ecdsa.PublicKey)(key.PublicKey()), *((*ecdsa.PublicKey)(key2.PublicKey())))
	record2.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, "objectName")
	record2.AddObjectIDFilter(eacl.MatchStringEqual, id)
	table.AddRecord(record2)

	expectedAst := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{Bucket: "bucketName"},
				Operations: []*astOperation{{
					Op:     eacl.OperationGet,
					Action: eacl.ActionAllow,
				}}},
			{
				resourceInfo: resourceInfo{
					Bucket:  "bucketName",
					Object:  "objectName",
					Version: id.EncodeToString(),
				},
				Operations: []*astOperation{{
					Users: []string{
						hex.EncodeToString(key.PublicKey().Bytes()),
						hex.EncodeToString(key2.PublicKey().Bytes()),
					},
					Op:     eacl.OperationPut,
					Action: eacl.ActionDeny,
				}}},
		},
	}

	actualAst := tableToAst(table, expectedAst.Resources[0].Bucket)

	if actualAst.Resources[0].Name() == expectedAst.Resources[0].Name() {
		require.Equal(t, expectedAst, actualAst)
	} else {
		require.Equal(t, len(expectedAst.Resources), len(actualAst.Resources))
		require.Equal(t, expectedAst.Resources[0], actualAst.Resources[1])
		require.Equal(t, expectedAst.Resources[1], actualAst.Resources[0])
	}
}

func TestPolicyToAst(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	policy := &bucketPolicy{
		Statement: []statement{
			{
				Effect:    "Allow",
				Principal: principal{AWS: allUsersWildcard},
				Action:    []string{"s3:PutObject"},
				Resource:  []string{"arn:aws:s3:::bucketName"},
			},
			{
				Effect: "Deny",
				Principal: principal{
					CanonicalUser: hex.EncodeToString(key.PublicKey().Bytes()),
				},
				Action:   []string{"s3:GetObject"},
				Resource: []string{"arn:aws:s3:::bucketName/object"},
			}},
	}
	policy.Bucket = "bucketName"

	expectedAst := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucketName",
				},
				Operations: []*astOperation{{
					Op:     eacl.OperationPut,
					Action: eacl.ActionAllow,
				}},
			},
			{
				resourceInfo: resourceInfo{
					Bucket: "bucketName",
					Object: "object",
				},
				Operations: getReadOps(key, false, eacl.ActionDeny),
			},
		},
	}

	actualAst, err := policyToAst(policy)
	require.NoError(t, err)

	if actualAst.Resources[0].Name() == expectedAst.Resources[0].Name() {
		require.Equal(t, expectedAst, actualAst)
	} else {
		require.Equal(t, len(expectedAst.Resources), len(actualAst.Resources))
		require.Equal(t, expectedAst.Resources[0], actualAst.Resources[1])
		require.Equal(t, expectedAst.Resources[1], actualAst.Resources[0])
	}
}

func getReadOps(key *keys.PrivateKey, groupGrantee bool, action eacl.Action) []*astOperation {
	var (
		result []*astOperation
		users  []string
	)
	if !groupGrantee {
		users = append(users, hex.EncodeToString(key.PublicKey().Bytes()))
	}

	for _, op := range readOps {
		result = append(result, &astOperation{
			Users:  users,
			Op:     op,
			Action: action,
		})
	}

	return result
}

func TestMergeAstUnModified(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	child := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Users:  []string{hex.EncodeToString(key.PublicKey().Bytes())},
					Op:     eacl.OperationPut,
					Action: eacl.ActionDeny,
				}},
			},
		},
	}

	parent := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
				},
				Operations: []*astOperation{{
					Op:     eacl.OperationGet,
					Action: eacl.ActionAllow,
				}},
			},
			child.Resources[0],
		},
	}

	result, updated := mergeAst(parent, child)
	require.False(t, updated)
	require.Equal(t, parent, result)
}

func TestMergeAstModified(t *testing.T) {
	child := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Op:     eacl.OperationPut,
					Action: eacl.ActionDeny,
				}, {
					Users:  []string{"user2"},
					Op:     eacl.OperationGet,
					Action: eacl.ActionDeny,
				}},
			},
		},
	}

	parent := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Users:  []string{"user1"},
					Op:     eacl.OperationGet,
					Action: eacl.ActionDeny,
				}},
			},
		},
	}

	expected := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{
					child.Resources[0].Operations[0],
					{
						Users:  []string{"user1", "user2"},
						Op:     eacl.OperationGet,
						Action: eacl.ActionDeny,
					},
				},
			},
		},
	}

	actual, updated := mergeAst(parent, child)
	require.True(t, updated)
	require.Equal(t, expected, actual)
}

func TestMergeAppended(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	users := []string{hex.EncodeToString(key.PublicKey().Bytes())}

	parent := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationGet,
						Action: eacl.ActionDeny,
					},
					{
						Op:     eacl.OperationPut,
						Action: eacl.ActionDeny,
					},
					{
						Op:     eacl.OperationDelete,
						Action: eacl.ActionDeny,
					},
				},
			},
		},
	}

	child := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
				},
			},
		},
	}

	expected := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationGet,
						Action: eacl.ActionDeny,
					},
					{
						Op:     eacl.OperationPut,
						Action: eacl.ActionDeny,
					},
					{
						Op:     eacl.OperationDelete,
						Action: eacl.ActionDeny,
					},
				},
			},
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Users:  users,
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationDelete,
						Action: eacl.ActionAllow,
					},
				},
			},
		},
	}
	actual, updated := mergeAst(parent, child)
	require.True(t, updated)
	require.Equal(t, expected, actual)
}

func TestOrder(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	users := []string{hex.EncodeToString(key.PublicKey().Bytes())}
	targetUser := eacl.NewTarget()
	targetUser.SetBinaryKeys([][]byte{key.PublicKey().Bytes()})
	targetOther := eacl.NewTarget()
	targetOther.SetRole(eacl.RoleOthers)
	bucketName := "bucket"
	objectName := "objectName"

	expectedAst := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: bucketName,
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationGet,
						Action: eacl.ActionDeny,
					},
				},
			},
			{
				resourceInfo: resourceInfo{
					Bucket: bucketName,
					Object: objectName,
				},
				Operations: []*astOperation{
					{
						Users:  users,
						Op:     eacl.OperationPut,
						Action: eacl.ActionAllow,
					},
					{
						Op:     eacl.OperationPut,
						Action: eacl.ActionDeny,
					},
				},
			},
		},
	}
	bucketServiceRec := &ServiceRecord{Resource: expectedAst.Resources[0].Name(), GroupRecordsLength: 2}
	bucketUsersGetRec := eacl.NewRecord()
	bucketUsersGetRec.SetOperation(eacl.OperationGet)
	bucketUsersGetRec.SetAction(eacl.ActionAllow)
	bucketUsersGetRec.SetTargets(*targetUser)
	bucketOtherGetRec := eacl.NewRecord()
	bucketOtherGetRec.SetOperation(eacl.OperationGet)
	bucketOtherGetRec.SetAction(eacl.ActionDeny)
	bucketOtherGetRec.SetTargets(*targetOther)
	objectServiceRec := &ServiceRecord{Resource: expectedAst.Resources[1].Name(), GroupRecordsLength: 2}
	objectUsersPutRec := eacl.NewRecord()
	objectUsersPutRec.SetOperation(eacl.OperationPut)
	objectUsersPutRec.SetAction(eacl.ActionAllow)
	objectUsersPutRec.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, objectName)
	objectUsersPutRec.SetTargets(*targetUser)
	objectOtherPutRec := eacl.NewRecord()
	objectOtherPutRec.SetOperation(eacl.OperationPut)
	objectOtherPutRec.SetAction(eacl.ActionDeny)
	objectOtherPutRec.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, objectName)
	objectOtherPutRec.SetTargets(*targetOther)

	expectedEacl := eacl.NewTable()
	expectedEacl.AddRecord(objectServiceRec.ToEACLRecord())
	expectedEacl.AddRecord(objectOtherPutRec)
	expectedEacl.AddRecord(objectUsersPutRec)
	expectedEacl.AddRecord(bucketServiceRec.ToEACLRecord())
	expectedEacl.AddRecord(bucketOtherGetRec)
	expectedEacl.AddRecord(bucketUsersGetRec)

	t.Run("astToTable order and vice versa", func(t *testing.T) {
		actualEacl, err := astToTable(expectedAst)
		require.NoError(t, err)
		require.Equal(t, expectedEacl, actualEacl)

		actualAst := tableToAst(actualEacl, bucketName)
		require.Equal(t, expectedAst, actualAst)
	})

	t.Run("tableToAst order and vice versa", func(t *testing.T) {
		actualAst := tableToAst(expectedEacl, bucketName)
		require.Equal(t, expectedAst, actualAst)

		actualEacl, err := astToTable(actualAst)
		require.NoError(t, err)
		require.Equal(t, expectedEacl, actualEacl)
	})

	t.Run("append a resource", func(t *testing.T) {
		childName := "child"
		child := &ast{Resources: []*astResource{{
			resourceInfo: resourceInfo{
				Bucket: bucketName,
				Object: childName,
			},
			Operations: []*astOperation{{Op: eacl.OperationDelete, Action: eacl.ActionDeny}}}},
		}

		childRecord := eacl.NewRecord()
		childRecord.SetOperation(eacl.OperationDelete)
		childRecord.SetAction(eacl.ActionDeny)
		childRecord.SetTargets(*targetOther)
		childRecord.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, childName)

		mergedAst, updated := mergeAst(expectedAst, child)
		require.True(t, updated)

		mergedEacl, err := astToTable(mergedAst)
		require.NoError(t, err)

		require.Equal(t, *childRecord, mergedEacl.Records()[1])
	})
}

func TestMergeAstModifiedConflict(t *testing.T) {
	child := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Users:  []string{"user1"},
					Op:     eacl.OperationPut,
					Action: eacl.ActionDeny,
				}, {
					Users:  []string{"user3"},
					Op:     eacl.OperationGet,
					Action: eacl.ActionAllow,
				}},
			},
		},
	}

	parent := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Users:  []string{"user1"},
					Op:     eacl.OperationPut,
					Action: eacl.ActionAllow,
				}, {
					Users:  []string{"user2"},
					Op:     eacl.OperationPut,
					Action: eacl.ActionDeny,
				}, {
					Users:  []string{"user3"},
					Op:     eacl.OperationGet,
					Action: eacl.ActionDeny,
				}},
			},
		},
	}

	expected := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucket",
					Object: "objectName",
				},
				Operations: []*astOperation{
					{
						Users:  []string{"user2", "user1"},
						Op:     eacl.OperationPut,
						Action: eacl.ActionDeny,
					}, {
						Users:  []string{"user3"},
						Op:     eacl.OperationGet,
						Action: eacl.ActionAllow,
					},
				},
			},
		},
	}

	actual, updated := mergeAst(parent, child)
	require.True(t, updated)
	require.Equal(t, expected, actual)
}

func TestAstToTable(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	ast := &ast{
		Resources: []*astResource{
			{
				resourceInfo: resourceInfo{
					Bucket: "bucketName",
				},
				Operations: []*astOperation{{
					Users:  []string{hex.EncodeToString(key.PublicKey().Bytes())},
					Op:     eacl.OperationPut,
					Action: eacl.ActionAllow,
				}},
			},
			{
				resourceInfo: resourceInfo{
					Bucket: "bucketName",
					Object: "objectName",
				},
				Operations: []*astOperation{{
					Op:     eacl.OperationGet,
					Action: eacl.ActionDeny,
				}},
			},
		},
	}

	expectedTable := eacl.NewTable()
	serviceRec1 := &ServiceRecord{Resource: ast.Resources[0].Name(), GroupRecordsLength: 1}
	record1 := eacl.NewRecord()
	record1.SetAction(eacl.ActionAllow)
	record1.SetOperation(eacl.OperationPut)
	// Unknown role is used, because it is ignored when keys are set
	eacl.AddFormedTarget(record1, eacl.RoleUnknown, *(*ecdsa.PublicKey)(key.PublicKey()))

	serviceRec2 := &ServiceRecord{Resource: ast.Resources[1].Name(), GroupRecordsLength: 1}
	record2 := eacl.NewRecord()
	record2.SetAction(eacl.ActionDeny)
	record2.SetOperation(eacl.OperationGet)
	eacl.AddFormedTarget(record2, eacl.RoleOthers)
	record2.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, "objectName")

	expectedTable.AddRecord(serviceRec2.ToEACLRecord())
	expectedTable.AddRecord(record2)
	expectedTable.AddRecord(serviceRec1.ToEACLRecord())
	expectedTable.AddRecord(record1)

	actualTable, err := astToTable(ast)
	require.NoError(t, err)
	require.Equal(t, expectedTable, actualTable)
}

func TestRemoveUsers(t *testing.T) {
	resource := &astResource{
		resourceInfo: resourceInfo{
			Bucket: "bucket",
		},
		Operations: []*astOperation{{
			Users:  []string{"user1", "user3", "user4"},
			Op:     eacl.OperationPut,
			Action: eacl.ActionAllow,
		},
			{
				Users:  []string{"user5"},
				Op:     eacl.OperationGet,
				Action: eacl.ActionDeny,
			},
		},
	}

	op1 := &astOperation{
		Op:     eacl.OperationPut,
		Action: eacl.ActionAllow,
	}
	op2 := &astOperation{
		Op:     eacl.OperationGet,
		Action: eacl.ActionDeny,
	}

	removeUsers(resource, op1, []string{"user1", "user2", "user4"}) // modify astOperation
	removeUsers(resource, op2, []string{"user5"})                   // remove astOperation

	require.Equal(t, len(resource.Operations), 1)
	require.Equal(t, []string{"user3"}, resource.Operations[0].Users)
}

func TestBucketAclToPolicy(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	id2 := hex.EncodeToString(key2.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				URI:  allUsersGroup,
				Type: acpGroup,
			},
			Permission: aclRead,
		}, {
			Grantee: &Grantee{
				ID:   id2,
				Type: acpCanonicalUser,
			},
			Permission: aclWrite,
		}},
	}

	resInfo := &resourceInfo{
		Bucket: "bucketName",
	}

	expectedPolicy := &bucketPolicy{
		Bucket: resInfo.Bucket,
		Statement: []statement{
			{
				Effect: "Allow",
				Principal: principal{
					CanonicalUser: id,
				},
				Action:   []string{"s3:ListBucket", "s3:ListBucketVersions", "s3:ListBucketMultipartUploads", "s3:PutObject", "s3:DeleteObject"},
				Resource: []string{arnAwsPrefix + resInfo.Name()},
			},
			{
				Effect:    "Allow",
				Principal: principal{AWS: allUsersWildcard},
				Action:    []string{"s3:ListBucket", "s3:ListBucketVersions", "s3:ListBucketMultipartUploads"},
				Resource:  []string{arnAwsPrefix + resInfo.Name()},
			},
			{
				Effect: "Allow",
				Principal: principal{
					CanonicalUser: id2,
				},
				Action:   []string{"s3:PutObject", "s3:DeleteObject"},
				Resource: []string{arnAwsPrefix + resInfo.Name()},
			},
		},
	}

	actualPolicy, err := aclToPolicy(acl, resInfo)
	require.NoError(t, err)
	require.Equal(t, expectedPolicy, actualPolicy)
}

func TestObjectAclToPolicy(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	id2 := hex.EncodeToString(key2.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				ID:   id,
				Type: acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				ID:   id2,
				Type: acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				URI:  allUsersGroup,
				Type: acpGroup,
			},
			Permission: aclRead,
		}},
	}

	resInfo := &resourceInfo{
		Bucket: "bucketName",
		Object: "object",
	}

	expectedPolicy := &bucketPolicy{
		Bucket: resInfo.Bucket,
		Statement: []statement{
			{
				Effect: "Allow",
				Principal: principal{
					CanonicalUser: id,
				},
				Action:   []string{"s3:GetObject", "s3:GetObjectVersion"},
				Resource: []string{arnAwsPrefix + resInfo.Name()},
			},
			{
				Effect: "Allow",
				Principal: principal{
					CanonicalUser: id2,
				},
				Action:   []string{"s3:GetObject", "s3:GetObjectVersion"},
				Resource: []string{arnAwsPrefix + resInfo.Name()},
			},
			{
				Effect:    "Allow",
				Principal: principal{AWS: allUsersWildcard},
				Action:    []string{"s3:GetObject", "s3:GetObjectVersion"},
				Resource:  []string{arnAwsPrefix + resInfo.Name()},
			},
		},
	}

	actualPolicy, err := aclToPolicy(acl, resInfo)
	require.NoError(t, err)
	require.Equal(t, expectedPolicy, actualPolicy)
}

func TestObjectWithVersionAclToTable(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	id := hex.EncodeToString(key.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				ID:   id,
				Type: acpCanonicalUser,
			},
			Permission: aclFullControl,
		}},
	}

	resInfoObject := &resourceInfo{
		Bucket: "bucketName",
		Object: "object",
	}
	expectedTable := allowedTableForPrivateObject(t, key, resInfoObject)
	actualTable := tableFromACL(t, acl, resInfoObject)
	checkTables(t, expectedTable, actualTable)

	resInfoObjectVersion := &resourceInfo{
		Bucket:  "bucketName",
		Object:  "objectVersion",
		Version: "Gfrct4Afhio8pCGCCKVNTf1kyexQjMBeaUfvDtQCkAvg",
	}
	expectedTable = allowedTableForPrivateObject(t, key, resInfoObjectVersion)
	actualTable = tableFromACL(t, acl, resInfoObjectVersion)
	checkTables(t, expectedTable, actualTable)
}

func allowedTableForPrivateObject(t *testing.T, key *keys.PrivateKey, resInfo *resourceInfo) *eacl.Table {
	var isVersion bool
	var objID oid.ID
	if resInfo.Version != "" {
		isVersion = true
		err := objID.DecodeString(resInfo.Version)
		require.NoError(t, err)
	}

	expectedTable := eacl.NewTable()
	serviceRec := &ServiceRecord{Resource: resInfo.Name(), GroupRecordsLength: len(readOps) * 2}
	expectedTable.AddRecord(serviceRec.ToEACLRecord())

	for i := len(readOps) - 1; i >= 0; i-- {
		op := readOps[i]
		record := getAllowRecord(op, key.PublicKey())
		if isVersion {
			record.AddObjectIDFilter(eacl.MatchStringEqual, objID)
		} else {
			record.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, resInfo.Object)
		}
		expectedTable.AddRecord(record)
	}
	for i := len(readOps) - 1; i >= 0; i-- {
		op := readOps[i]
		record := getOthersRecord(op, eacl.ActionDeny)
		if isVersion {
			record.AddObjectIDFilter(eacl.MatchStringEqual, objID)
		} else {
			record.AddObjectAttributeFilter(eacl.MatchStringEqual, object.AttributeFilePath, resInfo.Object)
		}
		expectedTable.AddRecord(record)
	}

	return expectedTable
}

func tableFromACL(t *testing.T, acl *AccessControlPolicy, resInfo *resourceInfo) *eacl.Table {
	actualPolicy, err := aclToPolicy(acl, resInfo)
	require.NoError(t, err)
	actualAst, err := policyToAst(actualPolicy)
	require.NoError(t, err)
	actualTable, err := astToTable(actualAst)
	require.NoError(t, err)
	return actualTable
}

func checkTables(t *testing.T, expectedTable, actualTable *eacl.Table) {
	require.Equal(t, len(expectedTable.Records()), len(actualTable.Records()), "different number of records")
	for i, record := range expectedTable.Records() {
		actRecord := actualTable.Records()[i]

		require.Equal(t, len(record.Targets()), len(actRecord.Targets()), "different number of targets")
		for j, target := range record.Targets() {
			actTarget := actRecord.Targets()[j]

			expected := fmt.Sprintf("%s %v", target.Role().String(), target.BinaryKeys())
			actual := fmt.Sprintf("%s %v", actTarget.Role().String(), actTarget.BinaryKeys())
			require.Equalf(t, target, actTarget, "want: '%s'\ngot: '%s'", expected, actual)
		}

		require.Equal(t, len(record.Filters()), len(actRecord.Filters()), "different number of filters")
		for j, filter := range record.Filters() {
			actFilter := actRecord.Filters()[j]

			expected := fmt.Sprintf("%s:%s %s %s", filter.From().String(), filter.Key(), filter.Matcher().String(), filter.Value())
			actual := fmt.Sprintf("%s:%s %s %s", actFilter.From().String(), actFilter.Key(), actFilter.Matcher().String(), actFilter.Value())
			require.Equalf(t, filter, actFilter, "want: '%s'\ngot: '%s'", expected, actual)
		}

		require.Equal(t, record.Action().String(), actRecord.Action().String())
		require.Equal(t, record.Operation().String(), actRecord.Operation().String())
	}
}

func TestParseCannedACLHeaders(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	address := key.PublicKey().Address()

	req := &http.Request{
		Header: map[string][]string{
			api.AmzACL: {basicACLReadOnly},
		},
	}

	expectedACL := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: address,
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				ID:          id,
				DisplayName: address,
				Type:        acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				URI:  allUsersGroup,
				Type: acpGroup,
			},
			Permission: aclRead,
		}},
	}

	actualACL, err := parseACLHeaders(req.Header, key.PublicKey())
	require.NoError(t, err)
	require.Equal(t, expectedACL, actualACL)
}

func TestParseACLHeaders(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	address := key.PublicKey().Address()

	req := &http.Request{
		Header: map[string][]string{
			api.AmzGrantFullControl: {"id=\"user1\""},
			api.AmzGrantRead:        {"uri=\"" + allUsersGroup + "\", id=\"user2\""},
			api.AmzGrantWrite:       {"id=\"user2\", id=\"user3\""},
		},
	}

	expectedACL := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: address,
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				ID:          id,
				DisplayName: address,
				Type:        acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				ID:   "user1",
				Type: acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				URI:  allUsersGroup,
				Type: acpGroup,
			},
			Permission: aclRead,
		}, {
			Grantee: &Grantee{
				ID:   "user2",
				Type: acpCanonicalUser,
			},
			Permission: aclRead,
		}, {
			Grantee: &Grantee{
				ID:   "user2",
				Type: acpCanonicalUser,
			},
			Permission: aclWrite,
		}, {
			Grantee: &Grantee{
				ID:   "user3",
				Type: acpCanonicalUser,
			},
			Permission: aclWrite,
		}},
	}

	actualACL, err := parseACLHeaders(req.Header, key.PublicKey())
	require.NoError(t, err)
	require.Equal(t, expectedACL, actualACL)
}

func TestAddGranteeError(t *testing.T) {
	headers := map[string][]string{
		api.AmzGrantFullControl: {"i=\"user1\""},
		api.AmzGrantRead:        {"uri, id=\"user2\""},
		api.AmzGrantWrite:       {"emailAddress=\"user2\""},
		"unknown header":        {"something"},
	}

	expectedList := []*Grant{{
		Permission: "predefined",
	}}

	actualList, err := addGrantees(expectedList, headers, "unknown header1")
	require.NoError(t, err)
	require.Equal(t, expectedList, actualList)

	actualList, err = addGrantees(expectedList, headers, "unknown header")
	require.Error(t, err)
	require.Nil(t, actualList)

	actualList, err = addGrantees(expectedList, headers, api.AmzGrantFullControl)
	require.Error(t, err)
	require.Nil(t, actualList)

	actualList, err = addGrantees(expectedList, headers, api.AmzGrantRead)
	require.Error(t, err)
	require.Nil(t, actualList)

	actualList, err = addGrantees(expectedList, headers, api.AmzGrantWrite)
	require.Error(t, err)
	require.Nil(t, actualList)
}

func TestBucketAclToTable(t *testing.T) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	id2 := hex.EncodeToString(key2.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				URI:  allUsersGroup,
				Type: acpGroup,
			},
			Permission: aclRead,
		}, {
			Grantee: &Grantee{
				ID:   id2,
				Type: acpCanonicalUser,
			},
			Permission: aclWrite,
		}},
	}

	expectedTable := new(eacl.Table)
	for _, op := range readOps {
		expectedTable.AddRecord(getOthersRecord(op, eacl.ActionAllow))
	}
	for _, op := range writeOps {
		expectedTable.AddRecord(getAllowRecord(op, key2.PublicKey()))
	}
	for _, op := range fullOps {
		expectedTable.AddRecord(getAllowRecord(op, key.PublicKey()))
	}
	for _, op := range fullOps {
		expectedTable.AddRecord(getOthersRecord(op, eacl.ActionDeny))
	}
	resInfo := &resourceInfo{
		Bucket: "bucketName",
	}

	actualTable, err := bucketACLToTable(acl, resInfo)
	require.NoError(t, err)
	require.Equal(t, expectedTable.Records(), actualTable.Records())
}

func TestObjectAclToAst(t *testing.T) {
	b := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, b)
	require.NoError(t, err)
	var objID oid.ID
	objID.SetSHA256(sha256.Sum256(b))

	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	id2 := hex.EncodeToString(key2.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{{
			Grantee: &Grantee{
				ID:   id,
				Type: acpCanonicalUser,
			},
			Permission: aclFullControl,
		}, {
			Grantee: &Grantee{
				ID:   id2,
				Type: acpCanonicalUser,
			},
			Permission: aclRead,
		},
		},
	}

	resInfo := &resourceInfo{
		Bucket:  "bucketName",
		Object:  "object",
		Version: objID.EncodeToString(),
	}

	var operations []*astOperation
	for _, op := range readOps {
		astOp := &astOperation{Users: []string{
			hex.EncodeToString(key.PublicKey().Bytes()),
			hex.EncodeToString(key2.PublicKey().Bytes()),
		},
			Op:     op,
			Action: eacl.ActionAllow,
		}
		operations = append(operations, astOp)
	}

	expectedAst := &ast{
		Resources: []*astResource{
			{
				resourceInfo: *resInfo,
				Operations:   operations,
			},
		},
	}

	actualAst, err := aclToAst(acl, resInfo)
	require.NoError(t, err)
	require.Equal(t, expectedAst, actualAst)
}

func TestBucketAclToAst(t *testing.T) {
	b := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, b)
	require.NoError(t, err)
	var objID oid.ID
	objID.SetSHA256(sha256.Sum256(b))

	key, err := keys.NewPrivateKey()
	require.NoError(t, err)
	key2, err := keys.NewPrivateKey()
	require.NoError(t, err)

	id := hex.EncodeToString(key.PublicKey().Bytes())
	id2 := hex.EncodeToString(key2.PublicKey().Bytes())

	acl := &AccessControlPolicy{
		Owner: Owner{
			ID:          id,
			DisplayName: "user1",
		},
		AccessControlList: []*Grant{
			{
				Grantee: &Grantee{
					ID:   id2,
					Type: acpCanonicalUser,
				},
				Permission: aclWrite,
			}, {
				Grantee: &Grantee{
					URI:  allUsersGroup,
					Type: acpGroup,
				},
				Permission: aclRead,
			},
		},
	}

	var operations []*astOperation
	for _, op := range readOps {
		astOp := &astOperation{Users: []string{
			hex.EncodeToString(key.PublicKey().Bytes()),
		},
			Op:     op,
			Action: eacl.ActionAllow,
		}
		operations = append(operations, astOp)
	}
	for _, op := range writeOps {
		astOp := &astOperation{Users: []string{
			hex.EncodeToString(key.PublicKey().Bytes()),
			hex.EncodeToString(key2.PublicKey().Bytes()),
		},
			Op:     op,
			Action: eacl.ActionAllow,
		}
		operations = append(operations, astOp)
	}
	for _, op := range readOps {
		astOp := &astOperation{
			Op:     op,
			Action: eacl.ActionAllow,
		}
		operations = append(operations, astOp)
	}

	resInfo := &resourceInfo{Bucket: "bucketName"}

	expectedAst := &ast{
		Resources: []*astResource{
			{
				resourceInfo: *resInfo,
				Operations:   operations,
			},
		},
	}

	actualAst, err := aclToAst(acl, resInfo)
	require.NoError(t, err)
	require.Equal(t, expectedAst, actualAst)
}

func TestPutBucketACL(t *testing.T) {
	tc := prepareHandlerContext(t)
	bktName := "bucket-for-acl"

	box, _ := createAccessBox(t)
	bktInfo := createBucket(t, tc, bktName, box)

	header := map[string]string{api.AmzACL: "public-read"}
	putBucketACL(t, tc, bktName, box, header)

	header = map[string]string{api.AmzACL: "private"}
	putBucketACL(t, tc, bktName, box, header)
	checkLastRecords(t, tc, bktInfo, eacl.ActionDeny)
}

func TestBucketPolicy(t *testing.T) {
	hc := prepareHandlerContext(t)
	bktName := "bucket-for-policy"

	box, key := createAccessBox(t)
	createBucket(t, hc, bktName, box)

	bktPolicy := getBucketPolicy(hc, bktName)
	for _, st := range bktPolicy.Statement {
		if st.Effect == "Allow" {
			require.Equal(t, hex.EncodeToString(key.PublicKey().Bytes()), st.Principal.CanonicalUser)
			require.Equal(t, []string{arnAwsPrefix + bktName}, st.Resource)
		} else {
			require.Equal(t, allUsersWildcard, st.Principal.AWS)
			require.Equal(t, "Deny", st.Effect)
			require.Equal(t, []string{arnAwsPrefix + bktName}, st.Resource)
		}
	}

	newPolicy := &bucketPolicy{
		Statement: []statement{{
			Effect:    "Allow",
			Principal: principal{AWS: allUsersWildcard},
			Action:    []string{s3GetObject},
			Resource:  []string{arnAwsPrefix + "dummy"},
		}},
	}

	putBucketPolicy(hc, bktName, newPolicy, box, http.StatusInternalServerError)

	newPolicy.Statement[0].Resource[0] = arnAwsPrefix + bktName
	putBucketPolicy(hc, bktName, newPolicy, box, http.StatusOK)

	bktPolicy = getBucketPolicy(hc, bktName)
	for _, st := range bktPolicy.Statement {
		if st.Effect == "Allow" && st.Principal.AWS == allUsersWildcard {
			require.Equal(t, []string{arnAwsPrefix + bktName}, st.Resource)
			require.ElementsMatch(t, []string{s3GetObject, s3ListBucket}, st.Action)
		}
	}
}

func getBucketPolicy(hc *handlerContext, bktName string) *bucketPolicy {
	w, r := prepareTestRequest(hc, bktName, "", nil)
	hc.Handler().GetBucketPolicyHandler(w, r)

	assertStatus(hc.t, w, http.StatusOK)
	policy := &bucketPolicy{}
	err := json.NewDecoder(w.Result().Body).Decode(policy)
	require.NoError(hc.t, err)
	return policy
}

func putBucketPolicy(hc *handlerContext, bktName string, bktPolicy *bucketPolicy, box *accessbox.Box, status int) {
	body, err := json.Marshal(bktPolicy)
	require.NoError(hc.t, err)

	w, r := prepareTestPayloadRequest(hc, bktName, "", bytes.NewReader(body))
	ctx := context.WithValue(r.Context(), api.BoxData, box)
	r = r.WithContext(ctx)
	hc.Handler().PutBucketPolicyHandler(w, r)
	assertStatus(hc.t, w, status)
}

func checkLastRecords(t *testing.T, tc *handlerContext, bktInfo *data.BucketInfo, action eacl.Action) {
	bktACL, err := tc.Layer().GetBucketACL(tc.Context(), bktInfo)
	require.NoError(t, err)

	length := len(bktACL.EACL.Records())

	if length < 7 {
		t.Fatalf("length of records is less than 7: '%d'", length)
	}

	for _, rec := range bktACL.EACL.Records()[length-7:] {
		if rec.Action() != action || rec.Targets()[0].Role() != eacl.RoleOthers {
			t.Fatalf("inavid last record: '%s', '%s', '%s',", rec.Action(), rec.Operation(), rec.Targets()[0].Role())
		}
	}
}

func createAccessBox(t *testing.T) (*accessbox.Box, *keys.PrivateKey) {
	key, err := keys.NewPrivateKey()
	require.NoError(t, err)

	var bearerToken bearer.Token
	err = bearerToken.Sign(key.PrivateKey)
	require.NoError(t, err)

	tok := new(session.Container)
	tok.ForVerb(session.VerbContainerSetEACL)

	tok2 := new(session.Container)
	tok2.ForVerb(session.VerbContainerPut)
	box := &accessbox.Box{
		Gate: &accessbox.GateData{
			SessionTokens: []*session.Container{tok, tok2},
			BearerToken:   &bearerToken,
		},
	}

	return box, key
}

func createBucket(t *testing.T, tc *handlerContext, bktName string, box *accessbox.Box) *data.BucketInfo {
	w, r := prepareTestRequest(tc, bktName, "", nil)
	ctx := context.WithValue(r.Context(), api.BoxData, box)
	r = r.WithContext(ctx)
	tc.Handler().CreateBucketHandler(w, r)
	assertStatus(t, w, http.StatusOK)

	bktInfo, err := tc.Layer().GetBucketInfo(tc.Context(), bktName)
	require.NoError(t, err)
	return bktInfo
}

func putBucketACL(t *testing.T, tc *handlerContext, bktName string, box *accessbox.Box, header map[string]string) {
	w, r := prepareTestRequest(tc, bktName, "", nil)
	for key, val := range header {
		r.Header.Set(key, val)
	}
	ctx := context.WithValue(r.Context(), api.BoxData, box)
	r = r.WithContext(ctx)
	tc.Handler().PutBucketACLHandler(w, r)
	assertStatus(t, w, http.StatusOK)
}
