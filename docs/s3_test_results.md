# S3 compatibility test results

FrostFS Node: v0.30.0
FrostFS S3 Gateway: v0.22.0-10-g87f6681

To update this file using tests result, run:

```sh
./updateTestsResult.sh ceph_tests_result.txt
```

## CopyObject

Compatibility: 16/16/17 out of 17

|     | Test                                                                      | s3-gw | minio | aws s3 |
|-----|---------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_copy_object_ifmatch_good            | ok    | ok    | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_copy_object_ifmatch_failed          | ok    | ok    | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_copy_object_ifnonematch_good        | ok    | ok    | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_copy_object_ifnonematch_failed      | ok    | ok    | ok     |
| 5   | s3tests_boto3.functional.test_s3.test_object_copy_zero_size               | ok    | ok    | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_object_copy_same_bucket             | ok    | ok    | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_object_copy_verify_contenttype      | ok    | ok    | ok     |
| 8   | s3tests_boto3.functional.test_s3.test_object_copy_to_itself               | ok    | ok    | ok     |
| 9   | s3tests_boto3.functional.test_s3.test_object_copy_to_itself_with_metadata | ok    | ok    | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_object_copy_diff_bucket             | ok    | ok    | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_object_copy_not_owned_bucket        | ok    | FAIL  | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_object_copy_not_owned_object_bucket | ERROR | ok    | ok     |
| 13  | s3tests_boto3.functional.test_s3.test_object_copy_canned_acl              | ok    | ok    | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_object_copy_retaining_metadata      | ok    | ok    | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_object_copy_replacing_metadata      | ok    | ok    | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_object_copy_bucket_not_found        | ok    | ok    | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_object_copy_key_not_found           | ok    | ok    | ok     |

## GetObject

Compatibility: 30/25/29 out of 33

|     | Test                                                                                     | s3-gw | minio | aws s3 |
|-----|------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_get_object_ifmatch_good                            | ok    | ok    | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_get_object_ifmatch_failed                          | ok    | ok    | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_get_object_ifnonematch_good                        | ok    | ok    | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_get_object_ifnonematch_failed                      | ok    | ok    | ok     |
| 5   | s3tests_boto3.functional.test_s3.test_get_object_ifmodifiedsince_good                    | ok    | ok    | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_get_object_ifmodifiedsince_failed                  | ok    | ok    | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_get_object_ifunmodifiedsince_good                  | ok    | ok    | ok     |
| 8   | s3tests_boto3.functional.test_s3.test_get_object_ifunmodifiedsince_failed                | ok    | ok    | ok     |
| 9   | s3tests_boto3.functional.test_s3.test_object_read_not_exist                              | ok    | ok    | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_object_requestid_matches_header_on_error           | ok    | ok    | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_object_head_zero_bytes                             | ok    | ok    | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_object_read_unreadable                             | FAIL  | FAIL  | FAIL   |
| 13  | s3tests_boto3.functional.test_s3.test_ranged_request_response_code                       | ok    | ok    | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_ranged_big_request_response_code                   | ok    | ok    | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_ranged_request_skip_leading_bytes_response_code    | ok    | ok    | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_ranged_request_return_trailing_bytes_response_code | ok    | ok    | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_ranged_request_invalid_range                       | ok    | ok    | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_ranged_request_empty_object                        | ok    | ok    | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_atomic_read_1mb                                    | ok    | ok    | ok     |
| 20  | s3tests_boto3.functional.test_s3.test_atomic_read_4mb                                    | ok    | ok    | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_atomic_read_8mb                                    | ok    | ok    | ok     |
| 22  | s3tests_boto3.functional.test_s3.test_object_raw_get_x_amz_expires_not_expired           | ok    | ok    | ok     |
| 23  | s3tests_boto3.functional.test_s3.test_object_raw_get_x_amz_expires_out_range_zero        | ok    | ok    | ok     |
| 24  | s3tests_boto3.functional.test_s3.test_object_raw_get_x_amz_expires_out_max_range         | FAIL  | FAIL  | FAIL   |
| 25  | s3tests_boto3.functional.test_s3.test_object_raw_get_x_amz_expires_out_positive_range    | ok    | FAIL  | FAIL   |
| 26  | s3tests_boto3.functional.test_s3.test_object_raw_get                                     | ok    | ERROR | ok     |
| 27  | s3tests_boto3.functional.test_s3.test_object_raw_get_bucket_gone                         | ok    | FAIL  | ok     |
| 28  | s3tests_boto3.functional.test_s3.test_object_delete_key_bucket_gone                      | ok    | FAIL  | ok     |
| 29  | s3tests_boto3.functional.test_s3.test_object_header_acl_grants                           | ERROR | FAIL  | ERROR  |
| 30  | s3tests_boto3.functional.test_s3.test_object_raw_get_object_gone                         | ok    | FAIL  | ok     |
| 31  | s3tests_boto3.functional.test_s3.test_object_raw_get_object_acl                          | ok    | ok    | ok     |
| 32  | s3tests_boto3.functional.test_s3.test_object_raw_authenticated                           | ok    | ok    | ok     |
| 33  | s3tests_boto3.functional.test_s3.test_object_raw_response_headers                        | ok    | ok    | ok     |

## PutObject

Compatibility: 33/43/37 out of 64

