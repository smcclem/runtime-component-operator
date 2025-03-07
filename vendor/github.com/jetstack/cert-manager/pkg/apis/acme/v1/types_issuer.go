/*
Copyright 2020 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
)

// ACMEIssuer contains the specification for an ACME issuer.
// This uses the RFC8555 specification to obtain certificates by completing
// 'challenges' to prove ownership of domain identifiers.
// Earlier draft versions of the ACME specification are not supported.
type ACMEIssuer struct {
	// Email is the email address to be associated with the ACME account.
	// This field is optional, but it is strongly recommended to be set.
	// It will be used to contact you in case of issues with your account or
	// certificates, including expiry notification emails.
	// This field may be updated after the account is initially registered.
	// +optional
	Email string `json:"email,omitempty"`

	// Server is the URL used to access the ACME server's 'directory' endpoint.
	// For example, for Let's Encrypt's staging endpoint, you would use:
	// "https://acme-staging-v02.api.letsencrypt.org/directory".
	// Only ACME v2 endpoints (i.e. RFC 8555) are supported.
	Server string `json:"server"`

	// PreferredChain is the chain to use if the ACME server outputs multiple.
	// PreferredChain is no guarantee that this one gets delivered by the ACME
	// endpoint.
	// For example, for Let's Encrypt's DST crosssign you would use:
	// "DST Root CA X3" or "ISRG Root X1" for the newer Let's Encrypt root CA.
	// This value picks the first certificate bundle in the ACME alternative
	// chains that has a certificate with this value as its issuer's CN
	// +optional
	// +kubebuilder:validation:MaxLength=64
	PreferredChain string `json:"preferredChain"`

	// Enables or disables validation of the ACME server TLS certificate.
	// If true, requests to the ACME server will not have their TLS certificate
	// validated (i.e. insecure connections will be allowed).
	// Only enable this option in development environments.
	// The cert-manager system installed roots will be used to verify connections
	// to the ACME server if this is false.
	// Defaults to false.
	// +optional
	SkipTLSVerify bool `json:"skipTLSVerify,omitempty"`

	// ExternalAccountBinding is a reference to a CA external account of the ACME
	// server.
	// If set, upon registration cert-manager will attempt to associate the given
	// external account credentials with the registered ACME account.
	// +optional
	ExternalAccountBinding *ACMEExternalAccountBinding `json:"externalAccountBinding,omitempty"`

	// PrivateKey is the name of a Kubernetes Secret resource that will be used to
	// store the automatically generated ACME account private key.
	// Optionally, a `key` may be specified to select a specific entry within
	// the named Secret resource.
	// If `key` is not specified, a default of `tls.key` will be used.
	PrivateKey cmmeta.SecretKeySelector `json:"privateKeySecretRef"`

	// Solvers is a list of challenge solvers that will be used to solve
	// ACME challenges for the matching domains.
	// Solver configurations must be provided in order to obtain certificates
	// from an ACME server.
	// For more information, see: https://cert-manager.io/docs/configuration/acme/
	// +optional
	Solvers []ACMEChallengeSolver `json:"solvers,omitempty"`

	// Enables or disables generating a new ACME account key.
	// If true, the Issuer resource will *not* request a new account but will expect
	// the account key to be supplied via an existing secret.
	// If false, the cert-manager system will generate a new ACME account key
	// for the Issuer.
	// Defaults to false.
	// +optional
	DisableAccountKeyGeneration bool `json:"disableAccountKeyGeneration,omitempty"`

	// Enables requesting a Not After date on certificates that matches the
	// duration of the certificate. This is not supported by all ACME servers
	// like Let's Encrypt. If set to true when the ACME server does not support
	// it it will create an error on the Order.
	// Defaults to false.
	// +optional
	EnableDurationFeature bool `json:"enableDurationFeature,omitempty"`
}

// ACMEExternalAccountBinding is a reference to a CA external account of the ACME
// server.
type ACMEExternalAccountBinding struct {
	// keyID is the ID of the CA key that the External Account is bound to.
	KeyID string `json:"keyID"`

	// keySecretRef is a Secret Key Selector referencing a data item in a Kubernetes
	// Secret which holds the symmetric MAC key of the External Account Binding.
	// The `key` is the index string that is paired with the key data in the
	// Secret and should not be confused with the key data itself, or indeed with
	// the External Account Binding keyID above.
	// The secret key stored in the Secret **must** be un-padded, base64 URL
	// encoded data.
	Key cmmeta.SecretKeySelector `json:"keySecretRef"`

	// Deprecated: keyAlgorithm field exists for historical compatibility
	// reasons and should not be used. The algorithm is now hardcoded to HS256
	// in golang/x/crypto/acme.
	// +optional
	KeyAlgorithm HMACKeyAlgorithm `json:"keyAlgorithm,omitempty"`
}

// HMACKeyAlgorithm is the name of a key algorithm used for HMAC encryption
// +kubebuilder:validation:Enum=HS256;HS384;HS512
type HMACKeyAlgorithm string

const (
	HS256 HMACKeyAlgorithm = "HS256"
	HS384 HMACKeyAlgorithm = "HS384"
	HS512 HMACKeyAlgorithm = "HS512"
)

// An ACMEChallengeSolver describes how to solve ACME challenges for the issuer it is part of.
// A selector may be provided to use different solving strategies for different DNS names.
// Only one of HTTP01 or DNS01 must be provided.
type ACMEChallengeSolver struct {
	// Selector selects a set of DNSNames on the Certificate resource that
	// should be solved using this challenge solver.
	// If not specified, the solver will be treated as the 'default' solver
	// with the lowest priority, i.e. if any other solver has a more specific
	// match, it will be used instead.
	// +optional
	Selector *CertificateDNSNameSelector `json:"selector,omitempty"`

	// Configures cert-manager to attempt to complete authorizations by
	// performing the HTTP01 challenge flow.
	// It is not possible to obtain certificates for wildcard domain names
	// (e.g. `*.example.com`) using the HTTP01 challenge mechanism.
	// +optional
	HTTP01 *ACMEChallengeSolverHTTP01 `json:"http01,omitempty"`

	// Configures cert-manager to attempt to complete authorizations by
	// performing the DNS01 challenge flow.
	// +optional
	DNS01 *ACMEChallengeSolverDNS01 `json:"dns01,omitempty"`
}

// CertificateDomainSelector selects certificates using a label selector, and
// can optionally select individual DNS names within those certificates.
// If both MatchLabels and DNSNames are empty, this selector will match all
// certificates and DNS names within them.
type CertificateDNSNameSelector struct {
	// A label selector that is used to refine the set of certificate's that
	// this challenge solver will apply to.
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	// List of DNSNames that this solver will be used to solve.
	// If specified and a match is found, a dnsNames selector will take
	// precedence over a dnsZones selector.
	// If multiple solvers match with the same dnsNames value, the solver
	// with the most matching labels in matchLabels will be selected.
	// If neither has more matches, the solver defined earlier in the list
	// will be selected.
	// +optional
	DNSNames []string `json:"dnsNames,omitempty"`

	// List of DNSZones that this solver will be used to solve.
	// The most specific DNS zone match specified here will take precedence
	// over other DNS zone matches, so a solver specifying sys.example.com
	// will be selected over one specifying example.com for the domain
	// www.sys.example.com.
	// If multiple solvers match with the same dnsZones value, the solver
	// with the most matching labels in matchLabels will be selected.
	// If neither has more matches, the solver defined earlier in the list
	// will be selected.
	// +optional
	DNSZones []string `json:"dnsZones,omitempty"`
}

// ACMEChallengeSolverHTTP01 contains configuration detailing how to solve
// HTTP01 challenges within a Kubernetes cluster.
// Typically this is accomplished through creating 'routes' of some description
// that configure ingress controllers to direct traffic to 'solver pods', which
// are responsible for responding to the ACME server's HTTP requests.
// Only one of Ingress / Gateway can be specified.
type ACMEChallengeSolverHTTP01 struct {
	// The ingress based HTTP01 challenge solver will solve challenges by
	// creating or modifying Ingress resources in order to route requests for
	// '/.well-known/acme-challenge/XYZ' to 'challenge solver' pods that are
	// provisioned by cert-manager for each Challenge to be completed.
	// +optional
	Ingress *ACMEChallengeSolverHTTP01Ingress `json:"ingress,omitempty"`

	// The Gateway API is a sig-network community API that models service networking
	// in Kubernetes (https://gateway-api.sigs.k8s.io/). The Gateway solver will
	// create HTTPRoutes with the specified labels in the same namespace as the challenge.
	// This solver is experimental, and fields / behaviour may change in the future.
	// +optional
	GatewayHTTPRoute *ACMEChallengeSolverHTTP01GatewayHTTPRoute `json:"gatewayHTTPRoute,omitempty"`
}

type ACMEChallengeSolverHTTP01Ingress struct {
	// Optional service type for Kubernetes solver service. Supported values
	// are NodePort or ClusterIP (default).
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// The ingress class to use when creating Ingress resources to solve ACME
	// challenges that use this challenge solver.
	// Only one of 'class' or 'name' may be specified.
	// +optional
	Class *string `json:"class,omitempty"`

	// The name of the ingress resource that should have ACME challenge solving
	// routes inserted into it in order to solve HTTP01 challenges.
	// This is typically used in conjunction with ingress controllers like
	// ingress-gce, which maintains a 1:1 mapping between external IPs and
	// ingress resources.
	// +optional
	Name string `json:"name,omitempty"`

	// Optional pod template used to configure the ACME challenge solver pods
	// used for HTTP01 challenges.
	// +optional
	PodTemplate *ACMEChallengeSolverHTTP01IngressPodTemplate `json:"podTemplate,omitempty"`

	// Optional ingress template used to configure the ACME challenge solver
	// ingress used for HTTP01 challenges.
	// +optional
	IngressTemplate *ACMEChallengeSolverHTTP01IngressTemplate `json:"ingressTemplate,omitempty"`
}

// The ACMEChallengeSolverHTTP01GatewayHTTPRoute solver will create HTTPRoute objects for a Gateway class
// routing to an ACME challenge solver pod.
type ACMEChallengeSolverHTTP01GatewayHTTPRoute struct {
	// Optional service type for Kubernetes solver service. Supported values
	// are NodePort or ClusterIP (default).
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// The labels that cert-manager will use when creating the temporary
	// HTTPRoute needed for solving the HTTP-01 challenge. These labels
	// must match the label selector of at least one Gateway.
	Labels map[string]string `json:"labels,omitempty"`
}

type ACMEChallengeSolverHTTP01IngressPodTemplate struct {
	// ObjectMeta overrides for the pod used to solve HTTP01 challenges.
	// Only the 'labels' and 'annotations' fields may be set.
	// If labels or annotations overlap with in-built values, the values here
	// will override the in-built values.
	// +optional
	ACMEChallengeSolverHTTP01IngressPodObjectMeta `json:"metadata"`

	// PodSpec defines overrides for the HTTP01 challenge solver pod.
	// Only the 'priorityClassName', 'nodeSelector', 'affinity',
	// 'serviceAccountName' and 'tolerations' fields are supported currently.
	// All other fields will be ignored.
	// +optional
	Spec ACMEChallengeSolverHTTP01IngressPodSpec `json:"spec"`
}

type ACMEChallengeSolverHTTP01IngressPodObjectMeta struct {
	// Annotations that should be added to the create ACME HTTP01 solver pods.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels that should be added to the created ACME HTTP01 solver pods.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type ACMEChallengeSolverHTTP01IngressPodSpec struct {
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// If specified, the pod's priorityClassName.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// If specified, the pod's service account
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type ACMEChallengeSolverHTTP01IngressTemplate struct {
	// ObjectMeta overrides for the ingress used to solve HTTP01 challenges.
	// Only the 'labels' and 'annotations' fields may be set.
	// If labels or annotations overlap with in-built values, the values here
	// will override the in-built values.
	// +optional
	ACMEChallengeSolverHTTP01IngressObjectMeta `json:"metadata"`
}

type ACMEChallengeSolverHTTP01IngressObjectMeta struct {
	// Annotations that should be added to the created ACME HTTP01 solver ingress.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels that should be added to the created ACME HTTP01 solver ingress.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// Used to configure a DNS01 challenge provider to be used when solving DNS01
// challenges.
// Only one DNS provider may be configured per solver.
type ACMEChallengeSolverDNS01 struct {
	// CNAMEStrategy configures how the DNS01 provider should handle CNAME
	// records when found in DNS zones.
	// +optional
	CNAMEStrategy CNAMEStrategy `json:"cnameStrategy,omitempty"`

	// Use the Akamai DNS zone management API to manage DNS01 challenge records.
	// +optional
	Akamai *ACMEIssuerDNS01ProviderAkamai `json:"akamai,omitempty"`

	// Use the Google Cloud DNS API to manage DNS01 challenge records.
	// +optional
	CloudDNS *ACMEIssuerDNS01ProviderCloudDNS `json:"cloudDNS,omitempty"`

	// Use the Cloudflare API to manage DNS01 challenge records.
	// +optional
	Cloudflare *ACMEIssuerDNS01ProviderCloudflare `json:"cloudflare,omitempty"`

	// Use the AWS Route53 API to manage DNS01 challenge records.
	// +optional
	Route53 *ACMEIssuerDNS01ProviderRoute53 `json:"route53,omitempty"`

	// Use the Microsoft Azure DNS API to manage DNS01 challenge records.
	// +optional
	AzureDNS *ACMEIssuerDNS01ProviderAzureDNS `json:"azureDNS,omitempty"`

	// Use the DigitalOcean DNS API to manage DNS01 challenge records.
	// +optional
	DigitalOcean *ACMEIssuerDNS01ProviderDigitalOcean `json:"digitalocean,omitempty"`

	// Use the 'ACME DNS' (https://github.com/joohoi/acme-dns) API to manage
	// DNS01 challenge records.
	// +optional
	AcmeDNS *ACMEIssuerDNS01ProviderAcmeDNS `json:"acmeDNS,omitempty"`

	// Use RFC2136 ("Dynamic Updates in the Domain Name System") (https://datatracker.ietf.org/doc/rfc2136/)
	// to manage DNS01 challenge records.
	// +optional
	RFC2136 *ACMEIssuerDNS01ProviderRFC2136 `json:"rfc2136,omitempty"`

	// Configure an external webhook based DNS01 challenge solver to manage
	// DNS01 challenge records.
	// +optional
	Webhook *ACMEIssuerDNS01ProviderWebhook `json:"webhook,omitempty"`
}

// CNAMEStrategy configures how the DNS01 provider should handle CNAME records
// when found in DNS zones.
// By default, the None strategy will be applied (i.e. do not follow CNAMEs).
// +kubebuilder:validation:Enum=None;Follow
type CNAMEStrategy string

const (
	// NoneStrategy indicates that no CNAME resolution strategy should be used
	// when determining which DNS zone to update during DNS01 challenges.
	NoneStrategy = "None"

	// FollowStrategy will cause cert-manager to recurse through CNAMEs in
	// order to determine which DNS zone to update during DNS01 challenges.
	// This is useful if you do not want to grant cert-manager access to your
	// root DNS zone, and instead delegate the _acme-challenge.example.com
	// subdomain to some other, less privileged domain.
	FollowStrategy = "Follow"
)

// ACMEIssuerDNS01ProviderAkamai is a structure containing the DNS
// configuration for Akamai DNS—Zone Record Management API
type ACMEIssuerDNS01ProviderAkamai struct {
	ServiceConsumerDomain string                   `json:"serviceConsumerDomain"`
	ClientToken           cmmeta.SecretKeySelector `json:"clientTokenSecretRef"`
	ClientSecret          cmmeta.SecretKeySelector `json:"clientSecretSecretRef"`
	AccessToken           cmmeta.SecretKeySelector `json:"accessTokenSecretRef"`
}

// ACMEIssuerDNS01ProviderCloudDNS is a structure containing the DNS
// configuration for Google Cloud DNS
type ACMEIssuerDNS01ProviderCloudDNS struct {
	// +optional
	ServiceAccount *cmmeta.SecretKeySelector `json:"serviceAccountSecretRef,omitempty"`
	Project        string                    `json:"project"`

	// HostedZoneName is an optional field that tells cert-manager in which
	// Cloud DNS zone the challenge record has to be created.
	// If left empty cert-manager will automatically choose a zone.
	// +optional
	HostedZoneName string `json:"hostedZoneName,omitempty"`
}

// ACMEIssuerDNS01ProviderCloudflare is a structure containing the DNS
// configuration for Cloudflare.
// One of `apiKeySecretRef` or `apiTokenSecretRef` must be provided.
type ACMEIssuerDNS01ProviderCloudflare struct {
	// Email of the account, only required when using API key based authentication.
	// +optional
	Email string `json:"email,omitempty"`

	// API key to use to authenticate with Cloudflare.
	// Note: using an API token to authenticate is now the recommended method
	// as it allows greater control of permissions.
	// +optional
	APIKey *cmmeta.SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// API token used to authenticate with Cloudflare.
	// +optional
	APIToken *cmmeta.SecretKeySelector `json:"apiTokenSecretRef,omitempty"`
}

// ACMEIssuerDNS01ProviderDigitalOcean is a structure containing the DNS
// configuration for DigitalOcean Domains
type ACMEIssuerDNS01ProviderDigitalOcean struct {
	Token cmmeta.SecretKeySelector `json:"tokenSecretRef"`
}

// ACMEIssuerDNS01ProviderRoute53 is a structure containing the Route 53
// configuration for AWS
type ACMEIssuerDNS01ProviderRoute53 struct {
	// The AccessKeyID is used for authentication. If not set we fall-back to using env vars, shared credentials file or AWS Instance metadata
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	AccessKeyID string `json:"accessKeyID,omitempty"`

	// The SecretAccessKey is used for authentication. If not set we fall-back to using env vars, shared credentials file or AWS Instance metadata
	// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	SecretAccessKey cmmeta.SecretKeySelector `json:"secretAccessKeySecretRef"`

	// Role is a Role ARN which the Route53 provider will assume using either the explicit credentials AccessKeyID/SecretAccessKey
	// or the inferred credentials from environment variables, shared credentials file or AWS Instance metadata
	// +optional
	Role string `json:"role,omitempty"`

	// If set, the provider will manage only this zone in Route53 and will not do an lookup using the route53:ListHostedZonesByName api call.
	// +optional
	HostedZoneID string `json:"hostedZoneID,omitempty"`

	// Always set the region when using AccessKeyID and SecretAccessKey
	Region string `json:"region"`
}

// ACMEIssuerDNS01ProviderAzureDNS is a structure containing the
// configuration for Azure DNS
type ACMEIssuerDNS01ProviderAzureDNS struct {
	// if both this and ClientSecret are left unset MSI will be used
	// +optional
	ClientID string `json:"clientID,omitempty"`

	// if both this and ClientID are left unset MSI will be used
	// +optional
	ClientSecret *cmmeta.SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	SubscriptionID string `json:"subscriptionID"`

	// when specifying ClientID and ClientSecret then this field is also needed
	// +optional
	TenantID string `json:"tenantID,omitempty"`

	ResourceGroupName string `json:"resourceGroupName"`

	// +optional
	HostedZoneName string `json:"hostedZoneName,omitempty"`

	// +optional
	Environment AzureDNSEnvironment `json:"environment,omitempty"`
}

// +kubebuilder:validation:Enum=AzurePublicCloud;AzureChinaCloud;AzureGermanCloud;AzureUSGovernmentCloud
type AzureDNSEnvironment string

const (
	AzurePublicCloud       AzureDNSEnvironment = "AzurePublicCloud"
	AzureChinaCloud        AzureDNSEnvironment = "AzureChinaCloud"
	AzureGermanCloud       AzureDNSEnvironment = "AzureGermanCloud"
	AzureUSGovernmentCloud AzureDNSEnvironment = "AzureUSGovernmentCloud"
)

// ACMEIssuerDNS01ProviderAcmeDNS is a structure containing the
// configuration for ACME-DNS servers
type ACMEIssuerDNS01ProviderAcmeDNS struct {
	Host string `json:"host"`

	AccountSecret cmmeta.SecretKeySelector `json:"accountSecretRef"`
}

// ACMEIssuerDNS01ProviderRFC2136 is a structure containing the
// configuration for RFC2136 DNS
type ACMEIssuerDNS01ProviderRFC2136 struct {
	// The IP address or hostname of an authoritative DNS server supporting
	// RFC2136 in the form host:port. If the host is an IPv6 address it must be
	// enclosed in square brackets (e.g [2001:db8::1]) ; port is optional.
	// This field is required.
	Nameserver string `json:"nameserver"`

	// The name of the secret containing the TSIG value.
	// If ``tsigKeyName`` is defined, this field is required.
	// +optional
	TSIGSecret cmmeta.SecretKeySelector `json:"tsigSecretSecretRef,omitempty"`

	// The TSIG Key name configured in the DNS.
	// If ``tsigSecretSecretRef`` is defined, this field is required.
	// +optional
	TSIGKeyName string `json:"tsigKeyName,omitempty"`

	// The TSIG Algorithm configured in the DNS supporting RFC2136. Used only
	// when ``tsigSecretSecretRef`` and ``tsigKeyName`` are defined.
	// Supported values are (case-insensitive): ``HMACMD5`` (default),
	// ``HMACSHA1``, ``HMACSHA256`` or ``HMACSHA512``.
	// +optional
	TSIGAlgorithm string `json:"tsigAlgorithm,omitempty"`
}

// ACMEIssuerDNS01ProviderWebhook specifies configuration for a webhook DNS01
// provider, including where to POST ChallengePayload resources.
type ACMEIssuerDNS01ProviderWebhook struct {
	// The API group name that should be used when POSTing ChallengePayload
	// resources to the webhook apiserver.
	// This should be the same as the GroupName specified in the webhook
	// provider implementation.
	GroupName string `json:"groupName"`

	// The name of the solver to use, as defined in the webhook provider
	// implementation.
	// This will typically be the name of the provider, e.g. 'cloudflare'.
	SolverName string `json:"solverName"`

	// Additional configuration that should be passed to the webhook apiserver
	// when challenges are processed.
	// This can contain arbitrary JSON data.
	// Secret values should not be specified in this stanza.
	// If secret values are needed (e.g. credentials for a DNS service), you
	// should use a SecretKeySelector to reference a Secret resource.
	// For details on the schema of this field, consult the webhook provider
	// implementation's documentation.
	// +optional
	Config *apiextensionsv1.JSON `json:"config,omitempty"`
}

type ACMEIssuerStatus struct {
	// URI is the unique account identifier, which can also be used to retrieve
	// account details from the CA
	// +optional
	URI string `json:"uri,omitempty"`

	// LastRegisteredEmail is the email associated with the latest registered
	// ACME account, in order to track changes made to registered account
	// associated with the  Issuer
	// +optional
	LastRegisteredEmail string `json:"lastRegisteredEmail,omitempty"`
}
