package alicloud

import (
	"fmt"
	"log"
	"strings"

	"net/http"
	"os"
	"strconv"
	"time"

	"net/url"

	"regexp"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/resource"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ots"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/pvtz"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/fc-go-sdk"
	"github.com/denverdino/aliyungo/cdn"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/cs"
	"github.com/denverdino/aliyungo/dns"
	"github.com/denverdino/aliyungo/kms"
	"github.com/denverdino/aliyungo/location"
	"github.com/denverdino/aliyungo/ram"
	"github.com/hashicorp/terraform/terraform"
)

// Config of aliyun
type Config struct {
	AccessKey       string
	SecretKey       string
	Region          common.Region
	RegionId        string
	SecurityToken   string
	OtsInstanceName string
	LogEndpoint     string
	AccountId       string
	FcEndpoint      string
}

// AliyunClient of aliyun
type AliyunClient struct {
	Region   common.Region
	RegionId string
	//In order to build ots table client, add accesskey and secretkey in aliyunclient temporarily.
	AccessKey       string
	SecretKey       string
	SecurityToken   string
	OtsInstanceName string
	AccountId       string
	ecsconn         *ecs.Client
	essconn         *ess.Client
	rdsconn         *rds.Client
	vpcconn         *vpc.Client
	slbconn         *slb.Client
	ossconn         *oss.Client
	dnsconn         *dns.Client
	ramconn         ram.RamClientInterface
	csconn          *cs.Client
	cdnconn         *cdn.CdnClient
	kmsconn         *kms.Client
	otsconn         *ots.Client
	cmsconn         *cms.Client
	logconn         *sls.Client
	fcconn          *fc.Client
	pvtzconn        *pvtz.Client
	ddsconn         *dds.Client
}

// Client for AliyunClient
func (c *Config) Client() (*AliyunClient, error) {
	err := c.loadAndValidate()
	if err != nil {
		return nil, err
	}

	ecsconn, err := c.ecsConn()
	if err != nil {
		return nil, err
	}

	rdsconn, err := c.rdsConn()
	if err != nil {
		return nil, err
	}

	slbconn, err := c.slbConn()
	if err != nil {
		return nil, err
	}

	vpcconn, err := c.vpcConn()
	if err != nil {
		return nil, err
	}

	essconn, err := c.essConn()
	if err != nil {
		return nil, err
	}
	ossconn, err := c.ossConn()
	if err != nil {
		return nil, err
	}
	dnsconn, err := c.dnsConn()
	if err != nil {
		return nil, err
	}
	ramconn, err := c.ramConn()
	if err != nil {
		return nil, err
	}
	csconn, err := c.csConn()
	if err != nil {
		return nil, err
	}
	cdnconn, err := c.cdnConn()
	if err != nil {
		return nil, err
	}
	kmsconn, err := c.kmsConn()
	if err != nil {
		return nil, err
	}
	otsconn, err := c.otsConn()
	if err != nil {
		return nil, err
	}
	cmsconn, err := c.cmsConn()
	if err != nil {
		return nil, err
	}
	pvtzconn, err := c.pvtzConn()
	if err != nil {
		return nil, err
	}
	fcconn, err := c.fcConn()
	if err != nil {
		return nil, err
	}
	ddsconn, err := c.ddsConn()
	if err != nil {
		return nil, err
	}
	return &AliyunClient{
		Region:          c.Region,
		RegionId:        c.RegionId,
		AccessKey:       c.AccessKey,
		SecretKey:       c.SecretKey,
		SecurityToken:   c.SecurityToken,
		OtsInstanceName: c.OtsInstanceName,
		AccountId:       c.AccountId,
		ecsconn:         ecsconn,
		vpcconn:         vpcconn,
		slbconn:         slbconn,
		rdsconn:         rdsconn,
		essconn:         essconn,
		ossconn:         ossconn,
		dnsconn:         dnsconn,
		ramconn:         ramconn,
		csconn:          csconn,
		cdnconn:         cdnconn,
		kmsconn:         kmsconn,
		otsconn:         otsconn,
		cmsconn:         cmsconn,
		logconn:         c.logConn(),
		ddsconn:         ddsconn,
		fcconn:          fcconn,
		pvtzconn:        pvtzconn,
	}, nil
}