|    | Test                                                                                           | s3-gw       | minio | aws s3 |
|----|------------------------------------------------------------------------------------------------|-------------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_put_object_ifmatch_good                                  | ok          | ok    | ERROR  |
| 2  | s3tests_boto3.functional.test_s3.test_put_object_ifmatch_failed                                | FAIL        | FAIL  | FAIL   |
| 3  | s3tests_boto3.functional.test_s3.test_put_object_ifmatch_overwrite_existed_good                | ok          | ok    | ERROR  |
| 4  | s3tests_boto3.functional.test_s3.test_put_object_ifmatch_nonexisted_failed                     | FAIL        | FAIL  | FAIL   |
| 5  | s3tests_boto3.functional.test_s3.test_put_object_ifnonmatch_good                               | ok          | ok    | ERROR  |
| 6  | s3tests_boto3.functional.test_s3.test_put_object_ifnonmatch_failed                             | FAIL        | FAIL  | FAIL   |
| 7  | s3tests_boto3.functional.test_s3.test_put_object_ifnonmatch_nonexisted_good                    | ok          | ok    | ERROR  |
| 8  | s3tests_boto3.functional.test_s3.test_put_object_ifnonmatch_overwrite_existed_failed           | FAIL        | FAIL  | FAIL   |
| 9  | s3tests_boto3.functional.test_headers.test_object_create_bad_md5_invalid_short                 | UNSUPPORTED | ok    | ok     |
| 10 | s3tests_boto3.functional.test_headers.test_object_create_bad_md5_bad                           | UNSUPPORTED | ok    | ok     |
| 11 | s3tests_boto3.functional.test_headers.test_object_create_bad_md5_empty                         | UNSUPPORTED | ok    | ok     |
| 12 | s3tests_boto3.functional.test_headers.test_object_create_bad_md5_none                          | ok          | ok    | ok     |
| 13 | s3tests_boto3.functional.test_headers.test_object_create_bad_expect_mismatch                   | ERROR       | ERROR | ok     |
| 14 | s3tests_boto3.functional.test_headers.test_object_create_bad_expect_empty                      | ok          | ok    | ok     |
| 15 | s3tests_boto3.functional.test_headers.test_object_create_bad_expect_none                       | ok          | ok    | ok     |
| 16 | s3tests_boto3.functional.test_headers.test_object_create_bad_contentlength_empty               | FAIL        | FAIL  | ok     |
| 17 | s3tests_boto3.functional.test_headers.test_object_create_bad_contentlength_negative            | ok          | ok    | ok     |
| 18 | s3tests_boto3.functional.test_headers.test_object_create_bad_contentlength_none                | FAIL        | FAIL  | FAIL   |
| 19 | s3tests_boto3.functional.test_headers.test_object_create_bad_contentlength_mismatch_above      | ERROR       | ERROR | ERROR  |
| 20 | s3tests_boto3.functional.test_headers.test_object_create_bad_contenttype_invalid               | ok          | ok    | ok     |
| 21 | s3tests_boto3.functional.test_headers.test_object_create_bad_contenttype_empty                 | ok          | ok    | ok     |
| 22 | s3tests_boto3.functional.test_headers.test_object_create_bad_contenttype_none                  | ok          | ok    | ok     |
| 23 | s3tests_boto3.functional.test_headers.test_object_create_bad_authorization_empty               | FAIL        | FAIL  | FAIL   |
| 24 | s3tests_boto3.functional.test_headers.test_object_create_date_and_amz_date                     | ERROR       | ERROR | ERROR  |
| 25 | s3tests_boto3.functional.test_headers.test_object_create_amz_date_and_no_date                  | ERROR       | ERROR | ERROR  |
| 26 | s3tests_boto3.functional.test_headers.test_object_create_bad_authorization_none                | FAIL        | FAIL  | FAIL   |
| 27 | s3tests_boto3.functional.test_headers.test_object_create_bad_md5_invalid_garbage_aws2          | UNSUPPORTED | ok     | ok     |
| 28 | s3tests_boto3.functional.test_headers.test_object_create_bad_contentlength_mismatch_below_aws2 | FAIL        | FAIL  | ok     |
| 29 | s3tests_boto3.functional.test_headers.test_object_create_bad_authorization_incorrect_aws2      | FAIL        | FAIL  | FAIL   |
| 30 | s3tests_boto3.functional.test_headers.test_object_create_bad_authorization_invalid_aws2        | FAIL        | FAIL  | FAIL   |
| 31 | s3tests_boto3.functional.test_headers.test_object_create_bad_ua_empty_aws2                     | ERROR       | ok    | ok     |
| 32 | s3tests_boto3.functional.test_headers.test_object_create_bad_ua_none_aws2                      | ERROR       | ok    | ok     |
| 33 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_invalid_aws2                 | FAIL        | FAIL  | ok     |
| 34 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_empty_aws2                   | FAIL        | FAIL  | ok     |
| 35 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_none_aws2                    | FAIL        | FAIL  | FAIL   |
| 36 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_before_today_aws2            | FAIL        | ok    | ok     |
| 37 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_before_epoch_aws2            | FAIL        | FAIL  | ok     |
| 38 | s3tests_boto3.functional.test_headers.test_object_create_bad_date_after_end_aws2               | FAIL        | ok    | ok     |
| 39 | s3tests_boto3.functional.test_s3.test_object_anon_put                                          | ok          | ok    | ok     |
| 40 | s3tests_boto3.functional.test_s3.test_object_put_authenticated                                 | ok          | ok    | ok     |
| 41 | s3tests_boto3.functional.test_s3.test_object_raw_put_authenticated_expired                     | ok          | FAIL  | FAIL   |
| 42 | s3tests_boto3.functional.test_s3.test_object_write_file                                        | ok          | ok    | ok     |
| 43 | s3tests_boto3.functional.test_s3.test_object_write_check_etag                                  | FAIL        | ok    | ok     |
| 44 | s3tests_boto3.functional.test_s3.test_object_write_cache_control                               | ok          | ok    | ok     |
| 45 | s3tests_boto3.functional.test_s3.test_object_write_expires                                     | ok          | ok    | ok     |
| 46 | s3tests_boto3.functional.test_s3.test_object_write_read_update_read_delete                     | ok          | ok    | ok     |
| 47 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_none_to_good                     | ok          | ok    | ok     |
| 48 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_none_to_empty                    | ERROR       | ok    | ok     |
| 49 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_overwrite_to_empty               | ERROR       | ok    | ok     |
| 50 | s3tests_boto3.functional.test_s3.test_object_set_get_non_utf8_metadata                         | ok          | ok    | FAIL   |
| 51 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_empty_to_unreadable_prefix       | ok          | ok    | FAIL   |
| 52 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_empty_to_unreadable_suffix       | ok          | ok    | FAIL   |
| 53 | s3tests_boto3.functional.test_s3.test_object_set_get_metadata_empty_to_unreadable_infix        | ok          | ok    | FAIL   |
| 54 | s3tests_boto3.functional.test_s3.test_object_metadata_replaced_on_put                          | ok          | ok    | ok     |
| 55 | s3tests_boto3.functional.test_s3.test_object_write_to_nonexist_bucket                          | ok          | ok    | ok     |
| 56 | s3tests_boto3.functional.test_s3.test_atomic_write_1mb                                         | ok          | ok    | ok     |
| 57 | s3tests_boto3.functional.test_s3.test_atomic_write_4mb                                         | ok          | ok    | ok     |
| 58 | s3tests_boto3.functional.test_s3.test_atomic_write_8mb                                         | ok          | ok    | ok     |
| 59 | s3tests_boto3.functional.test_s3.test_atomic_dual_write_1mb                                    | ok          | ok    | ERROR  |
| 60 | s3tests_boto3.functional.test_s3.test_atomic_dual_write_4mb                                    | ok          | ok    | ERROR  |
| 61 | s3tests_boto3.functional.test_s3.test_atomic_dual_write_8mb                                    | ok          | ok    | ERROR  |
| 62 | s3tests_boto3.functional.test_s3.test_atomic_conditional_write_1mb                             | ok          | ok    | ERROR  |
| 63 | s3tests_boto3.functional.test_s3.test_atomic_dual_conditional_write_1mb                        | FAIL        | FAIL  | FAIL   |
| 64 | s3tests_boto3.functional.test_s3.test_atomic_write_bucket_gone                                 | ok          | ok    | ok     |

## PostObject

Compatibility: 12/12/32 out of 33
This group is not explicitly supported by s3-gw, but some tests may pass.

