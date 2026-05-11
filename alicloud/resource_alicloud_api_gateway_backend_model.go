package alicloud

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAlicloudApiGatewayBackendModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudApiGatewayBackendModelCreate,
		Read:   resourceAlicloudApiGatewayBackendModelRead,
		Update: resourceAlicloudApiGatewayBackendModelUpdate,
		Delete: resourceAlicloudApiGatewayBackendModelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"backend_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"backend_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"HTTP", "VPC", "FC_EVENT", "FC_EVENT_V3", "FC_HTTP", "FC_HTTP_V3", "OSS", "MOCK", "EVENTBRIDGE"}, false),
			},
			"stage_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"backend_model_data": {
				Type:     schema.TypeString,
				Required: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == new {
						return true
					}
					var oldMap, newMap map[string]interface{}
					if err := json.Unmarshal([]byte(old), &oldMap); err != nil {
						return false
					}
					if err := json.Unmarshal([]byte(new), &newMap); err != nil {
						return false
					}
					return reflect.DeepEqual(oldMap, newMap)
				},
			},
			"backend_model_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAlicloudApiGatewayBackendModelCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	request := make(map[string]interface{})
	var err error

	request["BackendId"] = d.Get("backend_id").(string)
	request["BackendType"] = d.Get("backend_type").(string)
	request["StageName"] = d.Get("stage_name").(string)
	request["BackendModelData"] = d.Get("backend_model_data").(string)

	if v, ok := d.GetOk("description"); ok {
		request["Description"] = v
	}

	var response map[string]interface{}
	action := "CreateBackendModel"
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		resp, err := client.RpcPost("CloudAPI", "2016-07-14", action, nil, request, false)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		response = resp
		addDebug(action, response, request)
		return nil
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "alicloud_api_gateway_backend_model", action, AlibabaCloudSdkGoERROR)
	}

	backendId := d.Get("backend_id").(string)
	stageName := d.Get("stage_name").(string)
	d.SetId(fmt.Sprintf("%s%s%s", backendId, COLON_SEPARATED, stageName))

	return resourceAlicloudApiGatewayBackendModelRead(d, meta)
}

func resourceAlicloudApiGatewayBackendModelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	cloudApiService := CloudApiService{client}

	object, err := cloudApiService.DescribeApiGatewayBackendModel(d.Id())
	if err != nil {
		if NotFoundError(err) {
			log.Printf("[DEBUG] Resource alicloud_api_gateway_backend_model cloudApiService.DescribeApiGatewayBackendModel Failed!!! %s", err)
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}

	d.Set("backend_id", object["BackendId"])
	d.Set("backend_type", object["BackendType"])
	d.Set("stage_name", object["StageName"])
	d.Set("description", object["Description"])
	d.Set("backend_model_data", object["BackendModelData"])
	d.Set("backend_model_id", object["BackendModelId"])

	return nil
}

func resourceAlicloudApiGatewayBackendModelUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	var err error
	update := false

	parts, err := ParseResourceId(d.Id(), 2)
	if err != nil {
		return WrapError(err)
	}
	backendId := parts[0]
	stageName := parts[1]

	request := map[string]interface{}{
		"BackendId":      backendId,
		"StageName":      stageName,
		"BackendType":    d.Get("backend_type").(string),
		"BackendModelId": d.Get("backend_model_id").(string),
	}

	request["BackendModelData"] = d.Get("backend_model_data").(string)

	if d.HasChange("description") {
		update = true
		if v, ok := d.GetOk("description"); ok {
			request["Description"] = v
		}
	}

	if d.HasChange("backend_model_data") {
		update = true
	}

	if update {
		action := "ModifyBackendModel"
		wait := incrementalWait(3*time.Second, 3*time.Second)
		err = resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
			resp, err := client.RpcPost("CloudAPI", "2016-07-14", action, nil, request, false)
			if err != nil {
				if NeedRetry(err) {
					wait()
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			addDebug(action, resp, request)
			return nil
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
		}
	}

	return resourceAlicloudApiGatewayBackendModelRead(d, meta)
}

func resourceAlicloudApiGatewayBackendModelDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	var err error

	parts, err := ParseResourceId(d.Id(), 2)
	if err != nil {
		return WrapError(err)
	}
	backendId := parts[0]
	stageName := parts[1]

	request := map[string]interface{}{
		"BackendId":      backendId,
		"StageName":      stageName,
		"BackendModelId": d.Get("backend_model_id").(string),
	}

	action := "DeleteBackendModel"
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		resp, err := client.RpcPost("CloudAPI", "2016-07-14", action, nil, request, false)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug(action, resp, request)
		return nil
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"NotFoundBackend", "NotFoundBackendModel"}) || NotFoundError(err) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
	}
	return nil
}

