package data

import "encoding/xml"

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
		Prefix                         string                          `xml:"Prefix" json:"Prefix"`
		Status                         string                          `xml:"Status" json:"Status"`
		Transitions                    []Transition                    `xml:"Transition" json:"Transition"`
	}

	AbortIncompleteMultipartUpload struct {
		DaysAfterInitiation int64 `xml:"DaysAfterInitiation"`
	}

	Expiration struct {
		Date                      string `xml:"Date" json:"Date"`
		Days                      int64  `xml:"Days" json:"Days"`
		ExpiredObjectDeleteMarker bool   `xml:"ExpiredObjectDeleteMarker" json:"ExpiredObjectDeleteMarker"`
	}

	LifecycleRuleFilter struct {
		And                   *LifecycleRuleAndOperator `xml:"And" json:"And"`
		ObjectSizeGreaterThan int64                     `xml:"ObjectSizeGreaterThan" json:"ObjectSizeGreaterThan"`
		ObjectSizeLessThan    int64                     `xml:"ObjectSizeLessThan" json:"ObjectSizeLessThan"`
		Prefix                string                    `xml:"Prefix" json:"Prefix"`
		Tag                   *Tag                      `xml:"Tag" json:"Tag"`
	}

	LifecycleRuleAndOperator struct {
		ObjectSizeGreaterThan int64  `xml:"ObjectSizeGreaterThan" json:"ObjectSizeGreaterThan"`
		ObjectSizeLessThan    int64  `xml:"ObjectSizeLessThan" json:"ObjectSizeLessThan"`
		Prefix                string `xml:"Prefix" json:"Prefix"`
		Tags                  []Tag  `xml:"Tags" json:"Tags"`
	}

	Tag struct {
		Key   string `xml:"Key" json:"Key"`
		Value string `xml:"Value" json:"Value"`
	}

	NoncurrentVersionExpiration struct {
		NewerNoncurrentVersions int64 `xml:"NewerNoncurrentVersions" json:"NewerNoncurrentVersions"`
		NoncurrentDays          int64 `xml:"NoncurrentDays" json:"NoncurrentDays"`
	}

	NoncurrentVersionTransition struct {
		NewerNoncurrentVersions int64  `xml:"NewerNoncurrentVersions" json:"NewerNoncurrentVersions"`
		NoncurrentDays          int64  `xml:"NoncurrentDays" json:"NoncurrentDays"`
		StorageClass            string `xml:"StorageClass" json:"StorageClass"`
	}

	Transition struct {
		Date         string `xml:"Date" json:"Date"`
		Days         int64  `xml:"Days" json:"Days"`
		StorageClass string `xml:"StorageClass" json:"StorageClass"`
	}
)
