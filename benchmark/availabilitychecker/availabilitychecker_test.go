package availabilitychecker

import (
	"os"
	"testing"

	"github.com/mittz/role-play-webapp/benchmark/utils"
	. "gopkg.in/check.v1"
)

type BenchmarkTestSuite struct {
	ProjectID string
}

func init() {
	projectID := utils.GetEnvProjectID()
	Suite(&BenchmarkTestSuite{ProjectID: projectID})
}

func Test(t *testing.T) { TestingT(t) }

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TODO: This depends on the test environment, so delete this once complete tests
func (b *BenchmarkTestSuite) TestRateComputeEngine(c *C) {
	ac := NewAvailabilityChecker()
	ac.SetProjectID(b.ProjectID)

	rate, err := ac.RateComputeEngine("service_role_webapp", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_ZONAL)

	rate, err = ac.RateComputeEngine("service_role_db", "true")
	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_ZONAL)
}

func (b *BenchmarkTestSuite) TestRateAppEngine(c *C) {
	ac := NewAvailabilityChecker()
	ac.SetProjectID(b.ProjectID)

	rate, err := ac.RateAppEngine("service_role_webapp", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_NO_RESOURCE)
}

func (b *BenchmarkTestSuite) TestRateCloudRun(c *C) {
	ac := NewAvailabilityChecker()
	ac.SetProjectID(b.ProjectID)

	rate, err := ac.RateCloudRun("service_role_webapp", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_NO_RESOURCE)
}

func (b *BenchmarkTestSuite) TestRateCloudFunctions(c *C) {
	ac := NewAvailabilityChecker()
	ac.SetProjectID(b.ProjectID)

	rate, err := ac.RateCloudFunctions("service_role_webapp", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_NO_RESOURCE)
}

func (b *BenchmarkTestSuite) TestRateCloudSQL(c *C) {
	projectID := b.ProjectID
	ac := NewAvailabilityChecker()
	ac.SetProjectID(projectID)

	rate, err := ac.RateCloudSQL("service_role_db", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_NO_RESOURCE)
}

func (b *BenchmarkTestSuite) TestRateCloudSpanner(c *C) {
	projectID := b.ProjectID
	ac := NewAvailabilityChecker()
	ac.SetProjectID(projectID)

	rate, err := ac.RateCloudSpanner("service_role_db", "true")

	c.Check(err, IsNil)
	c.Check(rate, Equals, RATE_NO_RESOURCE)
}

func (b *BenchmarkTestSuite) TestRateAvaiability(c *C) {
	regions := make(map[string]interface{})
	zones := make(map[string]interface{})

	c.Check(rateAvailability(regions, zones), Equals, RATE_NO_RESOURCE)

	zones["zone-1"] = struct{}{}
	c.Check(rateAvailability(regions, zones), Equals, RATE_ZONAL)

	zones["zone-2"] = struct{}{}
	c.Check(rateAvailability(regions, zones), Equals, RATE_REGIONAL)

	regions["region-1"] = struct{}{}
	c.Check(rateAvailability(regions, zones), Equals, RATE_REGIONAL)

	regions["region-2"] = struct{}{}
	c.Check(rateAvailability(regions, zones), Equals, RATE_MULTI_REGIONAL)
}

func (b *BenchmarkTestSuite) TestRateArchitecture(c *C) {
	projectID := b.ProjectID
	labels := map[string]string{
		"service_role_webapp": "true",
		"service_role_db":     "true",
	}

	pfRate, err := RateArchitecture(projectID, labels)

	c.Check(err, IsNil)
	c.Check(pfRate, Equals, RATE_NO_RESOURCE)
}
