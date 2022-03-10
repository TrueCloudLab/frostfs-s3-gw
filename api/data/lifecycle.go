package data

import (
	"encoding/xml"
	"strings"
)

type (
	LifecycleConfiguration struct {
		XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LifecycleConfiguration" json:"-"`
		Rules   []Rule   `xml:"Rule" json:"Rule"`
	}

	Rule struct {
		AbortIncompleteMultipartUpload *AbortIncompleteMultipartUpload `xml:"AbortIncompleteMultipartUpload" json:"AbortIncompleteMultipartUpload"`
		Expiration                     *Expiration                     `xml:"Expiration" json:"Expiration"`
		Filter                         *LifecycleRuleFilter            `xml:"Filter" json:"Filter"`
		ID                             string                          `xml:"ID" json:"ID"`
		NoncurrentVersionExpiration    *NoncurrentVersionExpiration    `xml:"NoncurrentVersionExpiration" json:"NoncurrentVersionExpiration"`
		NoncurrentVersionTransitions   []NoncurrentVersionTransition   `xml:"NoncurrentVersionTransition" json:"NoncurrentVersionTransition"`
		Prefix                         *string                         `xml:"Prefix" json:"Prefix"`
		Status                         string                          `xml:"Status" json:"Status"`
		Transitions                    []Transition                    `xml:"Transition" json:"Transition"`
	}

	AbortIncompleteMultipartUpload struct {
		DaysAfterInitiation int64 `xml:"DaysAfterInitiation"`
	}

	Expiration struct {
		Date                      *string `xml:"Date" json:"Date"`
		Days                      *int64  `xml:"Days" json:"Days"`
		ExpiredObjectDeleteMarker bool    `xml:"ExpiredObjectDeleteMarker" json:"ExpiredObjectDeleteMarker"`
	}

	LifecycleRuleFilter struct {
		And                   *LifecycleRuleAndOperator `xml:"And" json:"And"`
		ObjectSizeGreaterThan *int64                    `xml:"ObjectSizeGreaterThan" json:"ObjectSizeGreaterThan"`
		ObjectSizeLessThan    *int64                    `xml:"ObjectSizeLessThan" json:"ObjectSizeLessThan"`
		Prefix                *string                   `xml:"Prefix" json:"Prefix"`
		Tag                   *Tag                      `xml:"Tag" json:"Tag"`
	}

	LifecycleRuleAndOperator struct {
		ObjectSizeGreaterThan *int64  `xml:"ObjectSizeGreaterThan" json:"ObjectSizeGreaterThan"`
		ObjectSizeLessThan    *int64  `xml:"ObjectSizeLessThan" json:"ObjectSizeLessThan"`
		Prefix                *string `xml:"Prefix" json:"Prefix"`
		Tags                  []Tag   `xml:"Tags" json:"Tags"`
	}

	Tag struct {
		Key   string `xml:"Key" json:"Key"`
		Value string `xml:"Value" json:"Value"`
	}

	NoncurrentVersionExpiration struct {
		NewerNoncurrentVersions *int64 `xml:"NewerNoncurrentVersions" json:"NewerNoncurrentVersions"`
		NoncurrentDays          *int64 `xml:"NoncurrentDays" json:"NoncurrentDays"`
	}

	NoncurrentVersionTransition struct {
		NewerNoncurrentVersions *int64 `xml:"NewerNoncurrentVersions" json:"NewerNoncurrentVersions"`
		NoncurrentDays          *int64 `xml:"NoncurrentDays" json:"NoncurrentDays"`
		StorageClass            string `xml:"StorageClass" json:"StorageClass"`
	}

	Transition struct {
		Date         *string `xml:"Date" json:"Date"`
		Days         *int64  `xml:"Days" json:"Days"`
		StorageClass string  `xml:"StorageClass" json:"StorageClass"`
	}

	ExpirationObject struct {
		Expiration        *Expiration
		RuleID            string
		LifecycleConfigID string
	}
)

func (r *Rule) RealPrefix() string {
	if r.Filter == nil {
		if r.Prefix != nil {
			return *r.Prefix
		}
		return ""
	}

	if r.Filter.And == nil {
		if r.Filter.Prefix != nil {
			return *r.Filter.Prefix
		}
		return ""
	}

	if r.Filter.And.Prefix != nil {
		return *r.Filter.And.Prefix
	}
	return ""
}

func (r *Rule) NeedTags() bool {
	if r.Filter == nil {
		return false
	}

	if r.Filter.And == nil {
		return r.Filter.Tag != nil
	}

	return len(r.Filter.And.Tags) != 0
}

func (r *Rule) MatchObject(obj *ObjectInfo, tags map[string]string) bool {
	if r.Filter == nil {
		if r.Prefix != nil {
			return strings.HasPrefix(obj.Name, *r.Prefix)
		}
		return true
	}

	if r.Filter.And == nil {
		if r.Filter.Prefix != nil && !strings.HasPrefix(obj.Name, *r.Filter.Prefix) {
			return false
		}

		if r.Filter.Tag != nil {
			if tags == nil {
				return false
			}
			if tagVal := tags[r.Filter.Tag.Key]; tagVal != r.Filter.Tag.Value {
				return false
			}
		}

		if r.Filter.ObjectSizeLessThan != nil && *r.Filter.ObjectSizeLessThan > 0 && obj.Size >= *r.Filter.ObjectSizeLessThan {
			return false
		}

		if r.Filter.ObjectSizeGreaterThan != nil && obj.Size <= *r.Filter.ObjectSizeGreaterThan {
			return false
		}

		return true
	}

	if r.Filter.And.Prefix != nil && !strings.HasPrefix(obj.Name, *r.Filter.And.Prefix) {
		return false
	}

	if len(r.Filter.And.Tags) != 0 {
		if tags == nil {
			return false
		}

		for _, tag := range r.Filter.And.Tags {
			if tagVal := tags[tag.Key]; tagVal != tag.Value {
				return false
			}
		}
	}

	if r.Filter.And.ObjectSizeLessThan != nil && obj.Size >= *r.Filter.And.ObjectSizeLessThan {
		return false
	}

	if r.Filter.And.ObjectSizeGreaterThan != nil && obj.Size <= *r.Filter.And.ObjectSizeGreaterThan {
		return false
	}

	return true
}
