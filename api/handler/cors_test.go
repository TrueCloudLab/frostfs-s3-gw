package handler

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
)

func TestCORSOriginWildcard(t *testing.T) {
	body := `
<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
	<CORSRule>
		<AllowedMethod>GET</AllowedMethod>
		<AllowedOrigin>*</AllowedOrigin>
	</CORSRule>
</CORSConfiguration>
`
	hc := prepareHandlerContext(t)

	bktName := "bucket-for-cors"
	box, _ := createAccessBox(t)
	w, r := prepareTestRequest(hc, bktName, "", nil)
	ctx := context.WithValue(r.Context(), api.BoxData, box)
	r = r.WithContext(ctx)
	r.Header.Add(api.AmzACL, "public-read")
	hc.Handler().CreateBucketHandler(w, r)
	assertStatus(t, w, http.StatusOK)

	w, r = prepareTestPayloadRequest(hc, bktName, "", strings.NewReader(body))
	ctx = context.WithValue(r.Context(), api.BoxData, box)
	r = r.WithContext(ctx)
	hc.Handler().PutBucketCorsHandler(w, r)
	assertStatus(t, w, http.StatusOK)

	w, r = prepareTestPayloadRequest(hc, bktName, "", nil)
	hc.Handler().GetBucketCorsHandler(w, r)
	assertStatus(t, w, http.StatusOK)
}
