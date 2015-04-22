package cloudfront

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/crowdmob/goamz/aws"
)

const (
	ServiceName = "cloudfront"
	ApiVersion  = "2014-11-06"
)

// Factory for the route53 type
func NewCloudFront(auth aws.Auth) (*CloudFront, error) {
	signer := aws.NewV4Signer(auth, "cloudfront", aws.USEast)

	return &CloudFront{
		Signer: signer,
		Auth:   auth,
	}, nil
}

type CloudFront struct {
	Signer    *aws.V4Signer
	Auth      aws.Auth
	BaseURL   string
	keyPairId string
	key       *rsa.PrivateKey
}

type DistributionConfig struct {
	XMLName              xml.Name `xml:"DistributionConfig"`
	CallerReference      string
	Aliases              Aliases
	DefaultRootObject    string
	Origins              Origins
	DefaultCacheBehavior CacheBehavior
	Comment              string
	CacheBehaviors       CacheBehaviors
	CustomErrorResponses CustomErrorResponses
	Restrictions         GeoRestriction `xml:"Restrictions>GeoRestriction"`
	Logging              Logging
	ViewerCertificate    *ViewerCertificate `xml:",omitempty"`
	PriceClass           string
	Enabled              bool
}

type Aliases []string

type EncodedAliases struct {
	Quantity int
	Items    []string `xml:"Items>CNAME"`
}

func (a Aliases) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedAliases{
		Quantity: len(a),
		Items:    []string(a),
	}

	return e.EncodeElement(enc, start)
}

type CustomErrorResponses []CustomErrorResponse

type EncodedCustomErrorResponses struct {
	Quantity int
	Items    []CustomErrorResponse `xml:"Items>CustomErrorResponse"`
}

func (a CustomErrorResponses) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedCustomErrorResponses{
		Quantity: len(a),
		Items:    []CustomErrorResponse(a),
	}

	return e.EncodeElement(enc, start)
}

type CacheBehaviors []CacheBehavior

type EncodedCacheBehaviors struct {
	Quantity int
	Items    []CacheBehavior `xml:"Items>CacheBehavior"`
}

func (a CacheBehaviors) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedCacheBehaviors{
		Quantity: len(a),
		Items:    []CacheBehavior(a),
	}

	return e.EncodeElement(enc, start)
}

type Logging struct {
	Enabled        bool
	IncludeCookies bool
	Bucket         string
	Prefix         string
}

type ViewerCertificate struct {
	IAMCertificateId             string `xml:",omitempty"`
	CloudFrontDefaultCertificate bool   `xml:",omitempty"`
	SSLSupportMethod             string
	MinimumProtocolVersion       string
}

type GeoRestriction struct {
	RestrictionType string
	Locations       []string
}

type EncodedGeoRestriction struct {
	RestrictionType string
	Quantity        int
	Locations       []string `xml:"Items>Location"`
}

func (a GeoRestriction) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedGeoRestriction{
		RestrictionType: a.RestrictionType,
		Quantity:        len(a.Locations),
		Locations:       []string(a.Locations),
	}

	return e.EncodeElement(enc, start)
}

type CustomErrorResponse struct {
	XMLName            xml.Name `xml:"CustomErrorResponse"`
	ErrorCode          int
	ResponsePagePath   string
	ResponseCode       int
	ErrorCachingMinTTL int
}

type Origin struct {
	XMLName            xml.Name `xml:"Origin"`
	Id                 string
	DomainName         string
	OriginPath         string              `xml:"OriginPath,omitempty"`
	S3OriginConfig     *S3OriginConfig     `xml:",omitempty"`
	CustomOriginConfig *CustomOriginConfig `xml:",omitempty"`
}

type S3OriginConfig struct {
	OriginAccessIdentity string
}

type CustomOriginConfig struct {
	HTTPPort             int
	HTTPSPort            int
	OriginProtocolPolicy string
}

type Origins []Origin

type EncodedOrigins struct {
	Quantity int
	Items    []Origin `xml:"Items>Origin"`
}

func (o Origins) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedOrigins{
		Quantity: len(o),
		Items:    []Origin(o),
	}

	return e.EncodeElement(enc, start)
}

type CacheBehavior struct {
	TargetOriginId       string
	PathPattern          string `xml:",omitempty"`
	ForwardedValues      ForwardedValues
	TrustedSigners       TrustedSigners
	ViewerProtocolPolicy string
	MinTTL               int
	AllowedMethods       AllowedMethods
	SmoothStreaming      bool
}

type ForwardedValues struct {
	QueryString bool
	Cookies     Cookies
	Headers     Names
}

type Cookies struct {
	Forward          string
	WhitelistedNames Names
}

type Names []string

type EncodedNames struct {
	Quantity int
	Items    []string `xml:"Items>Name"`
}

func (w Names) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedNames{
		Quantity: len(w),
		Items:    []string(w),
	}

	return e.EncodeElement(enc, start)
}

type ItemsList []string

type TrustedSigners struct {
	Enabled           bool
	AWSAccountNumbers []string
}

type EncodedTrustedSigners struct {
	Enabled  bool
	Quantity int
	Items    []string `xml:"Items>AWSAccountNumber"`
}

