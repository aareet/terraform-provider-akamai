package appsec

import (
	"encoding/json"
	"testing"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v2/pkg/appsec"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/mock"
)

func TestAccAkamaiIPGeoProtection_res_basic(t *testing.T) {
	t.Run("match by IPGeoProtection ID", func(t *testing.T) {
		client := &mockappsec{}

		cu := appsec.UpdateIPGeoProtectionResponse{}
		expectJSU := compactJSON(loadFixtureBytes("testdata/TestResIPGeoProtection/IPGeoProtectionUpdate.json"))
		json.Unmarshal([]byte(expectJSU), &cu)

		cr := appsec.GetIPGeoProtectionResponse{}
		expectJS := compactJSON(loadFixtureBytes("testdata/TestResIPGeoProtection/IPGeoProtection.json"))
		json.Unmarshal([]byte(expectJS), &cr)

		config := appsec.GetConfigurationResponse{}
		expectConfigs := compactJSON(loadFixtureBytes("testdata/TestResConfiguration/LatestConfiguration.json"))
		json.Unmarshal([]byte(expectConfigs), &config)

		client.On("GetConfiguration",
			mock.Anything,
			appsec.GetConfigurationRequest{ConfigID: 43253},
		).Return(&config, nil)

		client.On("GetIPGeoProtection",
			mock.Anything, // ctx is irrelevant for this test
			appsec.GetIPGeoProtectionRequest{ConfigID: 43253, Version: 7, PolicyID: "AAAA_81230"},
		).Return(&cr, nil)

		client.On("UpdateIPGeoProtection",
			mock.Anything, // ctx is irrelevant for this test
			appsec.UpdateIPGeoProtectionRequest{ConfigID: 43253, Version: 7, PolicyID: "AAAA_81230", ApplyNetworkLayerControls: false},
		).Return(&cu, nil)

		useClient(client, func() {
			resource.Test(t, resource.TestCase{
				PreCheck:   func() { testAccPreCheck(t) },
				IsUnitTest: true,
				Providers:  testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: loadFixtureString("testdata/TestResIPGeoProtection/match_by_id.tf"),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("akamai_appsec_ip_geo_protection.test", "id", "43253:AAAA_81230"),
							resource.TestCheckResourceAttr("akamai_appsec_ip_geo_protection.test", "enabled", "false"),
						),
					},
					{
						Config: loadFixtureString("testdata/TestResIPGeoProtection/update_by_id.tf"),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("akamai_appsec_ip_geo_protection.test", "id", "43253:AAAA_81230"),
							resource.TestCheckResourceAttr("akamai_appsec_ip_geo_protection.test", "enabled", "false"),
						),
					},
				},
			})
		})

		client.AssertExpectations(t)
	})

}
