package handler

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/nspcc-dev/neofs-s3-gw/api/data"
	apiErrors "github.com/nspcc-dev/neofs-s3-gw/api/errors"
	"github.com/stretchr/testify/require"
)

func TestCheckLifecycleConfiguration(t *testing.T) {
	numRules := 1001
	rules := make([]data.Rule, numRules)
	for i := 0; i < numRules; i++ {
		rules[i] = data.Rule{ID: strconv.Itoa(i), Status: disabledValue}
	}

	prefix := "prefix"
	invalidSize := int64(-1)
	days := int64(1)

	for _, tc := range []struct {
		name          string
		configuration *data.LifecycleConfiguration
		noError       bool
	}{
		{
			name: "basic",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				ID:         "Some ID",
				Status:     "Disabled",
				Expiration: &data.Expiration{Days: &days},
			}}},
			noError: true,
		},
		{
			name: "invalid status",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				ID:     "Some ID",
				Status: "",
			}}},
		},
		{
			name:          "zero rules",
			configuration: &data.LifecycleConfiguration{},
		},
		{
			name:          "more than max rules",
			configuration: &data.LifecycleConfiguration{Rules: rules},
		},
		{
			name: "invalid empty filter",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				Status: enabledValue,
				Filter: &data.LifecycleRuleFilter{},
			}}},
		},
		{
			name: "invalid filter not exactly one option",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				Status: enabledValue,
				Filter: &data.LifecycleRuleFilter{
					Prefix: &prefix,
					Tag:    &data.Tag{},
				},
			}}},
		},
		{
			name: "invalid filter greater obj size",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				Status: enabledValue,
				Filter: &data.LifecycleRuleFilter{
					ObjectSizeGreaterThan: &invalidSize,
				},
			}}},
		},
		{
			name: "invalid filter less obj size",
			configuration: &data.LifecycleConfiguration{Rules: []data.Rule{{
				Status: enabledValue,
				Filter: &data.LifecycleRuleFilter{
					ObjectSizeLessThan: &invalidSize,
				},
			}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkLifecycleConfiguration(tc.configuration)
			if tc.noError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestBucketLifecycleConfiguration(t *testing.T) {
	ctx := context.Background()
	hc := prepareHandlerContext(t)

	bktName := "bucket-for-lifecycle"
	createTestBucket(ctx, t, hc, bktName)

	w, r := prepareTestRequest(t, bktName, "", nil)
	hc.Handler().GetBucketLifecycleHandler(w, r)
	assertS3Error(t, w, apiErrors.GetAPIError(apiErrors.ErrNoSuchLifecycleConfiguration))

	days := int64(1)
	lifecycleConf := &data.LifecycleConfiguration{
		XMLName: xmlName("LifecycleConfiguration"),
		Rules: []data.Rule{
			{
				Expiration: &data.Expiration{Days: &days},
				ID:         "Test",
				Status:     "Disabled",
			},
		}}
	w, r = prepareTestRequest(t, bktName, "", lifecycleConf)
	hc.Handler().PutBucketLifecycleHandler(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	w, r = prepareTestRequest(t, bktName, "", nil)
	hc.Handler().GetBucketLifecycleHandler(w, r)
	assertXMLEqual(t, w, lifecycleConf, &data.LifecycleConfiguration{})

	w, r = prepareTestRequest(t, bktName, "", lifecycleConf)
	hc.Handler().DeleteBucketLifecycleHandler(w, r)
	require.Equal(t, http.StatusNoContent, w.Code)

	// make sure deleting is idempotent operation
	w, r = prepareTestRequest(t, bktName, "", lifecycleConf)
	hc.Handler().DeleteBucketLifecycleHandler(w, r)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func assertXMLEqual(t *testing.T, w *httptest.ResponseRecorder, expected, actual interface{}) {
	err := xml.NewDecoder(w.Result().Body).Decode(actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, http.StatusOK, w.Code)
}

func xmlName(local string) xml.Name {
	return xml.Name{
		Space: "http://s3.amazonaws.com/doc/2006-03-01/",
		Local: local,
	}
}