|     | Test                                                                                     | s3-gw | minio | aws s3 |
|-----|------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_post_object_anonymous_request                      | FAIL  | FAIL  | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_post_object_authenticated_request                  | FAIL  | FAIL  | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_post_object_authenticated_no_content_type          | FAIL  | FAIL  | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_post_object_authenticated_request_bad_access_key   | FAIL  | FAIL  | ok     |
| 5   | s3tests_boto3.functional.test_s3.test_post_object_set_success_code                       | FAIL  | FAIL  | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_post_object_set_invalid_success_code               | FAIL  | FAIL  | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_post_object_upload_larger_than_chunk               | FAIL  | FAIL  | ok     |
| 8   | s3tests_boto3.functional.test_s3.test_post_object_set_key_from_filename                  | FAIL  | FAIL  | ok     |
| 9   | s3tests_boto3.functional.test_s3.test_post_object_ignored_header                         | FAIL  | FAIL  | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_post_object_case_insensitive_condition_fields      | FAIL  | FAIL  | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_post_object_escaped_field_values                   | FAIL  | FAIL  | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_post_object_success_redirect_action                | FAIL  | FAIL  | ok     |
| 13  | s3tests_boto3.functional.test_s3.test_post_object_invalid_signature                      | FAIL  | FAIL  | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_post_object_invalid_access_key                     | FAIL  | FAIL  | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_post_object_invalid_date_format                    | ok    | ok    | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_post_object_no_key_specified                       | ok    | ok    | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_post_object_missing_signature                      | ok    | ok    | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_post_object_missing_policy_condition               | FAIL  | FAIL  | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_post_object_user_specified_header                  | FAIL  | FAIL  | ok     |
| 20  | s3tests_boto3.functional.test_s3.test_post_object_request_missing_policy_specified_field | FAIL  | FAIL  | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_post_object_condition_is_case_sensitive            | ok    | ok    | ok     |
| 22  | s3tests_boto3.functional.test_s3.test_post_object_expires_is_case_sensitive              | ok    | ok    | ok     |
| 23  | s3tests_boto3.functional.test_s3.test_post_object_expired_policy                         | FAIL  | FAIL  | ok     |
| 24  | s3tests_boto3.functional.test_s3.test_post_object_invalid_request_field_value            | FAIL  | FAIL  | ok     |
| 25  | s3tests_boto3.functional.test_s3.test_post_object_missing_expires_condition              | ok    | ok    | ok     |
| 26  | s3tests_boto3.functional.test_s3.test_post_object_missing_conditions_list                | ok    | ok    | ok     |
| 27  | s3tests_boto3.functional.test_s3.test_post_object_upload_size_limit_exceeded             | ok    | ok    | ok     |
| 28  | s3tests_boto3.functional.test_s3.test_post_object_missing_content_length_argument        | ok    | ok    | ok     |
| 29  | s3tests_boto3.functional.test_s3.test_post_object_invalid_content_length_argument        | ok    | ok    | ok     |
| 30  | s3tests_boto3.functional.test_s3.test_post_object_upload_size_below_minimum              | ok    | ok    | ok     |
| 31  | s3tests_boto3.functional.test_s3.test_post_object_empty_conditions                       | ok    | ok    | ok     |
| 32  | s3tests_boto3.functional.test_s3.test_post_object_tags_anonymous_request                 | FAIL  | FAIL  | FAIL   |
| 33  | s3tests_boto3.functional.test_s3.test_post_object_tags_authenticated_request             | FAIL  | FAIL  | ok     |

## ListObjects

Compatibility: 81/63/76 out of 84

|     | Test                                                                                            | s3-gw | minio | aws s3 |
|-----|-------------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_bucket_list_empty                                         | ok    | ok    | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_bucket_list_distinct                                      | ok    | ok    | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_bucket_list_many                                          | ok    | ok    | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_bucket_listv2_many                                        | ok    | ok    | ok     |
| 5   | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_basic                               | ok    | ok    | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_basic                             | ok    | ok    | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_bucket_listv2_encoding_basic                              | ok    | FAIL  | FAIL   |
| 8   | s3tests_boto3.functional.test_s3.test_bucket_list_encoding_basic                                | ok    | FAIL  | FAIL   |
| 9   | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_prefix                              | ok    | FAIL  | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_prefix                            | ok    | ok    | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_prefix_ends_with_delimiter        | ok    | ok    | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_prefix_ends_with_delimiter          | ok    | ok    | ok     |
| 13  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_alt                                 | ok    | ok    | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_alt                               | ok    | ok    | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_prefix_underscore                   | ok    | FAIL  | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_prefix_underscore                 | ok    | ok    | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_percentage                          | ok    | ok    | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_percentage                        | ok    | ok    | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_whitespace                          | ok    | ok    | ok     |
| 20  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_whitespace                        | ok    | ok    | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_dot                                 | ok    | ERROR | ok     |
| 22  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_dot                               | ok    | ERROR | ok     |
| 23  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_unreadable                          | ok    | ok    | ok     |
| 24  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_unreadable                        | ok    | ok    | ok     |
| 25  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_empty                               | ok    | FAIL  | ok     |
| 26  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_empty                             | ok    | FAIL  | ok     |
| 27  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_none                                | ok    | FAIL  | ok     |
| 28  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_none                              | ok    | FAIL  | ok     |
| 29  | s3tests_boto3.functional.test_s3.test_bucket_listv2_fetchowner_notempty                         | ok    | ok    | ok     |
| 30  | s3tests_boto3.functional.test_s3.test_bucket_listv2_fetchowner_defaultempty                     | ok    | FAIL  | ok     |
| 31  | s3tests_boto3.functional.test_s3.test_bucket_listv2_fetchowner_empty                            | ok    | FAIL  | ok     |
| 32  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_not_exist                           | ok    | ok    | ok     |
| 33  | s3tests_boto3.functional.test_s3.test_bucket_listv2_delimiter_not_exist                         | ok    | ok    | ok     |
| 35  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_basic                                  | ok    | ok    | ok     |
| 36  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_basic                                | ok    | ok    | ok     |
| 37  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_alt                                    | ok    | ok    | ok     |
| 38  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_alt                                  | ok    | ok    | ok     |
| 39  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_empty                                  | ok    | ok    | ok     |
| 40  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_empty                                | ok    | ok    | ok     |
| 41  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_none                                   | ok    | ok    | ok     |
| 42  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_none                                 | ok    | ok    | ok     |
| 43  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_not_exist                              | ok    | ok    | ok     |
| 44  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_not_exist                            | ok    | ok    | ok     |
| 45  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_unreadable                             | ok    | FAIL  | FAIL   |
| 46  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_unreadable                           | ok    | ok    | ok     |
| 47  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_delimiter_basic                        | ok    | ok    | ok     |
| 48  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_delimiter_basic                      | ok    | ok    | ok     |
| 49  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_delimiter_alt                          | ok    | ok    | ok     |
| 50  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_delimiter_alt                        | ok    | ok    | ok     |
| 51  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_delimiter_prefix_not_exist             | ok    | FAIL  | ok     |
| 52  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_delimiter_prefix_not_exist           | ok    | FAIL  | ok     |
| 53  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_delimiter_delimiter_not_exist          | ok    | ok    | ok     |
| 54  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_delimiter_delimiter_not_exist        | ok    | ok    | ok     |
| 55  | s3tests_boto3.functional.test_s3.test_bucket_list_prefix_delimiter_prefix_delimiter_not_exist   | ok    | ok    | ok     |
| 56  | s3tests_boto3.functional.test_s3.test_bucket_listv2_prefix_delimiter_prefix_delimiter_not_exist | ok    | ok    | ok     |
| 57  | s3tests_boto3.functional.test_s3.test_bucket_list_maxkeys_one                                   | ok    | ok    | ok     |
| 58  | s3tests_boto3.functional.test_s3.test_bucket_listv2_maxkeys_one                                 | ok    | ok    | ok     |
| 59  | s3tests_boto3.functional.test_s3.test_bucket_list_maxkeys_zero                                  | ok    | ok    | ok     |
| 60  | s3tests_boto3.functional.test_s3.test_bucket_listv2_maxkeys_zero                                | ok    | ok    | ok     |
| 61  | s3tests_boto3.functional.test_s3.test_bucket_list_maxkeys_none                                  | ok    | ok    | ok     |
| 62  | s3tests_boto3.functional.test_s3.test_bucket_listv2_maxkeys_none                                | ok    | ok    | ok     |
| 63  | s3tests_boto3.functional.test_s3.test_bucket_list_unordered                                     | FAIL  | FAIL  | FAIL   |
| 64  | s3tests_boto3.functional.test_s3.test_bucket_listv2_unordered                                   | FAIL  | FAIL  | FAIL   |
| 34  | s3tests_boto3.functional.test_s3.test_bucket_list_delimiter_not_skip_special                    | ok    | ok    |        |
| 65  | s3tests_boto3.functional.test_s3.test_bucket_list_maxkeys_invalid                               | ok    | ok    | ok     |
| 66  | s3tests_boto3.functional.test_s3.test_bucket_list_marker_none                                   | ok    | ok    | ERROR  |
| 67  | s3tests_boto3.functional.test_s3.test_bucket_list_marker_empty                                  | ok    | ok    | ok     |
| 68  | s3tests_boto3.functional.test_s3.test_bucket_listv2_continuationtoken_empty                     | ERROR | ERROR | ERROR  |
| 69  | s3tests_boto3.functional.test_s3.test_bucket_listv2_continuationtoken                           | ok    | ok    | ok     |
| 70  | s3tests_boto3.functional.test_s3.test_bucket_listv2_both_continuationtoken_startafter           | ok    | ok    | ERROR  |
| 71  | s3tests_boto3.functional.test_s3.test_bucket_list_marker_unreadable                             | ok    | ok    | ok     |
| 72  | s3tests_boto3.functional.test_s3.test_bucket_listv2_startafter_unreadable                       | ok    | ok    | ok     |
| 73  | s3tests_boto3.functional.test_s3.test_bucket_list_marker_not_in_list                            | ok    | ok    | ok     |
| 74  | s3tests_boto3.functional.test_s3.test_bucket_listv2_startafter_not_in_list                      | ok    | ok    | ok     |
| 75  | s3tests_boto3.functional.test_s3.test_bucket_list_marker_after_list                             | ok    | ok    | ok     |
| 76  | s3tests_boto3.functional.test_s3.test_bucket_listv2_startafter_after_list                       | ok    | ok    | ok     |
| 77  | s3tests_boto3.functional.test_s3.test_bucket_list_return_data                                   | ok    | FAIL  | ok     |
| 78  | s3tests_boto3.functional.test_s3.test_bucket_list_objects_anonymous                             | ok    | ERROR | ok     |
| 79  | s3tests_boto3.functional.test_s3.test_bucket_listv2_objects_anonymous                           | ok    | ERROR | ok     |
| 80  | s3tests_boto3.functional.test_s3.test_bucket_list_objects_anonymous_fail                        | ok    | ok    | ok     |
| 81  | s3tests_boto3.functional.test_s3.test_bucket_listv2_objects_anonymous_fail                      | ok    | ok    | ok     |
| 82  | s3tests_boto3.functional.test_s3.test_bucket_list_special_prefix                                | ok    | ok    | ok     |
| 83  | s3tests_boto3.functional.test_s3.test_bucket_list_long_name                                     | ok    | ok    | ok     |
| 84  | s3tests_boto3.functional.test_s3.test_basic_key_count                                           | ok    | ok    | ok     |

