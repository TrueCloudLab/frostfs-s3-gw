package handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	apiErrors "github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/stretchr/testify/require"
)

const defaultURL = "http://localhost/"

func TestFormObjectLock(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name          string
		bktInfo       *data.BucketInfo
		config        *data.ObjectLockConfiguration
		header        http.Header
		expectedError bool
		expectedLock  *data.ObjectLock
	}{
		{
			name:    "default days",
			bktInfo: &data.BucketInfo{ObjectLockEnabled: true},
			config: &data.ObjectLockConfiguration{Rule: &data.ObjectLockRule{
				DefaultRetention: &data.DefaultRetention{Mode: complianceMode, Days: 1}}},
			expectedLock: &data.ObjectLock{Retention: &data.RetentionLock{
				IsCompliance: true,
				Until:        time.Now().Add(24 * time.Hour)}},
		},
		{
			name:    "default years",
			bktInfo: &data.BucketInfo{ObjectLockEnabled: true},
			config: &data.ObjectLockConfiguration{Rule: &data.ObjectLockRule{
				DefaultRetention: &data.DefaultRetention{Mode: governanceMode, Years: 1}}},
			expectedLock: &data.ObjectLock{Retention: &data.RetentionLock{
				Until: time.Now().Add(365 * 24 * time.Hour)}},
		},
		{
			name:    "basic override",
			bktInfo: &data.BucketInfo{ObjectLockEnabled: true},
			config:  &data.ObjectLockConfiguration{Rule: &data.ObjectLockRule{DefaultRetention: &data.DefaultRetention{Mode: complianceMode, Days: 1}}},
			header: map[string][]string{
				api.AmzObjectLockRetainUntilDate: {time.Now().Add(time.Minute).Format(time.RFC3339)},
				api.AmzObjectLockMode:            {governanceMode},
				api.AmzObjectLockLegalHold:       {legalHoldOn},
			},
			expectedLock: &data.ObjectLock{
				LegalHold: &data.LegalHoldLock{Enabled: true},
				Retention: &data.RetentionLock{Until: time.Now().Add(time.Minute)}},
		},
		{
			name:          "lock disabled error",
			bktInfo:       &data.BucketInfo{},
			header:        map[string][]string{api.AmzObjectLockLegalHold: {legalHoldOn}},
			expectedError: true,
		},
		{
			name:    "invalid time format error",
			bktInfo: &data.BucketInfo{ObjectLockEnabled: true},
			header: map[string][]string{
				api.AmzObjectLockRetainUntilDate: {time.Now().Format(time.RFC822)},
			},
			expectedError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actualObjLock, err := formObjectLock(ctx, tc.bktInfo, tc.config, tc.header)
			if tc.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assertObjectLocks(t, tc.expectedLock, actualObjLock)
		})
	}
}

func TestFormObjectLockFromRetention(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name          string
		retention     *data.Retention
		header        http.Header
		expectedError bool
		expectedLock  *data.ObjectLock
	}{
		{
			name: "basic compliance",
			retention: &data.Retention{
				Mode:            complianceMode,
				RetainUntilDate: time.Now().Add(time.Minute).Format(time.RFC3339),
			},
			expectedLock: &data.ObjectLock{Retention: &data.RetentionLock{
				Until:        time.Now().Add(time.Minute),
				IsCompliance: true}},
		},
		{
			name: "basic governance",
			retention: &data.Retention{
				Mode:            governanceMode,
				RetainUntilDate: time.Now().Add(time.Minute).Format(time.RFC3339),
			},
			header: map[string][]string{
				api.AmzBypassGovernanceRetention: {strconv.FormatBool(true)},
			},
			expectedLock: &data.ObjectLock{Retention: &data.RetentionLock{Until: time.Now().Add(time.Minute)}},
		},
		{
			name: "error invalid mode",
			retention: &data.Retention{
				Mode:            "",
				RetainUntilDate: time.Now().Format(time.RFC3339),
			},
			expectedError: true,
		},
		{
			name: "error invalid date",
			retention: &data.Retention{
				Mode:            governanceMode,
				RetainUntilDate: "",
			},
			expectedError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actualObjLock, err := formObjectLockFromRetention(ctx, tc.retention, tc.header)
			if tc.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assertObjectLocks(t, tc.expectedLock, actualObjLock)
		})
	}
}

