package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/nas"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-alicloud/alicloud/connectivity"
)

func dataSourceAlicloudAccessRules() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAlicloudAccessRulesRead,

		Schema: map[string]*schema.Schema{
			"source_cidr_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"access_group_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"user_access": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"rw_access": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"output_file": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Computed values
			"rules": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_cidr_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"priority": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"access_rule_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"user_access": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"rw_access": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAlicloudAccessRulesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)

	request := nas.CreateDescribeAccessRulesRequest()
	request.AccessGroupName = d.Get("access_group_name").(string)
	request.RegionId = string(client.Region)
	request.PageSize = requests.NewInteger(PageSizeLarge)
	request.PageNumber = requests.NewInteger(1)
	var allArs []nas.AccessRule
	invoker := NewInvoker()
	for {
		var raw interface{}
		if err := invoker.Run(func() error {
			rsp, err := client.WithNasClient(func(nasClient *nas.Client) (interface{}, error) {
				return nasClient.DescribeAccessRules(request)
			})
			raw = rsp
			return err
		}); err != nil {
			return WrapErrorf(err, DataDefaultErrorMsg, "alicloud_nas_access_rules", request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		resp, _ := raw.(*nas.DescribeAccessRulesResponse)
		if resp == nil || len(resp.AccessRules.AccessRule) < 1 {
			break
		}
		for _, rule := range resp.AccessRules.AccessRule {
			if v, ok := d.GetOk("source_cidr_ip"); ok && rule.SourceCidrIp != Trim(v.(string)) {
				continue
			}
			if v, ok := d.GetOk("user_access"); ok && rule.UserAccess != Trim(v.(string)) {
				continue
			}
			if v, ok := d.GetOk("rw_access"); ok && rule.RWAccess != Trim(v.(string)) {
				continue
			}
			allArs = append(allArs, rule)
		}

		if len(resp.AccessRules.AccessRule) < PageSizeLarge {
			break
		}

		if page, err := getNextpageNumber(request.PageNumber); err != nil {
			return err
		} else {
			request.PageNumber = page
		}
	}

	return accessRulesDecriptionAttributes(d, allArs, meta)
}

func accessRulesDecriptionAttributes(d *schema.ResourceData, nasSetTypes []nas.AccessRule, meta interface{}) error {
	var ids []string
	var s []map[string]interface{}

	for _, ag := range nasSetTypes {
		mapping := map[string]interface{}{
			"source_cidr_ip": ag.SourceCidrIp,
			"priority":       ag.Priority,
			"access_rule_id": ag.AccessRuleId,
			"user_access":    ag.UserAccess,
			"rw_access":      ag.RWAccess,
		}
		ids = append(ids, d.Get("access_group_name").(string)+":"+ag.AccessRuleId)
		s = append(s, mapping)
	}

	d.SetId(dataResourceIdHash(ids))
	if err := d.Set("rules", s); err != nil {
		return WrapError(err)
	}
	if err := d.Set("ids", ids); err != nil {
		return WrapError(err)
	}
	// create a json file in current directory and write data source to it.
	if output, ok := d.GetOk("output_file"); ok && output.(string) != "" {
		writeToFile(output.(string), s)
	}
	return nil
}
