package layer

import (
	"bytes"
	"context"
	"encoding/xml"
	errorsStd "errors"
	"fmt"
	"io"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"go.uber.org/zap"
)

const wildcard = "*"

var supportedMethods = map[string]struct{}{"GET": {}, "HEAD": {}, "POST": {}, "PUT": {}, "DELETE": {}}

func (n *layer) PutBucketCORS(ctx context.Context, p *PutCORSParams) error {
	var (
		buf  bytes.Buffer
		tee  = io.TeeReader(p.Reader, &buf)
		cors = &data.CORSConfiguration{}
	)

	if err := xml.NewDecoder(tee).Decode(cors); err != nil {
		return fmt.Errorf("xml decode cors: %w", err)
	}

	if cors.CORSRules == nil {
		return errors.GetAPIError(errors.ErrMalformedXML)
	}

	if err := checkCORS(cors); err != nil {
		return err
	}

	prm := PrmObjectCreate{
		Container:    p.BktInfo.CID,
		Creator:      p.BktInfo.Owner,
		Payload:      &buf,
		Filepath:     p.BktInfo.CORSObjectName(),
		CreationTime: TimeNow(ctx),
		CopiesNumber: p.CopiesNumber,
	}

	objID, _, err := n.objectPutAndHash(ctx, prm, p.BktInfo)
	if err != nil {
		return fmt.Errorf("put system object: %w", err)
	}

	objIDToDelete, err := n.treeService.PutBucketCORS(ctx, p.BktInfo, objID)
	objIDToDeleteNotFound := errorsStd.Is(err, ErrNoNodeToRemove)
	if err != nil && !objIDToDeleteNotFound {
		return err
	}

	if !objIDToDeleteNotFound {
		if err = n.objectDelete(ctx, p.BktInfo, objIDToDelete); err != nil {
			n.log.Error("couldn't delete cors object", zap.Error(err),
				zap.String("cnrID", p.BktInfo.CID.EncodeToString()),
				zap.String("bucket name", p.BktInfo.Name),
				zap.String("objID", objIDToDelete.EncodeToString()))
		}
	}

	n.cache.PutCORS(n.Owner(ctx), p.BktInfo, cors)

	return nil
}

func (n *layer) GetBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) (*data.CORSConfiguration, error) {
	cors, err := n.getCORS(ctx, bktInfo)
	if err != nil {
		if errorsStd.Is(err, ErrNodeNotFound) {
			return nil, errors.GetAPIError(errors.ErrNoSuchCORSConfiguration)
		}
		return nil, err
	}

	return cors, nil
}

func (n *layer) DeleteBucketCORS(ctx context.Context, bktInfo *data.BucketInfo) error {
	objID, err := n.treeService.DeleteBucketCORS(ctx, bktInfo)
	objIDNotFound := errorsStd.Is(err, ErrNoNodeToRemove)
	if err != nil && !objIDNotFound {
		return err
	}
	if !objIDNotFound {
		if err = n.objectDelete(ctx, bktInfo, objID); err != nil {
			return err
		}
	}

	n.cache.DeleteCORS(bktInfo)

	return nil
}

func checkCORS(cors *data.CORSConfiguration) error {
	for _, r := range cors.CORSRules {
		for _, m := range r.AllowedMethods {
			if _, ok := supportedMethods[m]; !ok {
				return errors.GetAPIErrorWithError(errors.ErrCORSUnsupportedMethod, fmt.Errorf("unsupported method is %s", m))
			}
		}
		for _, h := range r.ExposeHeaders {
			if h == wildcard {
				return errors.GetAPIError(errors.ErrCORSWildcardExposeHeaders)
			}
		}
	}
	return nil
}