func assertObjectLocks(t *testing.T, expected, actual *data.ObjectLock) {
	require.Equal(t, expected.LegalHold, actual.LegalHold)
	if expected.Retention != nil {
		require.Equal(t, expected.Retention.IsCompliance, actual.Retention.IsCompliance)
		require.InDelta(t, expected.Retention.Until.Unix(), actual.Retention.Until.Unix(), 1)
	}
}

func TestLockConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name          string
		configuration *data.ObjectLockConfiguration
		expectedError bool
	}{
		{
			name:          "basic empty",
			configuration: &data.ObjectLockConfiguration{},
		},
		{
			name: "basic compliance",
			configuration: &data.ObjectLockConfiguration{
				ObjectLockEnabled: enabledValue,
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Days: 1,
						Mode: complianceMode,
					},
				},
			},
		},
		{
			name: "basic governance",
			configuration: &data.ObjectLockConfiguration{
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Mode:  governanceMode,
						Years: 1,
					},
				},
			},
		},
		{
			name: "error invalid enabled",
			configuration: &data.ObjectLockConfiguration{
				ObjectLockEnabled: "false",
			},
			expectedError: true,
		},
		{
			name: "error invalid mode",
			configuration: &data.ObjectLockConfiguration{
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Mode: "",
					},
				},
			},
			expectedError: true,
		},
		{
			name: "error no duration",
			configuration: &data.ObjectLockConfiguration{
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Mode: governanceMode,
					},
				},
			},
			expectedError: true,
		},
		{
			name: "error both durations",
			configuration: &data.ObjectLockConfiguration{
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Days:  1,
						Mode:  governanceMode,
						Years: 1,
					},
				},
			},
			expectedError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkLockConfiguration(tc.configuration)
			if tc.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestPutBucketLockConfigurationHandler(t *testing.T) {
	ctx := context.Background()
	hc := prepareHandlerContext(t)

	bktLockDisabled := "bucket-lock-disabled"
	createTestBucket(hc, bktLockDisabled)

	bktLockEnabled := "bucket-lock-enabled"
	createTestBucketWithLock(hc, bktLockEnabled, nil)

	bktLockEnabledWithOldConfig := "bucket-lock-enabled-old-conf"
	createTestBucketWithLock(hc, bktLockEnabledWithOldConfig,
		&data.ObjectLockConfiguration{
			Rule: &data.ObjectLockRule{
				DefaultRetention: &data.DefaultRetention{
					Days: 1,
					Mode: complianceMode,
				},
			},
		})

	for _, tc := range []struct {
		name          string
		bucket        string
		expectedError apiErrors.Error
		noError       bool
		configuration *data.ObjectLockConfiguration
	}{
		{
			name:          "bkt not found",
			expectedError: apiErrors.GetAPIError(apiErrors.ErrNoSuchBucket),
		},
		{
			name:          "bkt lock disabled",
			bucket:        bktLockDisabled,
			expectedError: apiErrors.GetAPIError(apiErrors.ErrObjectLockConfigurationNotAllowed),
		},
		{
			name:          "invalid configuration",
			bucket:        bktLockEnabled,
			expectedError: apiErrors.GetAPIError(apiErrors.ErrInternalError),
			configuration: &data.ObjectLockConfiguration{ObjectLockEnabled: "dummy"},
		},
		{
			name:          "basic",
			bucket:        bktLockEnabled,
			noError:       true,
			configuration: &data.ObjectLockConfiguration{},
		},
		{
			name:    "basic override",
			bucket:  bktLockEnabledWithOldConfig,
			noError: true,
			configuration: &data.ObjectLockConfiguration{
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Mode:  governanceMode,
						Years: 1,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := xml.Marshal(tc.configuration)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, defaultURL, bytes.NewReader(body))
			r = r.WithContext(api.SetReqInfo(r.Context(), api.NewReqInfo(w, r, api.ObjectRequest{Bucket: tc.bucket})))

			hc.Handler().PutBucketObjectLockConfigHandler(w, r)

			if !tc.noError {
				assertS3Error(t, w, tc.expectedError)
				return
			}

			bktInfo, err := hc.Layer().GetBucketInfo(ctx, tc.bucket)
			require.NoError(t, err)
			bktSettings, err := hc.Layer().GetBucketSettings(ctx, bktInfo)
			require.NoError(t, err)
			actualConf := bktSettings.LockConfiguration
			require.True(t, bktSettings.VersioningEnabled())
			require.Equal(t, tc.configuration.ObjectLockEnabled, actualConf.ObjectLockEnabled)
			require.Equal(t, tc.configuration.Rule, actualConf.Rule)
		})
	}
}

