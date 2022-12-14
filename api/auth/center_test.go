package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/stretchr/testify/require"
)

func TestAuthHeaderParse(t *testing.T) {
	defaultHeader := "AWS4-HMAC-SHA256 Credential=oid0cid/20210809/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=2811ccb9e242f41426738fb1f"

	center := &center{
		reg: NewRegexpMatcher(authorizationFieldRegexp),
	}

	for _, tc := range []struct {
		header   string
		err      error
		expected *authHeader
	}{
		{
			header: defaultHeader,
			err:    nil,
			expected: &authHeader{
				AccessKeyID:  "oid0cid",
				Service:      "s3",
				Region:       "us-east-1",
				SignatureV4:  "2811ccb9e242f41426738fb1f",
				SignedFields: []string{"host", "x-amz-content-sha256", "x-amz-date"},
				Date:         "20210809",
			},
		},
		{
			header:   strings.ReplaceAll(defaultHeader, "Signature=2811ccb9e242f41426738fb1f", ""),
			err:      errors.GetAPIError(errors.ErrAuthorizationHeaderMalformed),
			expected: nil,
		},
		{
			header:   strings.ReplaceAll(defaultHeader, "oid0cid", "oidcid"),
			err:      errors.GetAPIError(errors.ErrInvalidAccessKeyID),
			expected: nil,
		},
	} {
		authHeader, err := center.parseAuthHeader(tc.header)
		require.Equal(t, tc.err, err, tc.header)
		require.Equal(t, tc.expected, authHeader, tc.header)
	}
}

func TestAuthHeaderGetAddress(t *testing.T) {
	defaulErr := errors.GetAPIError(errors.ErrInvalidAccessKeyID)

	for _, tc := range []struct {
		authHeader *authHeader
		err        error
	}{
		{
			authHeader: &authHeader{
				AccessKeyID: "vWqF8cMDRbJcvnPLALoQGnABPPhw8NyYMcGsfDPfZJM0HrgjonN8CgFvCZ3kh9BUXw4W2tJ5E7EAGhueSF122HB",
			},
			err: nil,
		},
		{
			authHeader: &authHeader{
				AccessKeyID: "vWqF8cMDRbJcvnPLALoQGnABPPhw8NyYMcGsfDPfZJMHrgjonN8CgFvCZ3kh9BUXw4W2tJ5E7EAGhueSF122HB",
			},
			err: defaulErr,
		},
		{
			authHeader: &authHeader{
				AccessKeyID: "oid0cid",
			},
			err: defaulErr,
		},
		{
			authHeader: &authHeader{
				AccessKeyID: "oidcid",
			},
			err: defaulErr,
		},
	} {
		_, err := tc.authHeader.getAddress()
		require.Equal(t, tc.err, err, tc.authHeader.AccessKeyID)
	}
}

func TestSignature(t *testing.T) {
	secret := "66be461c3cd429941c55daf42fad2b8153e5a2016ba89c9494d97677cc9d3872"
	strToSign := "eyAiZXhwaXJhdGlvbiI6ICIyMDE1LTEyLTMwVDEyOjAwOjAwLjAwMFoiLAogICJjb25kaXRpb25zIjogWwogICAgeyJidWNrZXQiOiAiYWNsIn0sCiAgICBbInN0YXJ0cy13aXRoIiwgIiRrZXkiLCAidXNlci91c2VyMS8iXSwKICAgIHsic3VjY2Vzc19hY3Rpb25fcmVkaXJlY3QiOiAiaHR0cDovL2xvY2FsaG9zdDo4MDg0L2FjbCJ9LAogICAgWyJzdGFydHMtd2l0aCIsICIkQ29udGVudC1UeXBlIiwgImltYWdlLyJdLAogICAgeyJ4LWFtei1tZXRhLXV1aWQiOiAiMTQzNjUxMjM2NTEyNzQifSwKICAgIFsic3RhcnRzLXdpdGgiLCAiJHgtYW16LW1ldGEtdGFnIiwgIiJdLAoKICAgIHsiWC1BbXotQ3JlZGVudGlhbCI6ICI4Vmk0MVBIbjVGMXNzY2J4OUhqMXdmMUU2aERUYURpNndxOGhxTU05NllKdTA1QzVDeUVkVlFoV1E2aVZGekFpTkxXaTlFc3BiUTE5ZDRuR3pTYnZVZm10TS8yMDE1MTIyOS91cy1lYXN0LTEvczMvYXdzNF9yZXF1ZXN0In0sCiAgICB7IngtYW16LWFsZ29yaXRobSI6ICJBV1M0LUhNQUMtU0hBMjU2In0sCiAgICB7IlgtQW16LURhdGUiOiAiMjAxNTEyMjlUMDAwMDAwWiIgfSwKICAgIHsieC1pZ25vcmUtdG1wIjogInNvbWV0aGluZyIgfQogIF0KfQ=="

	signTime, err := time.Parse("20060102T150405Z", "20151229T000000Z")
	if err != nil {
		panic(err)
	}

	signature := signStr(secret, "s3", "us-east-1", signTime, strToSign)
	require.Equal(t, "dfbe886241d9e369cf4b329ca0f15eb27306c97aa1022cc0bb5a914c4ef87634", signature)
}
