package appsec

import (
	"context"
	"errors"
	"fmt"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v2/pkg/appsec"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSecurityPolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecurityPolicyRead,
		Schema: map[string]*schema.Schema{
			"security_policy_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"config_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"security_policy_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Policy ID",
			},
			"security_policy_id_list": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Policy ID List",
			},
			"output_text": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Text Export representation",
			},
		},
	}
}

func dataSourceSecurityPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "dataSourceSecurityPolicyRead")

	configid, err := tools.GetIntValue("config_id", d)
	if err != nil {
		return diag.FromErr(err)
	}
	version := getLatestConfigVersion(ctx, configid, m)
	securityPolicyName, err := tools.GetStringValue("security_policy_name", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}

	getSecurityPoliciesRequest := appsec.GetSecurityPoliciesRequest{
		ConfigID: configid,
		Version: version,
	}

	securitypolicies, err := client.GetSecurityPolicies(ctx, getSecurityPoliciesRequest)
	if err != nil {
		logger.Errorf("calling 'getSecurityPolicies': %s", err.Error())
		return diag.FromErr(err)
	}

	securityPoliciesList := make([]string, 0, len(securitypolicies.Policies))

	for _, val := range securitypolicies.Policies {
		securityPoliciesList = append(securityPoliciesList, val.PolicyID)
		if val.PolicyName == securityPolicyName {
			if err := d.Set("security_policy_id", val.PolicyID); err != nil {
				return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
			}
			if err = d.Set("security_policy_name", val.PolicyName); err != nil {
				return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
			}
		}
	}

	if err := d.Set("security_policy_id_list", securityPoliciesList); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	ots := OutputTemplates{}
	InitTemplates(ots)

	outputtext, err := RenderTemplates(ots, "securityPoliciesDS", securitypolicies)
	if err == nil {
		if err := d.Set("output_text", outputtext); err != nil {
			return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
		}
	}

	d.SetId(fmt.Sprintf("%d:%d", configid, version))

	return nil
}