func TestGetBucketLockConfigurationHandler(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktLockDisabled := "bucket-lock-disabled"
	createTestBucket(hc, bktLockDisabled)

	bktLockEnabled := "bucket-lock-enabled"
	createTestBucketWithLock(hc, bktLockEnabled, nil)

	oldConfig := &data.ObjectLockConfiguration{
		Rule: &data.ObjectLockRule{
			DefaultRetention: &data.DefaultRetention{
				Days: 1,
				Mode: complianceMode,
			},
		},
	}
	bktLockEnabledWithOldConfig := "bucket-lock-enabled-old-conf"
	createTestBucketWithLock(hc, bktLockEnabledWithOldConfig, oldConfig)

	for _, tc := range []struct {
		name          string
		bucket        string
		expectedError apiErrors.Error
		noError       bool
		expectedConf  *data.ObjectLockConfiguration
	}{
		{
			name:          "bkt not found",
			expectedError: apiErrors.GetAPIError(apiErrors.ErrNoSuchBucket),
		},
		{
			name:          "bkt lock disabled",
			bucket:        bktLockDisabled,
			expectedError: apiErrors.GetAPIError(apiErrors.ErrObjectLockConfigurationNotFound),
		},
		{
			name:         "bkt lock enabled empty default",
			bucket:       bktLockEnabled,
			noError:      true,
			expectedConf: &data.ObjectLockConfiguration{ObjectLockEnabled: enabledValue},
		},
		{
			name:         "bkt lock enabled",
			bucket:       bktLockEnabledWithOldConfig,
			noError:      true,
			expectedConf: oldConfig,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, defaultURL, bytes.NewReader(nil))
			r = r.WithContext(api.SetReqInfo(r.Context(), api.NewReqInfo(w, r, api.ObjectRequest{Bucket: tc.bucket})))

			hc.Handler().GetBucketObjectLockConfigHandler(w, r)

			if !tc.noError {
				assertS3Error(t, w, tc.expectedError)
				return
			}

			actualConf := &data.ObjectLockConfiguration{}
			err := xml.NewDecoder(w.Result().Body).Decode(actualConf)
			require.NoError(t, err)

			require.Equal(t, tc.expectedConf.ObjectLockEnabled, actualConf.ObjectLockEnabled)
			require.Equal(t, tc.expectedConf.Rule, actualConf.Rule)
		})
	}
}

func assertS3Error(t *testing.T, w *httptest.ResponseRecorder, expectedError apiErrors.Error) {
	actualErrorResponse := &api.ErrorResponse{}
	err := xml.NewDecoder(w.Result().Body).Decode(actualErrorResponse)
	require.NoError(t, err)

	require.Equal(t, expectedError.HTTPStatusCode, w.Code)
	require.Equal(t, expectedError.Code, actualErrorResponse.Code)

	if expectedError.ErrCode != apiErrors.ErrInternalError {
		require.Equal(t, expectedError.Description, actualErrorResponse.Message)
	}
}

func TestObjectLegalHold(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName := "bucket-lock-enabled"
	bktInfo := createTestBucketWithLock(hc, bktName, nil)

	objName := "obj-for-legal-hold"
	createTestObject(hc, bktInfo, objName)

	getObjectLegalHold(hc, bktName, objName, legalHoldOff)

	putObjectLegalHold(hc, bktName, objName, legalHoldOn)
	getObjectLegalHold(hc, bktName, objName, legalHoldOn)

	// to make sure put hold is an idempotent operation
	putObjectLegalHold(hc, bktName, objName, legalHoldOn)

	putObjectLegalHold(hc, bktName, objName, legalHoldOff)
	getObjectLegalHold(hc, bktName, objName, legalHoldOff)

	// to make sure put hold is an idempotent operation
	putObjectLegalHold(hc, bktName, objName, legalHoldOff)
}

func getObjectLegalHold(hc *handlerContext, bktName, objName, status string) {
	w, r := prepareTestRequest(hc, bktName, objName, nil)
	hc.Handler().GetObjectLegalHoldHandler(w, r)
	assertLegalHold(hc.t, w, status)
}