## Object ACL

Compatibility:  4/3/10 out of 19

|     | Test                                                                            | s3-gw | minio | aws s3 |
|-----|---------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_headers.test_object_acl_create_contentlength_none | ok    | ERROR | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_object_anon_put_write_access              | ok    | ERROR | ERROR  |
| 3   | s3tests_boto3.functional.test_s3.test_object_acl_default                        | FAIL  | FAIL  | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_object_acl_canned_during_create           | FAIL  | FAIL  | FAIL   |
| 5   | s3tests_boto3.functional.test_s3.test_object_acl_canned                         | FAIL  | FAIL  | FAIL   |
| 6   | s3tests_boto3.functional.test_s3.test_object_acl_canned_publicreadwrite         | FAIL  | FAIL  | FAIL   |
| 7   | s3tests_boto3.functional.test_s3.test_object_acl_canned_authenticatedread       | FAIL  | FAIL  | FAIL   |
| 8   | s3tests_boto3.functional.test_s3.test_object_acl_canned_bucketownerread         | ERROR | ERROR | ERROR  |
| 9   | s3tests_boto3.functional.test_s3.test_object_acl_canned_bucketownerfullcontrol  | ERROR | ERROR | ERROR  |
| 10  | s3tests_boto3.functional.test_s3.test_object_acl_full_control_verify_owner      | ERROR | ERROR | ERROR  |
| 11  | s3tests_boto3.functional.test_s3.test_object_acl_full_control_verify_attributes | ERROR | ok    | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_object_acl                                | ERROR | FAIL  | FAIL   |
| 13  | s3tests_boto3.functional.test_s3.test_object_acl_write                          | ERROR | ERROR | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_object_acl_writeacp                       | ERROR | ERROR | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_object_acl_read                           | ERROR | ERROR | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_object_acl_readacp                        | ERROR | ERROR | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_object_raw_authenticated_object_acl       | ok    | ok    | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_object_raw_authenticated_object_gone      | ok    | ok    | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_object_raw_get_bucket_acl                 | ERROR | ERROR | ok     |

## Locking

Compatibility:  4/5/29 out of 29

|     | Test                                                                                           | s3-gw | minio | aws s3 |
|-----|------------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock                                 | ok    | ERROR | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock_invalid_bucket                  | ok    | FAIL  | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock_with_days_and_years             | FAIL  | ERROR | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock_invalid_days                    | FAIL  | ERROR | FAIL   |
| 5   | s3tests_boto3.functional.test_s3.test_object_lock_uploading_obj                                | ERROR | ERROR | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock_invalid_years                   | FAIL  | ERROR | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_lock_invalid_status                  | FAIL  | ERROR | ok     |
| 8   | s3tests_boto3.functional.test_s3.test_object_lock_suspend_versioning                           | FAIL  | ERROR | ok     |
| 9   | s3tests_boto3.functional.test_s3.test_object_lock_get_obj_lock                                 | FAIL  | ERROR | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_object_lock_get_obj_lock_invalid_bucket                  | ok    | ok    | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention                            | ERROR | ERROR | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_invalid_bucket             | FAIL  | ok    | ok     |
| 13  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_invalid_mode               | FAIL  | ERROR | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_object_lock_get_obj_retention                            | ERROR | ERROR | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_object_lock_get_obj_retention_invalid_bucket             | FAIL  | ok    | ok     |
| 16  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_versionid                  | ERROR | ERROR | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_override_default_retention | ERROR | ERROR | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_increase_period            | ERROR | ERROR | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_shorten_period             | FAIL  | ERROR | ok     |
| 20  | s3tests_boto3.functional.test_s3.test_object_lock_put_obj_retention_shorten_period_bypass      | ERROR | ERROR | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_object_lock_delete_object_with_retention                 | ERROR | ERROR | ok     |
| 22  | s3tests_boto3.functional.test_s3.test_object_lock_put_legal_hold                               | ERROR | ERROR | ok     |
| 23  | s3tests_boto3.functional.test_s3.test_object_lock_put_legal_hold_invalid_bucket                | FAIL  | ok    | ok     |
| 24  | s3tests_boto3.functional.test_s3.test_object_lock_put_legal_hold_invalid_status                | FAIL  | ERROR | ok     |
| 25  | s3tests_boto3.functional.test_s3.test_object_lock_get_legal_hold                               | ERROR | ERROR | ok     |
| 26  | s3tests_boto3.functional.test_s3.test_object_lock_get_legal_hold_invalid_bucket                | FAIL  | ok    | ok     |
| 27  | s3tests_boto3.functional.test_s3.test_object_lock_delete_object_with_legal_hold_on             | ERROR | ERROR | ok     |
| 28  | s3tests_boto3.functional.test_s3.test_object_lock_delete_object_with_legal_hold_off            | ok    | ERROR | ok     |
| 29  | s3tests_boto3.functional.test_s3.test_object_lock_get_obj_metadata                             | ERROR | ERROR | ok     |

## Multipart

Compatibility: 19/15/19 out of 22

