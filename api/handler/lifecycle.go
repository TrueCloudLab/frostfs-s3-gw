package handler

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/nspcc-dev/neofs-s3-gw/api"
	"github.com/nspcc-dev/neofs-s3-gw/api/data"
	apiErrors "github.com/nspcc-dev/neofs-s3-gw/api/errors"
	"github.com/nspcc-dev/neofs-s3-gw/api/layer"
)

func (h *handler) PutBucketLifecycleHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.obj.GetBucketInfo(r.Context(), reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}
	if err = checkOwner(bktInfo, r.Header.Get(api.AmzExpectedBucketOwner)); err != nil {
		h.logAndSendError(w, "expected owner doesn't match", reqInfo, err)
		return
	}

	lifecycleConf := &data.LifecycleConfiguration{}
	if err = xml.NewDecoder(r.Body).Decode(lifecycleConf); err != nil {
		h.logAndSendError(w, "couldn't parse lifecycle configuration", reqInfo, err)
		return
	}

	if err = checkLifecycleConfiguration(lifecycleConf); err != nil {
		h.logAndSendError(w, "invalid lifecycle configuration", reqInfo, err)
		return
	}

	if err = h.updateLifecycleConfiguration(r.Context(), bktInfo, lifecycleConf); err != nil {
		h.logAndSendError(w, "couldn't put bucket settings", reqInfo, err)
		return
	}
}

func (h *handler) GetBucketLifecycleHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.obj.GetBucketInfo(r.Context(), reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}
	if err = checkOwner(bktInfo, r.Header.Get(api.AmzExpectedBucketOwner)); err != nil {
		h.logAndSendError(w, "expected owner doesn't match", reqInfo, err)
		return
	}

	settings, err := h.obj.GetBucketSettings(r.Context(), bktInfo)
	if err != nil {
		h.logAndSendError(w, "couldn't get bucket settings", reqInfo, err)
		return
	}

	if settings.LifecycleConfiguration == nil {
		h.logAndSendError(w, "lifecycle configuration doesn't exist", reqInfo,
			apiErrors.GetAPIError(apiErrors.ErrNoSuchLifecycleConfiguration))
		return
	}

	if err = api.EncodeToResponse(w, settings.LifecycleConfiguration); err != nil {
		h.logAndSendError(w, "something went wrong", reqInfo, err)
	}
}

func (h *handler) DeleteBucketLifecycleHandler(w http.ResponseWriter, r *http.Request) {
	reqInfo := api.GetReqInfo(r.Context())

	bktInfo, err := h.obj.GetBucketInfo(r.Context(), reqInfo.BucketName)
	if err != nil {
		h.logAndSendError(w, "could not get bucket info", reqInfo, err)
		return
	}
	if err = checkOwner(bktInfo, r.Header.Get(api.AmzExpectedBucketOwner)); err != nil {
		h.logAndSendError(w, "expected owner doesn't match", reqInfo, err)
		return
	}

	if err = h.updateLifecycleConfiguration(r.Context(), bktInfo, nil); err != nil {
		h.logAndSendError(w, "couldn't put bucket settings", reqInfo, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateLifecycleConfiguration(ctx context.Context, bktInfo *data.BucketInfo, lifecycleConf *data.LifecycleConfiguration) error {
	settings, err := h.obj.GetBucketSettings(ctx, bktInfo)
	if err != nil {
		return fmt.Errorf("couldn't get bucket settings: %w", err)
	}

	settings.LifecycleConfiguration = lifecycleConf
	sp := &layer.PutSettingsParams{
		BktInfo:  bktInfo,
		Settings: settings,
	}

	if err = h.obj.PutBucketSettings(ctx, sp); err != nil {
		return fmt.Errorf("couldn't put bucket settings: %w", err)
	}

	return nil
}

func checkLifecycleConfiguration(conf *data.LifecycleConfiguration) error {
	if len(conf.Rules) == 0 {
		return apiErrors.GetAPIError(apiErrors.ErrMalformedXML)
	}
	if len(conf.Rules) > 1000 {
		return fmt.Errorf("you cannot have more than 1000 rules")
	}

	for _, rule := range conf.Rules {
		if rule.Status != enabledValue && rule.Status != disabledValue {
			return apiErrors.GetAPIError(apiErrors.ErrMalformedXML)
		}
	}

	return nil
}
