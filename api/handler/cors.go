package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/layer"
	"go.uber.org/zap"
)

const (
	// DefaultMaxAge is a default value of Access-Control-Max-Age if this value is not set in a rule.
	DefaultMaxAge = 600
	wildcard      = "*"
)

func (h *handler) GetBucketCorsHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	cors, err := h.obj.GetBucketCORS(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get cors", reqInfo, err)
		return
	}

	if err = api.EncodeToResponse(w, cors); err != nil {
		h.logAndSendError(w, "could not encode cors to response", reqInfo, err)
		return
	}
}

func (h *handler) PutBucketCorsHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	p := &layer.PutCORSParams{
		BktInfo:      bktInfo,
		Reader:       r.Body,
		CopiesNumber: h.cfg.CopiesNumber,
	}

	if err = h.obj.PutBucketCORS(r.Context(), p); err != nil {
		h.logAndSendError(w, "could not put cors configuration", reqInfo, err)
		return
	}

	api.WriteSuccessResponseHeadersOnly(w)
}

func (h *handler) DeleteBucketCorsHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.getBucketAndCheckOwner(r, reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	if err = h.obj.DeleteBucketCORS(r.Context(), bktInfo); err != nil {
		h.logAndSendError(w, "could not delete cors", reqInfo, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) AppendCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	origin := r.Header.Get(api.Origin)
	if origin == "" {
		return
	}
	reqInfo := api.GetReqInfo(r.Context())
	if reqInfo.BucketName == "" {
		return
	}
	bktInfo, err := h.obj.GetBucketInfo(r.Context(), reqInfo.BucketName)
	if err != nil {
		h.log.Warn("get bucket info", zap.Error(err))
		return
	}

	cors, err := h.obj.GetBucketCORS(r.Context(), bktInfo)
	if err != nil {
		h.log.Warn("get bucket cors", zap.Error(err))
		return
	}

	withCredentials := r.Header.Get(api.Authorization) != ""

	for _, rule := range cors.CORSRules {
		for _, o := range rule.AllowedOrigins {
			if o == origin {
				for _, m := range rule.AllowedMethods {
					if m == r.Method {
						w.Header().Set(api.AccessControlAllowOrigin, origin)
						w.Header().Set(api.AccessControlAllowMethods, strings.Join(rule.AllowedMethods, ", "))
						w.Header().Set(api.AccessControlAllowCredentials, "true")
						w.Header().Set(api.Vary, api.Origin)
						return
					}
				}
			}
			if o == wildcard {
				for _, m := range rule.AllowedMethods {
					if m == r.Method {
						if withCredentials {
							w.Header().Set(api.AccessControlAllowOrigin, origin)
							w.Header().Set(api.AccessControlAllowCredentials, "true")
							w.Header().Set(api.Vary, api.Origin)
						} else {
							w.Header().Set(api.AccessControlAllowOrigin, o)
						}
						w.Header().Set(api.AccessControlAllowMethods, strings.Join(rule.AllowedMethods, ", "))
						return
					}
				}
			}
		}
	}
}

func (h *handler) Preflight(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())
	bktInfo, err := h.obj.GetBucketInfo(r.Context(), reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}

	origin := r.Header.Get(api.Origin)
	if origin == "" {
		h.logAndSendError(w, "origin request header needed", reqInfo, errors.GetAPIError(errors.ErrBadRequest))
	}

	method := r.Header.Get(api.AccessControlRequestMethod)
	if method == "" {
		h.logAndSendError(w, "Access-Control-Request-Method request header needed", reqInfo, errors.GetAPIError(errors.ErrBadRequest))
		return
	}

	var headers []string
	requestHeaders := r.Header.Get(api.AccessControlRequestHeaders)
	if requestHeaders != "" {
		headers = strings.Split(requestHeaders, ", ")
	}

	cors, err := h.obj.GetBucketCORS(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "could not get cors", reqInfo, err)
		return
	}

	for _, rule := range cors.CORSRules {
		for _, o := range rule.AllowedOrigins {
			if o == origin || o == wildcard {
				for _, m := range rule.AllowedMethods {
					if m == method {
						if !checkSubslice(rule.AllowedHeaders, headers) {
							continue
						}
						w.Header().Set(api.AccessControlAllowOrigin, o)
						w.Header().Set(api.AccessControlAllowMethods, strings.Join(rule.AllowedMethods, ", "))
						if headers != nil {
							w.Header().Set(api.AccessControlAllowHeaders, requestHeaders)
						}
						if rule.ExposeHeaders != nil {
							w.Header().Set(api.AccessControlExposeHeaders, strings.Join(rule.ExposeHeaders, ", "))
						}
						if rule.MaxAgeSeconds > 0 || rule.MaxAgeSeconds == -1 {
							w.Header().Set(api.AccessControlMaxAge, strconv.Itoa(rule.MaxAgeSeconds))
						} else {
							w.Header().Set(api.AccessControlMaxAge, strconv.Itoa(h.cfg.DefaultMaxAge))
						}
						if o != wildcard {
							w.Header().Set(api.AccessControlAllowCredentials, "true")
						}
						api.WriteSuccessResponseHeadersOnly(w)
						return
					}
				}
			}
		}
	}
	h.logAndSendError(w, "Forbidden", reqInfo, errors.GetAPIError(errors.ErrAccessDenied))
}

func checkSubslice(slice []string, subSlice []string) bool {
	if sliceContains(slice, wildcard) {
		return true
	}
	if len(subSlice) > len(slice) {
		return false
	}
	for _, r := range subSlice {
		if !sliceContains(slice, r) {
			return false
		}
	}
	return true
}

func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