func (n TrustedSigners) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedTrustedSigners{
		Enabled:  n.Enabled,
		Quantity: len(n.AWSAccountNumbers),
		Items:    n.AWSAccountNumbers,
	}

	return e.EncodeElement(enc, start)
}

type AllowedMethods struct {
	Allowed []string `xml:"Items"`
	Cached  []string `xml:"CachedMethods>Items,omitempty"`
}

type EncodedAllowedMethods struct {
	AllowedQuantity int      `xml:"Quantity"`
	Allowed         []string `xml:"Items>Method"`
	CachedQuantity  int      `xml:"CachedMethods>Quantity"`
	Cached          []string `xml:"CachedMethods>Items>Method"`
}

func (n AllowedMethods) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	enc := EncodedAllowedMethods{
		AllowedQuantity: len(n.Allowed),
		Allowed:         n.Allowed,
		CachedQuantity:  len(n.Cached),
		Cached:          n.Cached,
	}

	return e.EncodeElement(enc, start)
}

var base64Replacer = strings.NewReplacer("=", "_", "+", "-", "/", "~")

func NewKeyLess(auth aws.Auth, baseurl string) *CloudFront {
	return &CloudFront{keyPairId: auth.AccessKey, BaseURL: baseurl}
}

func New(baseurl string, key *rsa.PrivateKey, keyPairId string) *CloudFront {
	return &CloudFront{
		BaseURL:   baseurl,
		keyPairId: keyPairId,
		key:       key,
	}
}

type epochTime struct {
	EpochTime int64 `json:"AWS:EpochTime"`
}

type condition struct {
	DateLessThan epochTime
}

type statement struct {
	Resource  string
	Condition condition
}

type policy struct {
	Statement []statement
}

func buildPolicy(resource string, expireTime time.Time) ([]byte, error) {
	p := &policy{
		Statement: []statement{
			statement{
				Resource: resource,
				Condition: condition{
					DateLessThan: epochTime{
						EpochTime: expireTime.Truncate(time.Millisecond).Unix(),
					},
				},
			},
		},
	}

	return json.Marshal(p)
}

func (cf *CloudFront) generateSignature(policy []byte) (string, error) {
	hash := sha1.New()
	_, err := hash.Write(policy)
	if err != nil {
		return "", err
	}

	hashed := hash.Sum(nil)
	var signed []byte
	if cf.key.Validate() == nil {
		signed, err = rsa.SignPKCS1v15(nil, cf.key, crypto.SHA1, hashed)
		if err != nil {
			return "", err
		}
	} else {
		signed = hashed
	}
	encoded := base64Replacer.Replace(base64.StdEncoding.EncodeToString(signed))

	return encoded, nil
}

func (cf *CloudFront) CreateDistribution(config DistributionConfig) error {
	if config.CallerReference == "" {
		config.CallerReference = strconv.FormatInt(time.Now().Unix(), 10)
	}

	body, err := xml.Marshal(config)
	if err != nil {
		return err
	}

	client := http.Client{}
	req, err := http.NewRequest("POST", "https://"+ServiceName+".amazonaws.com/"+ApiVersion+"/distribution", bytes.NewReader(body))
	if err != nil {
		return err
	}

	cf.Signer.Sign(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errors := aws.ErrorResponse{}
		xml.NewDecoder(resp.Body).Decode(&errors)

		err := errors.Errors
		err.RequestId = errors.RequestId
		err.StatusCode = resp.StatusCode
		if err.Message == "" {
			err.Message = resp.Status
		}
		return &err
	}
	return nil
}

// Creates a signed url using RSAwithSHA1 as specified by
// http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-creating-signed-url-canned-policy.html#private-content-canned-policy-creating-signature
func (cf *CloudFront) CannedSignedURL(path, queryString string, expires time.Time) (string, error) {
	resource := cf.BaseURL + path
	if queryString != "" {
		resource = path + "?" + queryString
	}

	policy, err := buildPolicy(resource, expires)
	if err != nil {
		return "", err
	}

	signature, err := cf.generateSignature(policy)
	if err != nil {
		return "", err
	}

	// TOOD: Do this once
	uri, err := url.Parse(cf.BaseURL)
	if err != nil {
		return "", err
	}

	uri.RawQuery = queryString
	if queryString != "" {
		uri.RawQuery += "&"
	}

	expireTime := expires.Truncate(time.Millisecond).Unix()

	uri.Path = path
	uri.RawQuery += fmt.Sprintf("Expires=%d&Signature=%s&Key-Pair-Id=%s", expireTime, signature, cf.keyPairId)

	return uri.String(), nil
}

func (cloudfront *CloudFront) SignedURL(path, querystrings string, expires time.Time) string {
	policy := `{"Statement":[{"Resource":"` + path + "?" + querystrings + `,"Condition":{"DateLessThan":{"AWS:EpochTime":` + strconv.FormatInt(expires.Truncate(time.Millisecond).Unix(), 10) + `}}}]}`

	hash := sha1.New()
	hash.Write([]byte(policy))
	b := hash.Sum(nil)
	he := base64.StdEncoding.EncodeToString(b)

	policySha1 := he

	url := cloudfront.BaseURL + path + "?" + querystrings + "&Expires=" + strconv.FormatInt(expires.Unix(), 10) + "&Signature=" + policySha1 + "&Key-Pair-Id=" + cloudfront.keyPairId

	return url
}
