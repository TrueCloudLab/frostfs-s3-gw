package handler

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/nspcc-dev/neofs-s3-gw/api"
	"github.com/nspcc-dev/neofs-s3-gw/api/data"
	apiErrors "github.com/nspcc-dev/neofs-s3-gw/api/errors"
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

	if settings.LifecycleConfig == nil || settings.LifecycleConfig.CurrentConfiguration == nil {
		h.logAndSendError(w, "lifecycle configuration doesn't exist", reqInfo,
			apiErrors.GetAPIError(apiErrors.ErrNoSuchLifecycleConfiguration))
		return
	}

	if err = api.EncodeToResponse(w, settings.LifecycleConfig.CurrentConfiguration); err != nil {
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
	// todo consider run as separate goroutine
	if err := h.obj.ScheduleLifecycle(ctx, bktInfo, lifecycleConf); err != nil {
		return fmt.Errorf("couldn't apply lifecycle: %w", err)
	}

	return nil
}

func checkLifecycleConfiguration(conf *data.LifecycleConfiguration) error {
	err := apiErrors.GetAPIError(apiErrors.ErrMalformedXML)

	if len(conf.Rules) == 0 {
		return err
	}
	if len(conf.Rules) > 1000 {
		return fmt.Errorf("you cannot have more than 1000 rules")
	}

	for _, rule := range conf.Rules {
		if rule.Status != enabledValue && rule.Status != disabledValue {
			return err
		}
		if rule.Prefix != nil && rule.Filter != nil {
			return err
		}

		if rule.Filter != nil {
			if rule.Filter.ObjectSizeGreaterThan != nil && *rule.Filter.ObjectSizeGreaterThan < 0 ||
				rule.Filter.ObjectSizeLessThan != nil && *rule.Filter.ObjectSizeLessThan < 0 {
				return err
			}

			if !filterContainsOneOption(rule.Filter) {
				return err
			}
		}

		if !ruleHasAction(rule) {
			return err
		}

		// currently only expiration action is supported
		if rule.Expiration == nil {
			return err
		}
		if rule.Expiration.Days != nil && rule.Expiration.Date != nil ||
			rule.Expiration.Days == nil && rule.Expiration.Date == nil {
			return err
		}
	}

	return nil
}

func filterContainsOneOption(filter *data.LifecycleRuleFilter) bool {
	exactlyOneOption := 0
	if filter.Prefix != nil {
		exactlyOneOption++
	}
	if filter.And != nil {
		exactlyOneOption++
	}
	if filter.Tag != nil {
		exactlyOneOption++
	}

	return exactlyOneOption == 1
}

func ruleHasAction(rule data.Rule) bool {
	return rule.AbortIncompleteMultipartUpload != nil || rule.Expiration != nil ||
		rule.NoncurrentVersionExpiration != nil || len(rule.Transitions) != 0 ||
		len(rule.NoncurrentVersionTransitions) != 0
}