|    | Test                                                                             | s3-gw | minio | aws s3 |
|----|----------------------------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_multipart_upload_empty                     | ok    | FAIL  | FAIL   |
| 2  | s3tests_boto3.functional.test_s3.test_multipart_upload_small                     | ERROR | ERROR | ok     |
| 3  | s3tests_boto3.functional.test_s3.test_multipart_copy_small                       | ok    | ok    | ok     |
| 4  | s3tests_boto3.functional.test_s3.test_multipart_copy_invalid_range               | ok    | FAIL  | FAIL   |
| 5  | s3tests_boto3.functional.test_s3.test_multipart_copy_improper_range              | ok    | ok    | ok     |
| 6  | s3tests_boto3.functional.test_s3.test_multipart_copy_without_range               | ok    | ok    | ok     |
| 7  | s3tests_boto3.functional.test_s3.test_multipart_copy_special_names               | ok    | ok    | ok     |
| 8  | s3tests_boto3.functional.test_s3.test_multipart_upload                           | ERROR | ERROR | ok     |
| 9  | s3tests_boto3.functional.test_s3.test_multipart_upload_resend_part               | ok    | ok    | ok     |
| 10 | s3tests_boto3.functional.test_s3.test_multipart_upload_multiple_sizes            | ok    | ok    | ok     |
| 11 | s3tests_boto3.functional.test_s3.test_multipart_copy_multiple_sizes              | ok    | ok    | ok     |
| 12 | s3tests_boto3.functional.test_s3.test_multipart_upload_size_too_small            | ok    | ok    | ok     |
| 13 | s3tests_boto3.functional.test_s3.test_multipart_upload_contents                  | ok    | ok    | ok     |
| 14 | s3tests_boto3.functional.test_s3.test_multipart_upload_overwrite_existing_object | ok    | ok    | ok     |
| 15 | s3tests_boto3.functional.test_s3.test_abort_multipart_upload                     | ok    | ok    | ok     |
| 16 | s3tests_boto3.functional.test_s3.test_abort_multipart_upload_not_found           | ok    | ok    | ok     |
| 17 | s3tests_boto3.functional.test_s3.test_list_multipart_upload                      | ok    | ERROR | ok     |
| 18 | s3tests_boto3.functional.test_s3.test_multipart_upload_missing_part              | ok    | ok    | ok     |
| 19 | s3tests_boto3.functional.test_s3.test_multipart_upload_incorrect_etag            | ok    | ok    | ok     |
| 20 | s3tests_boto3.functional.test_s3.test_multipart_resend_first_finishes_last       | ERROR | ERROR | ERROR  |
| 21 | s3tests_boto3.functional.test_s3.test_atomic_multipart_upload_write              | ok    | ok    | ok     |
| 22 | s3tests_boto3.functional.test_s3.test_multipart_copy_versioned                   | ok    | ERROR | ok     |

