package networklists

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v2/pkg/networklists"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// network_lists v2
//
// https://developer.akamai.com/api/cloud_security/network_lists/v2.html
func resourceNetworkList() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkListCreate,
		ReadContext:   resourceNetworkListRead,
		UpdateContext: resourceNetworkListUpdate,
		DeleteContext: resourceNetworkListDelete,
		CustomizeDiff: customdiff.All(
			VerifyContractGroupUnchanged,
		),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					IP,
					Geo,
				}, false)),
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"list": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"mode": {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					Append,
					Replace,
					Remove,
				}, false)),
			},
			"uniqueid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "uniqueId",
			},
			"network_list_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "network_list_id",
			},
			"sync_point": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "sync point",
			},
			"contract_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Contract ID",
			},
			"group_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Group ID",
			},
		},
	}
}

func resourceNetworkListCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("NETWORKLIST", "resourceNetworkListCreate")

	createNetworkList := networklists.CreateNetworkListRequest{}

	name, err := tools.GetStringValue("name", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	createNetworkList.Name = name

	listType, err := tools.GetStringValue("type", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	createNetworkList.Type = listType

	description, err := tools.GetStringValue("description", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	createNetworkList.Description = description

	contractid, err := tools.GetStringValue("contract_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	createNetworkList.ContractID = contractid

	groupid, err := tools.GetIntValue("group_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	createNetworkList.GroupID = groupid

	if len(contractid) > 0 || groupid > 0 {
		if len(contractid) == 0 || groupid == 0 {
			return diag.FromErr(fmt.Errorf("If either a contract_id or group_id is provided, both must be provided"))
		}
	}

	mode, err := tools.GetStringValue("mode", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}

	getNetworkLists := networklists.GetNetworkListsRequest{}
	getNetworkLists.Name = name
	getNetworkLists.Type = listType

	networklists, err := client.GetNetworkLists(ctx, getNetworkLists)
	if err != nil {
		logger.Errorf("calling 'getNetworkList': %s", err.Error())
		return diag.FromErr(err)
	}

	netlist := d.Get("list").(*schema.Set)
	nru := make([]string, 0, len(netlist.List()))

	for _, h := range netlist.List() {
		nru = append(nru, strings.ToLower(h.(string)))
	}

	finallist := make([]string, 0, len(netlist.List()))

	switch mode {
	case Remove:
		for _, hl := range netlist.List() {
			for _, h := range networklists.NetworkLists {

				if h.Name == hl.(string) {
					finallist = append(finallist, strings.ToLower(h.Name))
				}
			}
		}
	case Append:
		var oneShot bool

		for _, h := range networklists.NetworkLists {
			finallist = appendIfMissing(finallist, strings.ToLower(h.Name))
			for _, hl := range netlist.List() {
				finallist = appendIfMissing(finallist, strings.ToLower(hl.(string)))
			}
			oneShot = true
		}

		if oneShot == false {
			finallist = nru
		}

	case Replace:
		finallist = nru
	default:
		finallist = nru
	}

	createNetworkList.List = finallist

	spcr, err := client.CreateNetworkList(ctx, createNetworkList)
	if err != nil {
		logger.Errorf("calling 'createNetworkList': %s", err.Error())
		return diag.FromErr(err)
	}

	d.Set("name", spcr.Name)

	d.Set("sync_point", strconv.Itoa(spcr.SyncPoint))

	if err := d.Set("uniqueid", spcr.UniqueID); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("network_list_id", spcr.UniqueID); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("mode", mode); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("contract_id", contractid); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("group_id", groupid); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	d.SetId(spcr.UniqueID)

	return resourceNetworkListRead(ctx, d, m)
}

func resourceNetworkListUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("NETWORKLIST", "resourceNetworkListUpdate")

	updateNetworkList := networklists.UpdateNetworkListRequest{}
	updateNetworkList.UniqueID = d.Id()

	name, err := tools.GetStringValue("name", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	updateNetworkList.Name = name

	listType, err := tools.GetStringValue("type", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	updateNetworkList.Type = listType

	description, err := tools.GetStringValue("description", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	updateNetworkList.Description = description

	contractid, err := tools.GetStringValue("contract_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	updateNetworkList.ContractID = contractid

	groupid, err := tools.GetIntValue("group_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	updateNetworkList.GroupID = groupid

	if len(contractid) > 0 || groupid > 0 {
		if len(contractid) == 0 || groupid == 0 {
			return diag.FromErr(fmt.Errorf("If either a contract_id or group_id is provided, both must be provided"))
		}
	}

	mode, err := tools.GetStringValue("mode", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}

	listRequest := networklists.GetNetworkListRequest{}
	listRequest.UniqueID = d.Id()

	networkLists, err := client.GetNetworkList(ctx, listRequest)
	if err != nil {
		logger.Errorf("calling 'getNetworkList': %s", err.Error())
		return diag.FromErr(err)
	}

	netlist := d.Get("list").(*schema.Set)
	nru := make([]string, 0, len(netlist.List()))

	for _, h := range netlist.List() {
		nru = append(nru, strings.ToLower(h.(string)))
	}

	finallist := make([]string, 0, len(netlist.List()))

	switch mode {
	case Remove:
		for _, hl := range netlist.List() {

			for idx, h := range networkLists.List {
				if strings.ToLower(h) == strings.ToLower(hl.(string)) {
					networkLists.List = RemoveIndex(networkLists.List, idx)
				}
			}
		}
		finallist = networkLists.List

	case Append:
		for _, h := range networkLists.List {
			finallist = append(finallist, strings.ToLower(h))
		}
		for _, hl := range netlist.List() {
			finallist = appendIfMissing(finallist, strings.ToLower(hl.(string)))
		}
	case Replace:
		finallist = nru
	default:
		finallist = nru
	}

	updateNetworkList.List = finallist

	syncPoint, err := tools.GetIntValue("sync_point", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}

	updateNetworkList.SyncPoint = syncPoint

	_, err = client.UpdateNetworkList(ctx, updateNetworkList)
	if err != nil {
		logger.Errorf("calling 'updateNetworkList': %s", err.Error())
		return diag.FromErr(err)
	}

	if err := d.Set("contract_id", contractid); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("group_id", groupid); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	return resourceNetworkListRead(ctx, d, m)
}

func resourceNetworkListDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("NETWORKLIST", "resourceNetworkListRemove")

	removeNetworkList := networklists.RemoveNetworkListRequest{}
	removeNetworkList.UniqueID = d.Id()
	_, errd := client.RemoveNetworkList(ctx, removeNetworkList)
	if errd != nil {
		logger.Errorf("calling 'removeNetworkList': %s", errd.Error())
	}

	d.SetId("")

	return nil
}

func resourceNetworkListRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("NETWORKLISTs", "resourceNetworkListRead")

	getNetworkList := networklists.GetNetworkListRequest{}
	getNetworkList.UniqueID = d.Id()

	mode, err := tools.GetStringValue("mode", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	var detectCase string

	netlist := d.Get("list").(*schema.Set)
	for _, hl := range netlist.List() {
		if hl.(string) == strings.ToLower(hl.(string)) {
			detectCase = "LOWER"
		} else {
			detectCase = "UPPER"
		}
	}
	finalldata := make([]string, 0, len(netlist.List()))

	networklist, err := client.GetNetworkList(ctx, getNetworkList)
	if err != nil {
		logger.Errorf("calling 'getNetworkList': %s", err.Error())
		return diag.FromErr(err)
	}

	switch mode {
	case Remove:
		for _, hl := range netlist.List() {
			for _, h := range networklist.List {

				if strings.ToLower(h) == strings.ToLower(hl.(string)) {
					finalldata = append(finalldata, strings.ToLower(h))
				}
			}
		}

		if len(finalldata) == 0 {
			for _, hl := range netlist.List() {
				finalldata = append(finalldata, strings.ToLower(hl.(string)))
			}
		}

	case Append:
		for _, h := range networklist.List {

			for _, hl := range netlist.List() {
				if strings.ToLower(h) == strings.ToLower(hl.(string)) {
					finalldata = append(finalldata, strings.ToLower(h))
				}
			}
		}
	case Replace:
		for _, h := range networklist.List {
			finalldata = append(finalldata, strings.ToLower(h))
		}
	default:
		for _, h := range networklist.List {
			finalldata = append(finalldata, strings.ToLower(h))
		}
	}

	sort.Strings(finalldata)

	if err := d.Set("sync_point", networklist.SyncPoint); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("name", networklist.Name); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("type", networklist.Type); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if detectCase == "LOWER" {
		for index, value := range finalldata {
			finalldata[index] = strings.ToLower(value)
		}
	} else {
		for index, value := range finalldata {
			finalldata[index] = strings.ToUpper(value)
		}
	}

	if err := d.Set("description", networklist.Description); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	d.Set("list", nil)
	if err := d.Set("list", finalldata); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if mode == "" {
		mode = "REPLACE"
	}

	if err := d.Set("mode", mode); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("uniqueid", networklist.UniqueID); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	if err := d.Set("network_list_id", networklist.UniqueID); err != nil {
		return diag.FromErr(fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error()))
	}

	d.SetId(networklist.UniqueID)

	return nil
}

func appendIfMissing(slice []string, s string) []string {
	for _, element := range slice {
		if element == s {
			return slice
		}
	}
	return append(slice, s)
}

// RemoveIndex removes an element from the slice and returns it
func RemoveIndex(hl []string, index int) []string {
	return append(hl[:index], hl[index+1:]...)
}

// VerifyContractGroupUnchanged compares the configuration's value for the contract_id and group_id with the resource's
// value specified in the resources's ID, to ensure that the user has not inadvertently modified the configuration's
// value; any such modifications indicate an incorrect understanding of the Update operation.

func VerifyContractGroupUnchanged(_ context.Context, d *schema.ResourceDiff, m interface{}) error {
	meta := akamai.Meta(m)
	logger := meta.Log("NETWORKLISTs", "VerifyContractGroupUnchanged")

	if d.HasChange("contract_id") {
		old, new := d.GetChange("contract_id")
		oldvalue := old.(string)
		newvalue := new.(string)
		if len(oldvalue) > 0 {
			logger.Errorf("%s value %s specified in configuration differs from resource ID's value %s", "contract_id", newvalue, oldvalue)
			return fmt.Errorf("%s value %s specified in configuration differs from resource ID's value %s", "contract_id", newvalue, oldvalue)
		}
	}

	if d.HasChange("group_id") {
		old, new := d.GetChange("group_id")
		oldvalue := old.(int)
		newvalue := new.(int)
		if oldvalue > 0 {
			logger.Errorf("%s value %d specified in configuration differs from resource ID's value %d", "group_id", newvalue, oldvalue)
			return fmt.Errorf("%s value %d specified in configuration differs from resource ID's value %d", "group_id", newvalue, oldvalue)
		}
	}

	return nil
}

// Append Replace Remove mode flags
const (
	Append  = "APPEND"
	Replace = "REPLACE"
	Remove  = "REMOVE"

	IP  = "IP"
	Geo = "GEO"
)
