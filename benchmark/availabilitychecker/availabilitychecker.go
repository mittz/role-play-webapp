package availabilitychecker

import (
	"context"
	"fmt"
	"math"
	"path"
	"strings"

	asset "cloud.google.com/go/asset/apiv1"
	compute "cloud.google.com/go/compute/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/mittz/role-play-webapp/benchmark/utils"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/sqladmin/v1"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
)

const (
	RATE_NO_RESOURCE = iota
	RATE_ZONAL
	RATE_REGIONAL
	RATE_MULTI_REGIONAL
)

type AvailabilityChecker struct {
	projectID string
}

func NewAvailabilityChecker() *AvailabilityChecker {
	return new(AvailabilityChecker)
}

func RateArchitecture(projectID string, labels map[string]string) (int, error) {
	if len(labels) == 0 {
		return 0, fmt.Errorf("labels are not set")
	}

	ac := NewAvailabilityChecker()
	ac.SetProjectID(projectID)

	minRate := math.MaxInt32
	rateFunctions := []func(string, string) (int, error){
		ac.RateComputeEngine,
		ac.RateAppEngine,
		ac.RateCloudRun,
		ac.RateCloudFunctions,
		ac.RateCloudSQL,
		ac.RateCloudSpanner,
		// ac.RateAlloyDB,
		// ac.RateBigTable,
		// ac.Datastore,
		// ac.RateBigQuery,
	}

	for k, v := range labels {
		minRateByLabel := math.MaxInt32
		for _, f := range rateFunctions {
			rate, err := f(k, v)
			if err != nil {
				return 0, err
			}

			if rate > RATE_NO_RESOURCE {
				minRateByLabel = utils.GetMin(minRateByLabel, rate)
			}
		}

		if minRateByLabel == math.MaxInt32 {
			return 0, fmt.Errorf("Resource labelled %s:%s is not found", k, v)
		}

		minRate = utils.GetMin(minRate, minRateByLabel)
	}

	return minRate, nil
}

func rateAvailability(regions map[string]interface{}, zones map[string]interface{}) int {
	if len(regions) >= 2 {
		return RATE_MULTI_REGIONAL
	}

	if len(zones) >= 2 {
		return RATE_REGIONAL
	}

	if len(zones) > 0 {
		return RATE_ZONAL
	}

	return RATE_NO_RESOURCE
}

func (ac *AvailabilityChecker) SetProjectID(projectID string) {
	ac.projectID = projectID
}

func (ac *AvailabilityChecker) RateComputeEngine(labelKey string, labelVal string) (int, error) {
	ctx := context.Background()
	c, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return 0, err
	}
	defer c.Close()

	req := &computepb.AggregatedListInstancesRequest{Project: ac.projectID}
	it := c.AggregatedList(ctx, req)

	type ComputeEngine struct {
		Fingerprint string
		Name        string
		Zone        string
		MachineType string
		Status      string
	}
	gces := make(map[string]ComputeEngine)

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		for _, instance := range resp.Value.Instances {
			if instance.GetStatus() == "RUNNING" && instance.Labels[labelKey] == labelVal {
				// For a custom machine type: zones/zone/machineTypes/custom-CPUS-MEMOR e.g. zones/us-central1-f/machineTypes/custom-4-5120
				gces[instance.GetFingerprint()] = ComputeEngine{
					Fingerprint: instance.GetFingerprint(),
					Name:        instance.GetName(),
					Zone:        path.Base(instance.GetZone()),
					MachineType: path.Base(instance.GetMachineType()),
					Status:      instance.GetStatus(),
				}
			}
		}
	}

	regions := make(map[string]interface{})
	zones := make(map[string]interface{})
	for _, gce := range gces {
		region := strings.Join(strings.Split(gce.Zone, "-")[0:2], "-")
		regions[region] = struct{}{}
		zones[gce.Zone] = struct{}{}
	}

	return rateAvailability(regions, zones), nil
}

func (ac *AvailabilityChecker) RateAppEngine(labelKey string, labelVal string) (int, error) {
	scope := fmt.Sprintf("projects/%s", ac.projectID)
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	req := &assetpb.SearchAllResourcesRequest{
		Scope: scope,
		AssetTypes: []string{
			"appengine.googleapis.com/Application",
			"appengine.googleapis.com/Service",
		},
	}

	it := client.SearchAllResources(ctx, req)
	isServed, isLabelled := false, false
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		if resource.GetAssetType() == "appengine.googleapis.com/Application" {
			isServed = resource.GetState() == "SERVING"
		}

		if resource.GetAssetType() == "appengine.googleapis.com/Service" {
			isLabelled = resource.GetLabels()[labelKey] == labelVal
		}
	}

	if isServed && isLabelled {
		return RATE_REGIONAL, nil
	}

	return RATE_NO_RESOURCE, nil
}