func convertBackendConfig(backendType string, backendConfig map[string]interface{}) (string, error) {
	modelData := make(map[string]interface{})

	// ServiceTimeout is a common field on BackendConfig
	if v, ok := backendConfig["ServiceTimeout"].(float64); ok && v > 0 {
		modelData["ServiceTimeout"] = int(v)
	}

	switch backendType {
	case "HTTP":
		if v, ok := backendConfig["ServiceAddress"]; ok && v != "" {
			modelData["ServiceAddress"] = v
		}
		if v, ok := backendConfig["HttpTargetHostName"]; ok && v != "" {
			modelData["HttpTargetHostName"] = v
		}
	case "VPC":
		if vpcConfig, ok := backendConfig["VpcConfig"].(map[string]interface{}); ok && len(vpcConfig) > 0 {
			newVpcConfig := make(map[string]interface{})
			if v, ok := vpcConfig["VpcAccessId"]; ok && v != "" {
				newVpcConfig["VpcAccessId"] = v
			}
			if v, ok := vpcConfig["VpcScheme"]; ok && v != "" {
				newVpcConfig["VpcScheme"] = v
			}
			if v, ok := vpcConfig["VpcTargetHostName"]; ok && v != "" {
				newVpcConfig["VpcTargetHostName"] = v
			}
			if v, ok := vpcConfig["ServiceHttpMethod"]; ok && v != "" {
				newVpcConfig["ServiceHttpMethod"] = v
			}
			if v, ok := vpcConfig["ServicePath"]; ok && v != "" {
				newVpcConfig["ServicePath"] = v
			}
			if len(newVpcConfig) > 0 {
				modelData["VpcConfig"] = newVpcConfig
			}
		}
	case "FC_EVENT", "FC_EVENT_V3", "FC_HTTP", "FC_HTTP_V3":
		if fcConfig, ok := backendConfig["FunctionComputeConfig"].(map[string]interface{}); ok && len(fcConfig) > 0 {
			newFcConfig := make(map[string]interface{})
			for _, key := range []string{
				"FcRegionId", "FcType", "ServiceName", "FunctionName", "RoleArn",
				"Qualifier", "FcVersion", "Path", "FcBaseUrl", "Method",
				"ContentTypeCatagory", "ContentTypeValue", "OnlyBusinessPath", "TriggerName",
			} {
				if v, ok := fcConfig[key]; ok && v != "" {
					newFcConfig[key] = v
				}
			}
			if len(newFcConfig) > 0 {
				modelData["FunctionComputeConfig"] = newFcConfig
			}
		}
	case "OSS":
		if ossConfig, ok := backendConfig["OssConfig"].(map[string]interface{}); ok && len(ossConfig) > 0 {
			newOssConfig := make(map[string]interface{})
			for _, key := range []string{"OssRegionId", "BucketName", "Key", "Action"} {
				if v, ok := ossConfig[key]; ok && v != "" {
					newOssConfig[key] = v
				}
			}
			if len(newOssConfig) > 0 {
				modelData["OssConfig"] = newOssConfig
			}
		}
	case "MOCK":
		if mockConfig, ok := backendConfig["MockConfig"].(map[string]interface{}); ok && len(mockConfig) > 0 {
			newMockConfig := make(map[string]interface{})
			if v, ok := mockConfig["MockResult"]; ok && v != "" {
				newMockConfig["MockResult"] = v
			}
			if v, ok := mockConfig["MockStatusCode"]; ok {
				if code, ok := v.(float64); ok && code > 0 {
					newMockConfig["MockStatusCode"] = int(code)
				} else if code, ok := v.(int); ok && code > 0 {
					newMockConfig["MockStatusCode"] = code
				}
			}
			if headers, ok := mockConfig["MockHeaders"].([]interface{}); ok && len(headers) > 0 {
				newMockConfig["MockHeaders"] = headers
			}
			if len(newMockConfig) > 0 {
				modelData["MockConfig"] = newMockConfig
			}
		}
	case "EVENTBRIDGE":
		if ebConfig, ok := backendConfig["EventBridgeConfig"].(map[string]interface{}); ok && len(ebConfig) > 0 {
			newEbConfig := make(map[string]interface{})
			for k, v := range ebConfig {
				if v != "" && v != nil {
					newEbConfig[k] = v
				}
			}
			if len(newEbConfig) > 0 {
				modelData["EventBridgeConfig"] = newEbConfig
			}
		}
	}

	bytes, err := json.Marshal(modelData)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *CloudApiService) DescribeApiGatewayBackendModel(id string) (object map[string]interface{}, err error) {
	client := s.client

	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return object, WrapError(err)
	}
	backendId := parts[0]
	stageName := parts[1]

	request := map[string]interface{}{
		"BackendId": backendId,
	}

	var response map[string]interface{}
	action := "DescribeBackendInfo"
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err := client.RpcPost("CloudAPI", "2016-07-14", action, nil, request, true)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		response = resp
		addDebug(action, response, request)
		return nil
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"NotFoundBackend"}) {
			return object, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}

	v, err := jsonpath.Get("$.BackendInfo", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.BackendInfo", response)
	}

	backendInfo, ok := v.(map[string]interface{})
	if !ok {
		return object, WrapError(Error("DescribeApiGatewayBackendModel: BackendInfo is not a map"))
	}

	// Try to find backend models from various possible response structures
	var backendModels []interface{}

	if models, ok := backendInfo["BackendModels"]; ok {
		if modelList, ok := models.([]interface{}); ok {
			backendModels = modelList
		}
	} else if models, ok := backendInfo["BackendModelList"]; ok {
		if modelList, ok := models.([]interface{}); ok {
			backendModels = modelList
		}
	} else if models, ok := backendInfo["BackendModel"]; ok {
		if modelList, ok := models.([]interface{}); ok {
			backendModels = modelList
		} else if modelMap, ok := models.(map[string]interface{}); ok {
			backendModels = []interface{}{modelMap}
		}
	}

	backendType := ""
	if v, ok := backendInfo["BackendType"]; ok {
		backendType = fmt.Sprint(v)
	}

	for _, model := range backendModels {
		modelMap, ok := model.(map[string]interface{})
		if !ok {
			continue
		}
		if modelMap["StageName"] == stageName {
			modelMap["BackendId"] = backendId
			modelMap["BackendType"] = backendType

			if backendConfig, ok := modelMap["BackendConfig"].(map[string]interface{}); ok {
				if modelDataStr, err := convertBackendConfig(backendType, backendConfig); err == nil {
					modelMap["BackendModelData"] = modelDataStr
				}
			}

			return modelMap, nil
		}
	}

	return object, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
}