Comments: in [PR](https://github.com/nspcc-dev/s3-tests/pull/5)

## Tagging

Compatibility: 9/6/8 out of 11

|    | Test                                                       | s3-gw | minio | aws s3 |
|----|------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_set_bucket_tagging   | FAIL  | FAIL  | FAIL   |
| 2  | s3tests_boto3.functional.test_s3.test_get_obj_tagging      | ok    | ok    | ok     |
| 3  | s3tests_boto3.functional.test_s3.test_get_obj_head_tagging | ok    | ok    | ok     |
| 4  | s3tests_boto3.functional.test_s3.test_put_max_tags         | ok    | FAIL  | ok     |
| 5  | s3tests_boto3.functional.test_s3.test_put_excess_tags      | FAIL  | FAIL  | FAIL   |
| 6  | s3tests_boto3.functional.test_s3.test_put_max_kvsize_tags  | ok    | ok    | ok     |
| 7  | s3tests_boto3.functional.test_s3.test_put_excess_key_tags  | ok    | ok    | ok     |
| 8  | s3tests_boto3.functional.test_s3.test_put_excess_val_tags  | ok    | ok    | ok     |
| 9  | s3tests_boto3.functional.test_s3.test_put_modify_tags      | ok    | FAIL  | FAIL   |
| 10 | s3tests_boto3.functional.test_s3.test_put_delete_tags      | ok    | ok    | ok     |
| 11 | s3tests_boto3.functional.test_s3.test_put_obj_with_tags    | ok    | FAIL  | ok     |

## Versioning

Compatibility: 23/19/24 out of 26

|    | Test                                                                                        | s3-gw | minio | aws s3 |
|----|---------------------------------------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_versioning_bucket_create_suspend                      | ok    | ok    | ok     |
| 2  | s3tests_boto3.functional.test_s3.test_versioning_obj_create_read_remove                     | ok    | ok    | ok     |
| 3  | s3tests_boto3.functional.test_s3.test_versioning_obj_create_read_remove_head                | ok    | ok    | ok     |
| 4  | s3tests_boto3.functional.test_s3.test_versioning_obj_plain_null_version_removal             | ok    | ok    | ok     |
| 5  | s3tests_boto3.functional.test_s3.test_versioning_obj_plain_null_version_overwrite           | ok    | ok    | ok     |
| 6  | s3tests_boto3.functional.test_s3.test_versioning_obj_plain_null_version_overwrite_suspended | ok    | ok    | ok     |
| 7  | s3tests_boto3.functional.test_s3.test_versioning_obj_suspend_versions                       | ok    | ok    | ok     |
| 8  | s3tests_boto3.functional.test_s3.test_versioning_obj_create_versions_remove_all             | ok    | ok    | ok     |
| 9  | s3tests_boto3.functional.test_s3.test_versioning_obj_create_versions_remove_special_names   | ok    | ok    | ok     |
| 10 | s3tests_boto3.functional.test_s3.test_versioning_obj_create_overwrite_multipart             | ok    | ok    | ok     |
| 11 | s3tests_boto3.functional.test_s3.test_versioning_obj_list_marker                            | ok    | ok    | ok     |
| 12 | s3tests_boto3.functional.test_s3.test_versioning_copy_obj_version                           | ok    | ok    | ok     |
| 13 | s3tests_boto3.functional.test_s3.test_versioning_multi_object_delete                        | ok    | ok    | ok     |
| 14 | s3tests_boto3.functional.test_s3.test_versioning_multi_object_delete_with_marker            | ok    | ok    | ok     |
| 15 | s3tests_boto3.functional.test_s3.test_versioning_multi_object_delete_with_marker_create     | ok    | ERROR | ok     |
| 16 | s3tests_boto3.functional.test_s3.test_versioned_object_acl                                  | FAIL  | FAIL  | FAIL   |
| 17 | s3tests_boto3.functional.test_s3.test_versioned_object_acl_no_version_specified             | FAIL  | FAIL  | FAIL   |
| 18 | s3tests_boto3.functional.test_s3.test_versioned_concurrent_object_create_concurrent_remove  | ok    | ok    | ok     |
| 19 | s3tests_boto3.functional.test_s3.test_versioned_concurrent_object_create_and_remove         | ok    | ok    | ok     |
| 20 | s3tests_boto3.functional.test_s3.test_versioning_bucket_atomic_upload_return_version_id     | ok    | FAIL  | ok     |
| 21 | s3tests_boto3.functional.test_s3.test_versioning_bucket_multipart_upload_return_version_id  | ok    | FAIL  | ok     |
| 22 | s3tests_boto3.functional.test_s3.test_bucket_list_return_data_versioning                    | ERROR | ERROR | ok     |
| 23 | s3tests_boto3.functional.test_s3.test_object_copy_versioned_bucket                          | ok    | ok    | ok     |
| 24 | s3tests_boto3.functional.test_s3.test_object_copy_versioned_url_encoding                    | ok    | ok    | ok     |
| 25 | s3tests_boto3.functional.test_s3.test_object_copy_versioning_multipart_upload               | ok    | ok    | ok     |
| 26 | s3tests_boto3.functional.test_s3.test_multipart_copy_versioned                              | ok    | ERROR | ok     |

## Bucket

Compatibility:  38/38/45 out of 59

|    | Test                                                                                         | s3-gw | minio | aws s3 |
|----|----------------------------------------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_authorization_invalid_aws2      | FAIL  | FAIL  | FAIL   |
| 2  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_ua_empty_aws2                   | ERROR | ok    | ok     |
| 3  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_ua_none_aws2                    | ERROR | ok    | ok     |
| 4  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_invalid_aws2               | FAIL  | FAIL  | ok     |
| 5  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_empty_aws2                 | FAIL  | FAIL  | ok     |
| 6  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_none_aws2                  | FAIL  | FAIL  | FAIL   |
| 7  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_before_today_aws2          | FAIL  | ok    | ok     |
| 8  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_after_today_aws2           | FAIL  | ok    | ok     |
| 9  | s3tests_boto3.functional.test_headers.test_bucket_create_bad_date_before_epoch_aws2          | FAIL  | FAIL  | ok     |
| 10 | s3tests_boto3.functional.test_headers.test_bucket_create_contentlength_none                  | ok    | ok    | ok     |
| 11 | s3tests_boto3.functional.test_headers.test_bucket_put_bad_canned_acl                         | ok    | FAIL  | ok     |
| 12 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_expect_mismatch                 | ERROR | ERROR | ok     |
| 13 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_expect_empty                    | ok    | ok    | ok     |
| 14 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_contentlength_empty             | FAIL  | FAIL  | ok     |
| 15 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_contentlength_negative          | ok    | ok    | ok     |
| 16 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_contentlength_none              | ok    | ok    | ok     |
| 17 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_authorization_empty             | FAIL  | FAIL  | FAIL   |
| 18 | s3tests_boto3.functional.test_headers.test_bucket_create_bad_authorization_none              | FAIL  | FAIL  | FAIL   |
| 19 | s3tests_boto3.functional.test_s3.test_bucket_notexist                                        | ok    | ok    | ok     |
| 20 | s3tests_boto3.functional.test_s3.test_bucketv2_notexist                                      | ok    | ok    | ok     |
| 21 | s3tests_boto3.functional.test_s3.test_bucket_delete_notexist                                 | ok    | ok    | ok     |
| 22 | s3tests_boto3.functional.test_s3.test_bucket_delete_nonempty                                 | ok    | ok    | ok     |
| 23 | s3tests_boto3.functional.test_s3.test_bucket_concurrent_set_canned_acl                       | ok    | FAIL  | FAIL   |
| 24 | s3tests_boto3.functional.test_s3.test_bucket_create_delete                                   | ok    | ok    | ok     |
| 25 | s3tests_boto3.functional.test_s3.test_bucket_head                                            | ok    | ok    | ok     |
| 26 | s3tests_boto3.functional.test_s3.test_bucket_head_notexist                                   | ok    | ok    | ok     |
| 27 | s3tests_boto3.functional.test_s3.test_bucket_head_extended                                   | ERROR | ERROR | ERROR  |
| 28 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_starts_nonalpha               | ok    | ok    | ok     |
| 29 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_short_empty                   | ERROR | ERROR | ERROR  |
| 30 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_short_one                     | ok    | ok    | ok     |
| 31 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_short_two                     | ok    | ok    | ok     |
| 32 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_long                          | ERROR | ERROR | ERROR  |
| 33 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_long_60                      | ok    | ok    | ok     |
| 34 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_long_61                      | ok    | ok    | ok     |
| 35 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_long_62                      | ok    | ok    | ok     |
| 36 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_long_63                      | ok    | ok    | ok     |
| 37 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_ip                            | ok    | ok    | FAIL   |
| 38 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_bad_punctuation                   | ERROR | ERROR | ERROR  |
| 39 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_underscore                    | ok    | ok    | ok     |
| 40 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_long                          | ok    | ok    | ok     |
| 41 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_dash_at_end                   | ok    | ok    | ok     |
| 42 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_dot_dot                       | ok    | ok    | ok     |
| 43 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_dot_dash                      | ok    | ok    | ok     |
| 44 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_dns_dash_dot                      | ok    | ok    | ok     |
| 45 | s3tests_boto3.functional.test_s3.test_bucket_create_exists                                   | ERROR | ERROR | ok     |
| 46 | s3tests_boto3.functional.test_s3.test_bucket_get_location                                    | ok    | FAIL  | ERROR  |
| 47 | s3tests_boto3.functional.test_s3.test_bucket_create_exists_nonowner                          | ok    | FAIL  | ok     |
| 48 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_starts_alpha                 | ok    | ok    | ok     |
| 49 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_starts_digit                 | ok    | ok    | ok     |
| 50 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_contains_period              | ERROR | ok    | ok     |
| 51 | s3tests_boto3.functional.test_s3.test_bucket_create_naming_good_contains_hyphen              | ok    | ok    | ok     |
| 52 | s3tests_boto3.functional.test_s3.test_bucket_recreate_not_overriding                         | ERROR | ERROR | ok     |
| 53 | s3tests_boto3.functional.test_s3.test_bucket_create_special_key_names                        | ok    | ok    | ok     |
| 54 | s3tests_boto3.functional.test_s3.test_bucket_policy_set_condition_operator_end_with_IfExists | ERROR | ERROR | FAIL   |
| 55 | s3tests_boto3.functional.test_s3.test_buckets_create_then_list                               | ok    | ok    | ok     |
| 56 | s3tests_boto3.functional.test_s3.test_buckets_list_ctime                                     | ok    | ok    | FAIL   |
| 57 | s3tests_boto3.functional.test_s3.test_list_buckets_anonymous                                 | ok    | ERROR | ERROR  |
| 58 | s3tests_boto3.functional.test_s3.test_list_buckets_invalid_auth                              | ok    | ok    | ok     |
| 59 | s3tests_boto3.functional.test_s3.test_list_buckets_bad_auth                                  | ok    | ok    | ok     |

## Bucket ACL

Compatibility:  3/3/16 out of 33
This group is not explicitly supported by s3-gw, but some tests may pass.

|     | Test                                                                                       | s3-gw | minio | aws s3 |
|-----|--------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_bucket_recreate_overwrite_acl                        | FAIL  | FAIL  | FAIL   |
| 2   | s3tests_boto3.functional.test_s3.test_bucket_recreate_new_acl                              | FAIL  | FAIL  | FAIL   |
| 3   | s3tests_boto3.functional.test_s3.test_bucket_acl_default                                   | FAIL  | FAIL  | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_bucket_acl_canned_during_create                      | FAIL  | FAIL  | FAIL   |
| 5   | s3tests_boto3.functional.test_s3.test_bucket_acl_canned                                    | FAIL  | FAIL  | FAIL   |
| 6   | s3tests_boto3.functional.test_s3.test_bucket_acl_canned_publicreadwrite                    | FAIL  | FAIL  | FAIL   |
| 7   | s3tests_boto3.functional.test_s3.test_bucket_acl_canned_authenticatedread                  | FAIL  | FAIL  | FAIL   |
| 8   | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_userid_fullcontrol                  | ERROR | FAIL  | ERROR  |
| 9   | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_userid_read                         | ERROR | FAIL  | ERROR  |
| 10  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_userid_readacp                      | FAIL  | FAIL  | ERROR  |
| 11  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_userid_write                        | ERROR | FAIL  | ERROR  |
| 12  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_userid_writeacp                     | FAIL  | FAIL  | ERROR  |
| 13  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_nonexist_user                       | FAIL  | FAIL  | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_bucket_acl_no_grants                                 | ERROR | ERROR | FAIL   |
| 15  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_email                               | ERROR | FAIL  | ERROR  |
| 16  | s3tests_boto3.functional.test_s3.test_bucket_acl_grant_email_not_exist                     | FAIL  | FAIL  | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_bucket_acl_revoke_all                                | ERROR | ERROR | ok     |
| 18  | s3tests_boto3.functional.test_s3.test_bucket_acl_canned_private_to_private                 | ok    | ok    | ok     |
| 19  | s3tests_boto3.functional.test_s3.test_bucket_header_acl_grants                             | ERROR | FAIL  | FAIL   |
| 20  | s3tests_boto3.functional.test_s3.test_access_bucket_private_object_private                 | ok    | FAIL  | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_access_bucket_private_objectv2_private               | ok    | FAIL  | ok     |
| 22  | s3tests_boto3.functional.test_s3.test_access_bucket_private_object_publicread              | ERROR | ERROR | ok     |
| 23  | s3tests_boto3.functional.test_s3.test_access_bucket_private_objectv2_publicread            | ERROR | ERROR | ok     |
| 24  | s3tests_boto3.functional.test_s3.test_access_bucket_private_object_publicreadwrite         | ERROR | ERROR | ok     |
| 25  | s3tests_boto3.functional.test_s3.test_access_bucket_private_objectv2_publicreadwrite       | ERROR | ERROR | ok     |
| 26  | s3tests_boto3.functional.test_s3.test_access_bucket_publicread_object_private              | FAIL  | ERROR | ok     |
| 27  | s3tests_boto3.functional.test_s3.test_access_bucket_publicread_object_publicread           | FAIL  | ERROR | ok     |
| 28  | s3tests_boto3.functional.test_s3.test_access_bucket_publicread_object_publicreadwrite      | FAIL  | ERROR | ok     |
| 29  | s3tests_boto3.functional.test_s3.test_access_bucket_publicreadwrite_object_private         | FAIL  | ERROR | FAIL   |
| 30  | s3tests_boto3.functional.test_s3.test_access_bucket_publicreadwrite_object_publicread      | FAIL  | ERROR | FAIL   |
| 31  | s3tests_boto3.functional.test_s3.test_access_bucket_publicreadwrite_object_publicreadwrite | FAIL  | ERROR | FAIL   |
| 32  | s3tests_boto3.functional.test_s3.test_object_raw_authenticated_bucket_acl                  | ok    | ok    | ok     |
| 33  | s3tests_boto3.functional.test_s3.test_object_raw_authenticated_bucket_gone                 | ok    | ok    | ok     |

## CORS

Compatibility: 3/0/3 out of 4

|     | Test                                                       | s3-gw | minio | aws s3 |
|-----|------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_set_cors             | ok    | ERROR | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_cors_origin_response | FAIL  | ERROR | FAIL   |
| 3   | s3tests_boto3.functional.test_s3.test_cors_origin_wildcard | ok    | ERROR | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_cors_header_option   | ok    | ERROR | ok     |

## Encryption

Compatibility: 23/9/16 out of 29
Currently only SSE-C is supported
This group is not explicitly supported by s3-gw, but some other tests may pass.

|     | Test                                                                                     | s3-gw | minio | aws s3 |
|-----|------------------------------------------------------------------------------------------|-------|-------|--------|
| 1   | s3tests_boto3.functional.test_s3.test_encrypted_transfer_1b                              | ok    | ERROR | ok     |
| 2   | s3tests_boto3.functional.test_s3.test_encrypted_transfer_1kb                             | ok    | ERROR | ok     |
| 3   | s3tests_boto3.functional.test_s3.test_encrypted_transfer_1MB                             | ok    | ERROR | ok     |
| 4   | s3tests_boto3.functional.test_s3.test_encrypted_transfer_13b                             | ok    | ERROR | ok     |
| 5   | s3tests_boto3.functional.test_s3.test_encryption_sse_c_method_head                       | ok    | ERROR | ok     |
| 6   | s3tests_boto3.functional.test_s3.test_encryption_sse_c_present                           | ok    | ERROR | ok     |
| 7   | s3tests_boto3.functional.test_s3.test_encryption_sse_c_other_key                         | ok    | ERROR | FAIL   |
| 8   | s3tests_boto3.functional.test_s3.test_encryption_sse_c_invalid_md5                       | ok    | ok    | ok     |
| 9   | s3tests_boto3.functional.test_s3.test_encryption_sse_c_no_md5                            | ok    | ok    | ok     |
| 10  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_no_key                            | ok    | ok    | ok     |
| 11  | s3tests_boto3.functional.test_s3.test_encryption_key_no_sse_c                            | ok    | ok    | ok     |
| 12  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_multipart_upload                  | ok    | ERROR | ok     |
| 13  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_multipart_invalid_chunks_1        | ok    | ok    | ok     |
| 14  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_multipart_invalid_chunks_2        | ok    | ok    | ok     |
| 15  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_multipart_bad_download            | ok    | ERROR | FAIL   |
| 16  | s3tests_boto3.functional.test_s3.test_encryption_sse_c_post_object_authenticated_request | FAIL  | FAIL  | ok     |
| 17  | s3tests_boto3.functional.test_s3.test_sse_kms_method_head                                | ERROR | ERROR | ERROR  |
| 18  | s3tests_boto3.functional.test_s3.test_sse_kms_present                                    | ok    | ERROR | ERROR  |
| 19  | s3tests_boto3.functional.test_s3.test_sse_kms_no_key                                     | FAIL  | ok    | FAIL   |
| 20  | s3tests_boto3.functional.test_s3.test_sse_kms_not_declared                               | FAIL  | ok    | ok     |
| 21  | s3tests_boto3.functional.test_s3.test_sse_kms_multipart_upload                           | ok    | ERROR | ERROR  |
| 22  | s3tests_boto3.functional.test_s3.test_sse_kms_multipart_invalid_chunks_1                 | ok    | ERROR | ERROR  |
| 23  | s3tests_boto3.functional.test_s3.test_sse_kms_multipart_invalid_chunks_2                 | ok    | ERROR | ERROR  |
| 24  | s3tests_boto3.functional.test_s3.test_sse_kms_post_object_authenticated_request          | FAIL  | FAIL  | FAIL   |
| 25  | s3tests_boto3.functional.test_s3.test_sse_kms_transfer_1b                                | ok    | ERROR | ERROR  |
| 26  | s3tests_boto3.functional.test_s3.test_sse_kms_transfer_1kb                               | ok    | ERROR | ERROR  |
| 27  | s3tests_boto3.functional.test_s3.test_sse_kms_transfer_1MB                               | ok    | ERROR | ERROR  |
| 28  | s3tests_boto3.functional.test_s3.test_sse_kms_transfer_13b                               | ok    | ERROR | ERROR  |
| 29  | s3tests_boto3.functional.test_s3.test_sse_kms_read_declare                               | FAIL  | ok    | ok     |

## Lifecycle

Compatibility: 0/10/18 out of 29
This group is not explicitly supported by s3-gw, but some tests may pass.

|    | Test                                                                            | s3-gw | minio | aws s3 |
|----|---------------------------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_lifecycle_set                             | ERROR | ok    | ok     |
| 2  | s3tests_boto3.functional.test_s3.test_lifecycle_get                             | ERROR | FAIL  | ok     |
| 3  | s3tests_boto3.functional.test_s3.test_lifecycle_get_no_id                       | ERROR | ERROR | ok     |
| 4  | s3tests_boto3.functional.test_s3.test_lifecycle_expiration                      | ERROR | FAIL  | FAIL   |
| 5  | s3tests_boto3.functional.test_s3.test_lifecyclev2_expiration                    | ERROR | FAIL  | FAIL   |
| 6  | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_versioning_enabled   | ERROR | ERROR | ok     |
| 7  | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_tags1                | ERROR | ERROR | ERROR  |
| 8  | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_tags2                | ERROR | ERROR | ERROR  |
| 9  | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_versioned_tags2      | ERROR | ERROR | ERROR  |
| 10 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_noncur_tags1         | ERROR | ERROR | ERROR  |
| 11 | s3tests_boto3.functional.test_s3.test_lifecycle_id_too_long                     | FAIL  | FAIL  | ok     |
| 12 | s3tests_boto3.functional.test_s3.test_lifecycle_same_id                         | FAIL  | FAIL  | ok     |
| 13 | s3tests_boto3.functional.test_s3.test_lifecycle_invalid_status                  | FAIL  | FAIL  | ok     |
| 14 | s3tests_boto3.functional.test_s3.test_lifecycle_set_date                        | ERROR | ok    | ok     |
| 15 | s3tests_boto3.functional.test_s3.test_lifecycle_set_invalid_date                | FAIL  | ok    | ok     |
| 16 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_date                 | ERROR | FAIL  | FAIL   |
| 17 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_days0                | FAIL  | FAIL  | ok     |
| 18 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_header_put           | ERROR | ok    | ok     |
| 19 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_header_head          | ERROR | ok    | ok     |
| 20 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_header_tags_head     | ERROR | ok    | FAIL   |
| 21 | s3tests_boto3.functional.test_s3.test_lifecycle_expiration_header_and_tags_head | ERROR | ERROR | ok     |
| 22 | s3tests_boto3.functional.test_s3.test_lifecycle_set_noncurrent                  | ERROR | ok    | ok     |
| 23 | s3tests_boto3.functional.test_s3.test_lifecycle_noncur_expiration               | ERROR | ERROR | FAIL   |
| 24 | s3tests_boto3.functional.test_s3.test_lifecycle_set_deletemarker                | ERROR | ok    | ok     |
| 25 | s3tests_boto3.functional.test_s3.test_lifecycle_set_filter                      | ERROR | ok    | ok     |
| 26 | s3tests_boto3.functional.test_s3.test_lifecycle_set_empty_filter                | ERROR | ok    | ok     |
| 27 | s3tests_boto3.functional.test_s3.test_lifecycle_deletemarker_expiration         | ERROR | ERROR | FAIL   |
| 28 | s3tests_boto3.functional.test_s3.test_lifecycle_set_multipart                   | ERROR | ERROR | ok     |
| 29 | s3tests_boto3.functional.test_s3.test_lifecycle_multipart_expiration            | ERROR | ERROR | FAIL   |

## Policy and replication

Compatibility:  0/7/20 out of 35
This group is not explicitly supported by s3-gw, but some tests may pass.

|    | Test                                                                                | s3-gw | minio | aws s3 |
|----|-------------------------------------------------------------------------------------|-------|-------|--------|
| 1  | s3tests_boto3.functional.test_s3.test_bucket_policy                                 | ERROR | ok    | ok     |
| 2  | s3tests_boto3.functional.test_s3.test_bucketv2_policy                               | ERROR | ok    | ok     |
| 3  | s3tests_boto3.functional.test_s3.test_bucket_policy_acl                             | ERROR | ERROR | ok     |
| 4  | s3tests_boto3.functional.test_s3.test_bucketv2_policy_acl                           | ERROR | ERROR | ok     |
| 5  | s3tests_boto3.functional.test_s3.test_bucket_policy_different_tenant                | ERROR | ERROR | ERROR  |
| 6  | s3tests_boto3.functional.test_s3.test_bucketv2_policy_different_tenant              | ERROR | ERROR | ERROR  |
| 7  | s3tests_boto3.functional.test_s3.test_bucket_policy_another_bucket                  | ERROR | ok    | ERROR  |
| 8  | s3tests_boto3.functional.test_s3.test_bucketv2_policy_another_bucket                | ERROR | ok    | ERROR  |
| 9  | s3tests_boto3.functional.test_s3.test_bucket_policy_get_obj_existing_tag            | ERROR | ERROR | ok     |
| 10 | s3tests_boto3.functional.test_s3.test_bucket_policy_get_obj_tagging_existing_tag    | ERROR | ERROR | ok     |
| 11 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_tagging_existing_tag    | ERROR | ERROR | ok     |
| 12 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_copy_source             | ERROR | FAIL  | ok     |
| 13 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_copy_source_meta        | ERROR | FAIL  | ok     |
| 14 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_acl                     | ERROR | ERROR | ok     |
| 15 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_grant                   | ERROR | ERROR | ok     |
| 16 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_enc                     | ERROR | FAIL  | ERROR  |
| 17 | s3tests_boto3.functional.test_s3.test_bucket_policy_put_obj_request_obj_tag         | ERROR | ERROR | FAIL   |
| 18 | s3tests_boto3.functional.test_s3.test_bucket_policy_get_obj_acl_existing_tag        | ERROR | ERROR | ok     |
| 19 | s3tests_boto3.functional.test_s3.test_user_policy                                   | ERROR | ERROR | ERROR  |
| 20 | s3tests_boto3.functional.test_s3.test_get_bucket_policy_status                      | ERROR | ok    | ERROR  |
| 21 | s3tests_boto3.functional.test_s3.test_get_public_acl_bucket_policy_status           | ERROR | ERROR | ERROR  |
| 22 | s3tests_boto3.functional.test_s3.test_get_authpublic_acl_bucket_policy_status       | ERROR | ERROR | ERROR  |
| 23 | s3tests_boto3.functional.test_s3.test_get_publicpolicy_acl_bucket_policy_status     | ERROR | FAIL  | ERROR  |
| 24 | s3tests_boto3.functional.test_s3.test_get_nonpublicpolicy_acl_bucket_policy_status  | ERROR | ok    | ERROR  |
| 25 | s3tests_boto3.functional.test_s3.test_get_nonpublicpolicy_deny_bucket_policy_status | ERROR | ERROR | ERROR  |
| 26 | s3tests_boto3.functional.test_s3.test_get_default_public_block                      | ERROR | ERROR | ERROR  |
| 27 | s3tests_boto3.functional.test_s3.test_put_public_block                              | ERROR | ERROR | ok     |
| 28 | s3tests_boto3.functional.test_s3.test_block_public_put_bucket_acls                  | ERROR | ERROR | ok     |
| 29 | s3tests_boto3.functional.test_s3.test_block_public_object_canned_acls               | ERROR | ERROR | ok     |
| 30 | s3tests_boto3.functional.test_s3.test_block_public_policy                           | ERROR | ERROR | ok     |
| 31 | s3tests_boto3.functional.test_s3.test_ignore_public_acls                            | ERROR | ERROR | FAIL   |
| 32 | s3tests_boto3.functional.test_s3.test_get_tags_acl_public                           | ERROR | FAIL  | ok     |
| 33 | s3tests_boto3.functional.test_s3.test_put_tags_acl_public                           | ERROR | FAIL  | ok     |
| 34 | s3tests_boto3.functional.test_s3.test_delete_tags_obj_public                        | ERROR | ok    | ok     |
| 35 | s3tests_boto3.functional.test_s3.test_multipart_upload_on_a_bucket_with_policy      | ERROR | ERROR | ok     |

## Others

Compatibility: 2/2/3 out of 6

|   | Test                                                        | s3-gw | minio | aws s3 |
|---|-------------------------------------------------------------|-------|-------|--------|
| 1 | s3tests_boto3.functional.test_s3.test_100_continue          | FAIL  | ERROR | ok     |
| 2 | s3tests_boto3.functional.test_s3.test_account_usage         | ERROR | ERROR | ERROR  |
| 3 | s3tests_boto3.functional.test_s3.test_head_bucket_usage     | ERROR | ERROR | ERROR  |
| 4 | s3tests_boto3.functional.test_s3.test_logging_toggle        | ERROR | ERROR | ERROR  |
| 5 | s3tests_boto3.functional.test_s3.test_multi_object_delete   | ok    | ok    | ok     |
| 6 | s3tests_boto3.functional.test_s3.test_multi_objectv2_delete | ok    | ok    | ok     |