func (ac *AvailabilityChecker) RateCloudRun(labelKey string, labelVal string) (int, error) {
	scope := fmt.Sprintf("projects/%s", ac.projectID)
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	req := &assetpb.SearchAllResourcesRequest{
		Scope: scope,
		AssetTypes: []string{
			"run.googleapis.com/Service",
		},
	}

	regions := make(map[string]interface{})
	it := client.SearchAllResources(ctx, req)
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		if resource.GetLabels()[labelKey] == labelVal {
			regions[resource.GetLocation()] = struct{}{}
		}
	}

	if len(regions) == 0 {
		return RATE_NO_RESOURCE, nil
	}

	if len(regions) >= 2 {
		return RATE_MULTI_REGIONAL, nil
	}

	return RATE_REGIONAL, nil
}

func (ac *AvailabilityChecker) RateCloudFunctions(labelKey string, labelVal string) (int, error) {
	scope := fmt.Sprintf("projects/%s", ac.projectID)
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	req := &assetpb.SearchAllResourcesRequest{
		Scope: scope,
		AssetTypes: []string{
			"cloudfunctions.googleapis.com/CloudFunction",
		},
	}

	regions := make(map[string]interface{})
	it := client.SearchAllResources(ctx, req)
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		if resource.GetLabels()[labelKey] == labelVal {
			regions[resource.GetLocation()] = struct{}{}
		}
	}

	if len(regions) == 0 {
		return RATE_NO_RESOURCE, nil
	}

	if len(regions) >= 2 {
		return RATE_MULTI_REGIONAL, nil
	}

	return RATE_REGIONAL, nil
}

// TODO: Consider case of multi-regional instances
// Use Cloud SQL Admin API because Asset Inventory doesn't reply availability type
func (ac *AvailabilityChecker) RateCloudSQL(labelKey string, labelVal string) (int, error) {
	ctx := context.Background()

	// Create an http.Client that uses Application Default Credentials.
	hc, err := google.DefaultClient(ctx, sqladmin.SqlserviceAdminScope)
	if err != nil {
		return 0, err
	}

	// Create the Google Cloud SQL service.
	service, err := sqladmin.New(hc)
	if err != nil {
		return 0, err
	}

	// List instances for the project ID.
	instances, err := service.Instances.List(ac.projectID).Do()
	if err != nil {
		return 0, err
	}

	if len(instances.Items) == 0 {
		return RATE_NO_RESOURCE, nil
	}

	type CloudSQL struct {
		Name             string
		AvailabilityType string
		Status           string
		MachineType      string
	}

	sqls := make(map[string]CloudSQL)
	for _, instance := range instances.Items {
		if instance.State == "RUNNABLE" && instance.Settings.UserLabels[labelKey] == labelVal {
			sqls[instance.Name] = CloudSQL{
				Name:             instance.Name,
				AvailabilityType: instance.Settings.AvailabilityType,
				Status:           instance.State,
				MachineType:      instance.Settings.Tier,
			}
		}
	}

	if len(sqls) == 0 {
		return RATE_NO_RESOURCE, nil
	}

	for _, sql := range sqls {
		if sql.AvailabilityType == "REGIONAL" {
			return RATE_REGIONAL, nil
		}
	}

	return RATE_ZONAL, nil
}

// In order to get processing_units metrics, sdk library is required as Asset Inventory can't collect.
func (ac *AvailabilityChecker) RateCloudSpannerSDK(labelKey string, labelVal string) (int, error) {
	ctx := context.Background()
	c, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return 0, err
	}
	defer c.Close()

	req := &instancepb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s", ac.projectID),
	}
	it := c.ListInstances(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		// TODO: Use resp.
		fmt.Println(resp)
	}

	return 0, nil
}

func (ac *AvailabilityChecker) RateCloudSpanner(labelKey string, labelVal string) (int, error) {
	scope := fmt.Sprintf("projects/%s", ac.projectID)
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	req := &assetpb.SearchAllResourcesRequest{
		Scope: scope,
		AssetTypes: []string{
			"spanner.googleapis.com/Instance",
		},
	}

	regions := make(map[string]interface{})
	it := client.SearchAllResources(ctx, req)
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return 0, err
		}

		if resource.GetLabels()[labelKey] == labelVal {
			location := resource.GetLocation()
			if !strings.Contains(location, "regional") {
				return RATE_MULTI_REGIONAL, nil
			}
			regions[location] = struct{}{}
		}
	}

	if len(regions) == 0 {
		return RATE_NO_RESOURCE, nil
	}

	return RATE_REGIONAL, nil
}

// TODO: Call AlloyDB API directly as client library doesn't support the instance management
// https://cloud.google.com/alloydb/docs/reference/rest
// func (ac *AvailabilityChecker) RateAlloyDB(labelKey string, labelVal string) (int, error) {
// 	return 0, nil
// }

// func (ac *AvailabilityChecker) RateBigTable() (int, error) {
// 	return 0, nil
// }