const BusinessInfoKey = "Terraform"

func (c *Config) loadAndValidate() error {
	err := c.validateRegion()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) validateRegion() error {

	for _, valid := range common.ValidRegions {
		if c.Region == valid {
			return nil
		}
	}

	return fmt.Errorf("Not a valid region: %s", c.Region)
}

func (c *Config) ecsConn() (client *ecs.Client, err error) {
	endpoint := LoadEndpoint(c.RegionId, ECSCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(ECSCode), endpoint)
	}
	client, err = ecs.NewClientWithOptions(c.RegionId, getSdkConfig().WithTimeout(60000000000), c.getAuthCredential(true))
	if err != nil {
		return
	}

	if _, err := client.DescribeRegions(ecs.CreateDescribeRegionsRequest()); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Config) rdsConn() (*rds.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, RDSCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(RDSCode), endpoint)
	}
	return rds.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}

func (c *Config) slbConn() (*slb.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, SLBCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(SLBCode), endpoint)
	}
	return slb.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}

func (c *Config) vpcConn() (*vpc.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, VPCCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(VPCCode), endpoint)
	}
	return vpc.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))

}
func (c *Config) essConn() (*ess.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, ESSCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(ESSCode), endpoint)
	}
	return ess.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}
func (c *Config) ossConn() (*oss.Client, error) {

	endpointClient := location.NewClient(c.AccessKey, c.SecretKey)
	endpointClient.SetSecurityToken(c.SecurityToken)
	endpoint := LoadEndpoint(c.RegionId, OSSCode)
	if endpoint == "" {
		args := &location.DescribeEndpointsArgs{
			Id:          c.Region,
			ServiceCode: "oss",
			Type:        "openAPI",
		}
		invoker := NewInvoker()
		var endpoints *location.DescribeEndpointsResponse
		if err := invoker.Run(func() error {
			es, err := endpointClient.DescribeEndpoints(args)
			if err != nil {
				return err
			}
			endpoints = es
			return nil
		}); err != nil {
			log.Printf("[DEBUG] Describe endpoint using region: %#v got an error: %#v.", c.Region, err)
		} else {
			if endpoints != nil && len(endpoints.Endpoints.Endpoint) > 0 {
				endpoint = strings.ToLower(endpoints.Endpoints.Endpoint[0].Protocols.Protocols[0]) + "://" + endpoints.Endpoints.Endpoint[0].Endpoint
			} else {
				endpoint = fmt.Sprintf("http://oss-%s.aliyuncs.com", c.Region)
			}
		}
	}

	log.Printf("[DEBUG] Instantiate OSS client using endpoint: %#v", endpoint)
	client, err := oss.New(endpoint, c.AccessKey, c.SecretKey, oss.UserAgent(getUserAgent()))

	return client, err
}

func (c *Config) dnsConn() (*dns.Client, error) {
	client := dns.NewClientNew(c.AccessKey, c.SecretKey)
	client.SetBusinessInfo(BusinessInfoKey)
	client.SetUserAgent(getUserAgent())
	return client, nil
}

func (c *Config) ramConn() (ram.RamClientInterface, error) {
	client := ram.NewClient(c.AccessKey, c.SecretKey)
	return client, nil
}

func (c *Config) csConn() (*cs.Client, error) {
	client := cs.NewClientForAussumeRole(c.AccessKey, c.SecretKey, c.SecurityToken)
	client.SetUserAgent(getUserAgent())
	return client, nil
}

func (c *Config) cdnConn() (*cdn.CdnClient, error) {
	client := cdn.NewClient(c.AccessKey, c.SecretKey)
	client.SetBusinessInfo(BusinessInfoKey)
	client.SetUserAgent(getUserAgent())
	return client, nil
}

func (c *Config) kmsConn() (*kms.Client, error) {
	client := kms.NewECSClientWithSecurityToken(c.AccessKey, c.SecretKey, c.SecurityToken, c.Region)
	client.SetBusinessInfo(BusinessInfoKey)
	client.SetUserAgent(getUserAgent())
	return client, nil
}

