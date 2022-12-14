package neofs

import (
	"errors"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"github.com/stretchr/testify/require"
)

func TestLockConfigurationEncoding(t *testing.T) {
	for _, tc := range []struct {
		name            string
		encoded         string
		expectedEncoded string
		expected        data.ObjectLockConfiguration
		error           bool
	}{
		{
			name:            "empty",
			encoded:         "",
			expectedEncoded: "",
			expected:        data.ObjectLockConfiguration{},
		},
		{
			name:            "Enabled",
			encoded:         "Enabled",
			expectedEncoded: "Enabled",
			expected: data.ObjectLockConfiguration{
				ObjectLockEnabled: "Enabled",
			},
		},
		{
			name:            "Fully enabled",
			encoded:         "Enabled,10,COMPLIANCE,",
			expectedEncoded: "Enabled,10,COMPLIANCE,0",
			expected: data.ObjectLockConfiguration{
				ObjectLockEnabled: "Enabled",
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Days: 10,
						Mode: "COMPLIANCE",
					},
				},
			},
		},
		{
			name:            "Missing numbers",
			encoded:         "Enabled,,COMPLIANCE,",
			expectedEncoded: "Enabled,0,COMPLIANCE,0",
			expected: data.ObjectLockConfiguration{
				ObjectLockEnabled: "Enabled",
				Rule: &data.ObjectLockRule{
					DefaultRetention: &data.DefaultRetention{
						Mode: "COMPLIANCE",
					},
				},
			},
		},
		{
			name:            "Missing all",
			encoded:         ",,,",
			expectedEncoded: ",0,,0",
			expected:        data.ObjectLockConfiguration{Rule: &data.ObjectLockRule{DefaultRetention: &data.DefaultRetention{}}},
		},
		{
			name:    "Invalid args",
			encoded: ",,",
			error:   true,
		},
		{
			name:    "Invalid days",
			encoded: ",a,,",
			error:   true,
		},
		{
			name:    "Invalid years",
			encoded: ",,,b",
			error:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lockConfiguration, err := parseLockConfiguration(tc.encoded)
			if tc.error {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, *lockConfiguration)

			encoded := encodeLockConfiguration(lockConfiguration)
			require.Equal(t, tc.expectedEncoded, encoded)
		})
	}
}

func TestHandleError(t *testing.T) {
	defaultError := errors.New("default error")
	for _, tc := range []struct {
		err           error
		expectedError error
	}{
		{
			err:           defaultError,
			expectedError: defaultError,
		},
		{
			err:           errors.New("something not found"),
			expectedError: layer.ErrNodeNotFound,
		},
		{
			err:           errors.New("something is denied by some acl rule"),
			expectedError: layer.ErrNodeAccessDenied,
		},
	} {
		t.Run("", func(t *testing.T) {
			err := handleError("err message", tc.err)
			require.True(t, errors.Is(err, tc.expectedError))
		})
	}
}
