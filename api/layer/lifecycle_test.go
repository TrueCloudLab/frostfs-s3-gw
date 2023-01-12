package layer

import (
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/stretchr/testify/require"
)

func TestComputeLifecycleID(t *testing.T) {
	conf := &data.LifecycleConfiguration{Rules: []data.Rule{
		{
			ID:     "id",
			Status: "Enabled",
		},
	}}

	id, err := computeLifecycleID(conf)
	require.NoError(t, err)
	require.Equal(t, "51ff619dc848622287764fc7c4aec06b7c1a5936c25b8eee48a0dbcb4eeac9f4", id)
}

func TestRuleMatchObject(t *testing.T) {
	prefix, suffix := "prefix", "suffix"
	objSizeMin, objSizeMax := int64(512), int64(1024)

	for _, tc := range []struct {
		name     string
		rule     data.Rule
		obj      *data.ObjectInfo
		tags     map[string]string
		expected bool
	}{
		{
			name:     "basic match",
			rule:     data.Rule{Prefix: &prefix},
			obj:      &data.ObjectInfo{Name: prefix + suffix},
			expected: true,
		},
		{
			name:     "basic no match",
			rule:     data.Rule{Prefix: &prefix},
			obj:      &data.ObjectInfo{Name: suffix + prefix},
			expected: false,
		},
		{
			name: "filter and sizes",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				And: &data.LifecycleRuleAndOperator{
					ObjectSizeGreaterThan: &objSizeMin,
					ObjectSizeLessThan:    &objSizeMax,
				},
			}},
			obj:      &data.ObjectInfo{Name: suffix, Size: 768},
			expected: true,
		},
		{
			name: "filter prefix",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				Prefix: &prefix,
			}},
			obj:      &data.ObjectInfo{Name: prefix + suffix},
			expected: true,
		},
		{
			name: "filter prefix no match",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				Prefix: &prefix,
			}},
			obj:      &data.ObjectInfo{Name: suffix},
			expected: false,
		},
		{
			name: "filter tags",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				Tag: &data.Tag{
					Key:   "key",
					Value: "val",
				},
			}},
			tags:     map[string]string{"key": "val"},
			obj:      &data.ObjectInfo{},
			expected: true,
		},
		{
			name: "filter and tags no match",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				And: &data.LifecycleRuleAndOperator{
					Tags: []data.Tag{{
						Key:   "key",
						Value: "val",
					}},
				},
			}},
			tags:     map[string]string{"key": "val2"},
			obj:      &data.ObjectInfo{},
			expected: false,
		},
		{
			name: "filter size no match",
			rule: data.Rule{Filter: &data.LifecycleRuleFilter{
				ObjectSizeGreaterThan: &objSizeMax,
			}},
			obj:      &data.ObjectInfo{Size: objSizeMin},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.rule.MatchObject(tc.obj, tc.tags))
		})
	}
}

func TestScheduleLifecycle(t *testing.T) {
	tc := prepareContext(t)

	obj1 := tc.putObject([]byte("content"))

	date := "2022-03-14T09:59:03Z"
	date2 := "2022-03-15T09:59:03Z"
	prefix := "prefix"
	tc.obj = prefix
	obj2 := tc.putObject([]byte("content2"))

	conf := &data.LifecycleConfiguration{
		Rules: []data.Rule{{
			Filter: &data.LifecycleRuleFilter{
				Prefix: &prefix,
			},
			Expiration: &data.Expiration{
				Date: &date,
			}},
		},
	}

	err := tc.layer.ScheduleLifecycle(tc.ctx, tc.bktInfo, conf)
	require.NoError(t, err)

	expObj1, _ := tc.getObject(obj1.ExpirationObject(), "", false)
	require.Nil(t, expObj1)
	expObj2, _ := tc.getObject(obj2.ExpirationObject(), "", false)
	require.NotNil(t, expObj2)
	assertExpirationObject(t, expObj2, date)

	conf.Rules[0].Expiration.Date = &date2
	err = tc.layer.ScheduleLifecycle(tc.ctx, tc.bktInfo, conf)
	require.NoError(t, err)

	expObj2, _ = tc.getObject(obj2.ExpirationObject(), "", false)
	require.NotNil(t, expObj2)
	assertExpirationObject(t, expObj2, date2)
}

func assertExpirationObject(t *testing.T, expObjInfo *data.ObjectInfo, date string) {
	require.Equal(t, expObjInfo.Headers[AttributeExpireDate], date)
	require.Contains(t, expObjInfo.Headers, AttributeSysTickEpoch)
	require.Contains(t, expObjInfo.Headers, AttributeSysTickTopic)
	require.Contains(t, expObjInfo.Headers, AttributeLifecycleConfigID)
}