func putObjectLegalHold(hc *handlerContext, bktName, objName, status string) {
	w, r := prepareTestRequest(hc, bktName, objName, &data.LegalHold{Status: status})
	hc.Handler().PutObjectLegalHoldHandler(w, r)
	assertStatus(hc.t, w, http.StatusOK)
}

func assertLegalHold(t *testing.T, w *httptest.ResponseRecorder, status string) {
	actualHold := &data.LegalHold{}
	err := xml.NewDecoder(w.Result().Body).Decode(actualHold)
	require.NoError(t, err)
	require.Equal(t, status, actualHold.Status)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestObjectRetention(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName := "bucket-lock-enabled"
	bktInfo := createTestBucketWithLock(hc, bktName, nil)

	objName := "obj-for-retention"
	createTestObject(hc, bktInfo, objName)

	getObjectRetention(hc, bktName, objName, nil, apiErrors.ErrNoSuchKey)

	retention := &data.Retention{Mode: governanceMode, RetainUntilDate: time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}
	putObjectRetention(hc, bktName, objName, retention, false, 0)
	getObjectRetention(hc, bktName, objName, retention, 0)

	retention = &data.Retention{Mode: governanceMode, RetainUntilDate: time.Now().UTC().Add(time.Minute).Format(time.RFC3339)}
	putObjectRetention(hc, bktName, objName, retention, false, apiErrors.ErrInternalError)

	retention = &data.Retention{Mode: complianceMode, RetainUntilDate: time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}
	putObjectRetention(hc, bktName, objName, retention, true, 0)
	getObjectRetention(hc, bktName, objName, retention, 0)

	putObjectRetention(hc, bktName, objName, retention, true, apiErrors.ErrInternalError)
}

func getObjectRetention(hc *handlerContext, bktName, objName string, retention *data.Retention, errCode apiErrors.ErrorCode) {
	w, r := prepareTestRequest(hc, bktName, objName, nil)
	hc.Handler().GetObjectRetentionHandler(w, r)
	if errCode == 0 {
		assertRetention(hc.t, w, retention)
	} else {
		assertS3Error(hc.t, w, apiErrors.GetAPIError(errCode))
	}
}

func putObjectRetention(hc *handlerContext, bktName, objName string, retention *data.Retention, byPass bool, errCode apiErrors.ErrorCode) {
	w, r := prepareTestRequest(hc, bktName, objName, retention)
	if byPass {
		r.Header.Set(api.AmzBypassGovernanceRetention, strconv.FormatBool(true))
	}
	hc.Handler().PutObjectRetentionHandler(w, r)
	if errCode == 0 {
		assertStatus(hc.t, w, http.StatusOK)
	} else {
		assertS3Error(hc.t, w, apiErrors.GetAPIError(errCode))
	}
}

func assertRetention(t *testing.T, w *httptest.ResponseRecorder, retention *data.Retention) {
	actualRetention := &data.Retention{}
	err := xml.NewDecoder(w.Result().Body).Decode(actualRetention)
	require.NoError(t, err)
	require.Equal(t, retention.Mode, actualRetention.Mode)
	require.Equal(t, retention.RetainUntilDate, actualRetention.RetainUntilDate)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestPutObjectWithLock(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName := "bucket-lock-enabled"
	lockConfig := &data.ObjectLockConfiguration{
		ObjectLockEnabled: enabledValue,
		Rule: &data.ObjectLockRule{
			DefaultRetention: &data.DefaultRetention{
				Days: 1,
				Mode: governanceMode,
			},
		},
	}
	createTestBucketWithLock(hc, bktName, lockConfig)

	objDefault := "obj-default-retention"
	putObject(t, hc, bktName, objDefault)

	getObjectRetentionApproximate(hc, bktName, objDefault, governanceMode, time.Now().Add(24*time.Hour))
	getObjectLegalHold(hc, bktName, objDefault, legalHoldOff)

	objOverride := "obj-override-retention"
	w, r := prepareTestRequest(hc, bktName, objOverride, nil)
	r.Header.Set(api.AmzObjectLockMode, complianceMode)
	r.Header.Set(api.AmzObjectLockLegalHold, legalHoldOn)
	r.Header.Set(api.AmzBypassGovernanceRetention, "true")
	r.Header.Set(api.AmzObjectLockRetainUntilDate, time.Now().Add(2*24*time.Hour).Format(time.RFC3339))
	hc.Handler().PutObjectHandler(w, r)
	assertStatus(t, w, http.StatusOK)

	getObjectRetentionApproximate(hc, bktName, objOverride, complianceMode, time.Now().Add(2*24*time.Hour))
	getObjectLegalHold(hc, bktName, objOverride, legalHoldOn)
}

func getObjectRetentionApproximate(hc *handlerContext, bktName, objName, mode string, untilDate time.Time) {
	w, r := prepareTestRequest(hc, bktName, objName, nil)
	hc.Handler().GetObjectRetentionHandler(w, r)
	expectedRetention := &data.Retention{
		Mode:            mode,
		RetainUntilDate: untilDate.Format(time.RFC3339),
	}
	assertRetentionApproximate(hc.t, w, expectedRetention, 1)
}

func TestPutLockErrors(t *testing.T) {
	hc := prepareHandlerContext(t)

	bktName, objName := "bucket-lock-enabled", "object"
	createTestBucketWithLock(hc, bktName, nil)

	headers := map[string]string{api.AmzObjectLockMode: complianceMode}
	putObjectWithLockFailed(t, hc, bktName, objName, headers, apiErrors.ErrObjectLockInvalidHeaders)

	delete(headers, api.AmzObjectLockMode)
	headers[api.AmzObjectLockRetainUntilDate] = time.Now().Add(time.Minute).Format(time.RFC3339)
	putObjectWithLockFailed(t, hc, bktName, objName, headers, apiErrors.ErrObjectLockInvalidHeaders)

	headers[api.AmzObjectLockMode] = "dummy"
	putObjectWithLockFailed(t, hc, bktName, objName, headers, apiErrors.ErrUnknownWORMModeDirective)

	headers[api.AmzObjectLockMode] = complianceMode
	headers[api.AmzObjectLockRetainUntilDate] = time.Now().Format(time.RFC3339)
	putObjectWithLockFailed(t, hc, bktName, objName, headers, apiErrors.ErrPastObjectLockRetainDate)

	headers[api.AmzObjectLockRetainUntilDate] = "dummy"
	putObjectWithLockFailed(t, hc, bktName, objName, headers, apiErrors.ErrInvalidRetentionDate)

	putObject(t, hc, bktName, objName)

	retention := &data.Retention{Mode: governanceMode}
	putObjectRetentionFailed(t, hc, bktName, objName, retention, apiErrors.ErrMalformedXML)

	retention.Mode = "dummy"
	retention.RetainUntilDate = time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
	putObjectRetentionFailed(t, hc, bktName, objName, retention, apiErrors.ErrMalformedXML)

	retention.Mode = governanceMode
	retention.RetainUntilDate = time.Now().UTC().Format(time.RFC3339)
	putObjectRetentionFailed(t, hc, bktName, objName, retention, apiErrors.ErrPastObjectLockRetainDate)
}

func putObjectWithLockFailed(t *testing.T, hc *handlerContext, bktName, objName string, headers map[string]string, errCode apiErrors.ErrorCode) {
	w, r := prepareTestRequest(hc, bktName, objName, nil)

	for key, val := range headers {
		r.Header.Set(key, val)
	}

	hc.Handler().PutObjectHandler(w, r)
	assertS3Error(t, w, apiErrors.GetAPIError(errCode))
}

func putObjectRetentionFailed(t *testing.T, hc *handlerContext, bktName, objName string, retention *data.Retention, errCode apiErrors.ErrorCode) {
	w, r := prepareTestRequest(hc, bktName, objName, retention)
	hc.Handler().PutObjectRetentionHandler(w, r)
	assertS3Error(t, w, apiErrors.GetAPIError(errCode))
}

func assertRetentionApproximate(t *testing.T, w *httptest.ResponseRecorder, retention *data.Retention, delta float64) {
	actualRetention := &data.Retention{}
	err := xml.NewDecoder(w.Result().Body).Decode(actualRetention)
	require.NoError(t, err)
	require.Equal(t, retention.Mode, actualRetention.Mode)
	require.Equal(t, http.StatusOK, w.Code)

	actualUntil, err := time.Parse(time.RFC3339, actualRetention.RetainUntilDate)
	require.NoError(t, err)

	expectedUntil, err := time.Parse(time.RFC3339, retention.RetainUntilDate)
	require.NoError(t, err)

	require.InDelta(t, expectedUntil.Unix(), actualUntil.Unix(), delta)
}