func (c *Config) otsConn() (*ots.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, OTSCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(OTSCode), endpoint)
	}
	return ots.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}

func (c *Config) cmsConn() (*cms.Client, error) {
	return cms.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(false))
}

func (c *Config) pvtzConn() (*pvtz.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, PVTZCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(PVTZCode), endpoint)
	} else {
		endpoints.AddEndpointMapping(c.RegionId, string(PVTZCode), "pvtz.aliyuncs.com")
	}
	return pvtz.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}

func (c *Config) logConn() *sls.Client {
	endpoint := c.LogEndpoint
	if endpoint == "" {
		endpoint = LoadEndpoint(c.RegionId, LOGCode)
		if endpoint == "" {
			endpoint = fmt.Sprintf("%s.log.aliyuncs.com", c.RegionId)
		}
	}

	return &sls.Client{
		AccessKeyID:     c.AccessKey,
		AccessKeySecret: c.SecretKey,
		Endpoint:        endpoint,
		SecurityToken:   c.SecurityToken,
		UserAgent:       getUserAgent(),
	}
}

func (c *Config) fcConn() (client *fc.Client, err error) {
	endpoint := c.LogEndpoint
	if endpoint == "" {
		endpoint = LoadEndpoint(c.RegionId, FCCode)
		if endpoint == "" {
			endpoint = fmt.Sprintf("%s.fc.aliyuncs.com", c.RegionId)
		}
	}

	config := getSdkConfig()
	client, err = fc.NewClient(fmt.Sprintf("%s%s%s", c.AccountId, DOT_SEPARATED, endpoint), ApiVersion20160815, c.AccessKey, c.SecretKey, fc.WithTransport(config.HttpTransport))
	if err != nil {
		return
	}
	client.Config.UserAgent = getUserAgent()
	client.Config.SecurityToken = c.SecurityToken
	return
}

func (c *Config) ddsConn() (*dds.Client, error) {
	endpoint := LoadEndpoint(c.RegionId, DDSCode)
	if endpoint != "" {
		endpoints.AddEndpointMapping(c.RegionId, string(DDSCode), endpoint)
	}
	return dds.NewClientWithOptions(c.RegionId, getSdkConfig(), c.getAuthCredential(true))
}

func getSdkConfig() *sdk.Config {
	// Fix bug "open /usr/local/go/lib/time/zoneinfo.zip: no such file or directory" which happened in windows.
	if data, ok := resource.GetTZData("GMT"); ok {
		utils.TZData = data
		utils.LoadLocationFromTZData = time.LoadLocationFromTZData
	}
	return sdk.NewConfig().
		WithMaxRetryTime(5).
		WithTimeout(time.Duration(30000000000)).
		WithUserAgent(getUserAgent()).
		WithGoRoutinePoolSize(10).
		WithDebug(false).
		WithHttpTransport(getTransport())
}

func (c *Config) getAuthCredential(stsSupported bool) auth.Credential {
	if stsSupported {
		return credentials.NewStsTokenCredential(c.AccessKey, c.SecretKey, c.SecurityToken)
	}

	return credentials.NewAccessKeyCredential(c.AccessKey, c.SecretKey)
}

func getUserAgent() string {
	return fmt.Sprintf("HashiCorp-Terraform-v%s", strings.TrimSuffix(terraform.VersionString(), "-dev"))
}

func getTransport() *http.Transport {
	handshakeTimeout, err := strconv.Atoi(os.Getenv("TLSHandshakeTimeout"))
	if err != nil {
		handshakeTimeout = 120
	}
	transport := &http.Transport{}
	transport.TLSHandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	// After building a new transport and it need to set http proxy to support proxy.
	for _, v := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		if value := Trim(os.Getenv(v)); value != "" {
			if !regexp.MustCompile(`^http(s)?://`).MatchString(value) {
				value = fmt.Sprintf("http://%s", value)
			}
			proxyUrl, err := url.Parse(value)
			if err != nil {
				return transport
			}
			transport.Proxy = http.ProxyURL(proxyUrl)
			break
		}
	}
	return transport
}
