package handler

import (
	"testing"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/data"
	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
	"github.com/stretchr/testify/require"
)

func TestFilterSubjects(t *testing.T) {
	config := &data.NotificationConfiguration{
		QueueConfigurations: []data.QueueConfiguration{
			{
				ID:       "test1",
				QueueArn: "test1",
				Events:   []string{EventObjectCreated, EventObjectRemovedDelete},
			},
			{
				ID:       "test2",
				QueueArn: "test2",
				Events:   []string{EventObjectTagging},
				Filter: data.Filter{Key: data.Key{FilterRules: []data.FilterRule{
					{Name: "prefix", Value: "dir/"},
					{Name: "suffix", Value: ".png"},
				}}},
			},
		},
	}

	t.Run("no topics because suitable events not found", func(t *testing.T) {
		topics := filterSubjects(config, EventObjectACLPut, "dir/a.png")
		require.Empty(t, topics)
	})

	t.Run("no topics because of not suitable prefix", func(t *testing.T) {
		topics := filterSubjects(config, EventObjectTaggingPut, "dirw/cat.png")
		require.Empty(t, topics)
	})

	t.Run("no topics because of not suitable suffix", func(t *testing.T) {
		topics := filterSubjects(config, EventObjectTaggingPut, "a.jpg")
		require.Empty(t, topics)
	})

	t.Run("filter topics from queue configs without prefix suffix filter and exact event", func(t *testing.T) {
		topics := filterSubjects(config, EventObjectCreatedPut, "dir/a.png")
		require.Contains(t, topics, "test1")
		require.Len(t, topics, 1)
		require.Equal(t, topics["test1"], "test1")
	})

	t.Run("filter topics from queue configs with prefix suffix filter and '*' ending event", func(t *testing.T) {
		topics := filterSubjects(config, EventObjectTaggingPut, "dir/a.png")
		require.Contains(t, topics, "test2")
		require.Len(t, topics, 1)
		require.Equal(t, topics["test2"], "test2")
	})
}

func TestCheckRules(t *testing.T) {
	t.Run("correct rules with prefix and suffix", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "prefix", Value: "asd"},
			{Name: "suffix", Value: "asd"},
		}
		err := checkRules(rules)
		require.NoError(t, err)
	})

	t.Run("correct rules with prefix", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "prefix", Value: "asd"},
		}
		err := checkRules(rules)
		require.NoError(t, err)
	})

	t.Run("correct rules with suffix", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "suffix", Value: "asd"},
		}
		err := checkRules(rules)
		require.NoError(t, err)
	})

	t.Run("incorrect rules with wrong name", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "prefix", Value: "sdf"},
			{Name: "sfx", Value: "asd"},
		}
		err := checkRules(rules)
		require.ErrorIs(t, err, errors.GetAPIError(errors.ErrFilterNameInvalid))
	})

	t.Run("incorrect rules with repeating suffix", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "suffix", Value: "asd"},
			{Name: "suffix", Value: "asdf"},
			{Name: "prefix", Value: "jk"},
		}
		err := checkRules(rules)
		require.ErrorIs(t, err, errors.GetAPIError(errors.ErrFilterNameSuffix))
	})

	t.Run("incorrect rules with repeating prefix", func(t *testing.T) {
		rules := []data.FilterRule{
			{Name: "suffix", Value: "ds"},
			{Name: "prefix", Value: "asd"},
			{Name: "prefix", Value: "asdf"},
		}
		err := checkRules(rules)
		require.ErrorIs(t, err, errors.GetAPIError(errors.ErrFilterNamePrefix))
	})
}
