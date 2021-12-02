package utils

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	appstacksv1beta2 "github.com/application-stacks/runtime-component-operator/api/v1beta2"
	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	name                     = "my-app"
	namespace                = "runtime"
	labels                   = map[string]string{"key1": "value1"}
	annotations              = map[string]string{"key2": "value2"}
	objMeta                  = metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}
	objMetaAnnos             = metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels, Annotations: annotations}
	applicationVersion       = "testing"
	stack                    = "java-microprofile"
	appImage                 = "my-image"
	replicas           int32 = 2
	expose                   = true
	createKNS                = true
	targetCPUPer       int32 = 30
	targetPort         int32 = 3333
	nodePort           int32 = 3011
	ports                    = []corev1.ServicePort{{Name: "https", Port: 9080, TargetPort: intstr.FromInt(9000)}, {Port: targetPort}}
	targetHelper       int32 = 9000
	svcPortName              = "myservice"
	serviceType2             = corev1.ServiceTypeNodePort
	autoscaling              = &appstacksv1beta2.RuntimeComponentAutoScaling{
		TargetCPUUtilizationPercentage: &targetCPUPer,
		MinReplicas:                    &replicas,
		MaxReplicas:                    3,
	}
	envFrom            = []corev1.EnvFromSource{{Prefix: namespace}}
	env                = []corev1.EnvVar{{Name: namespace}}
	pullPolicy         = corev1.PullAlways
	pullSecret         = "mysecret"
	serviceAccountName = "service-account"
	serviceType        = corev1.ServiceTypeClusterIP
	service            = &appstacksv1beta2.RuntimeComponentService{Type: &serviceType, Port: 8443}
	deploymentAnnos    = map[string]string{"depAnno": "depAnno"}
	ssAnnos            = map[string]string{"setAnno": "setAnno"}
	volumeCT           = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc", Namespace: namespace},
		TypeMeta:   metav1.TypeMeta{Kind: "StatefulSet"}}
	storage        = appstacksv1beta2.RuntimeComponentStorage{Size: "10Mi", MountPath: "/mnt/data", VolumeClaimTemplate: volumeCT}
	arch           = []string{"ppc64le"}
	readinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet:   &corev1.HTTPGetAction{},
			TCPSocket: &corev1.TCPSocketAction{},
		},
	}
	livenessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet:   &corev1.HTTPGetAction{},
			TCPSocket: &corev1.TCPSocketAction{},
		},
	}
	startupProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet:   &corev1.HTTPGetAction{},
			TCPSocket: &corev1.TCPSocketAction{},
		},
	}
	probes = &appstacksv1beta2.RuntimeComponentProbes{
		Readiness: readinessProbe,
		Liveness:  livenessProbe,
		Startup:   startupProbe}
	volume      = corev1.Volume{Name: "runtime-volume"}
	volumeMount = corev1.VolumeMount{Name: volumeCT.Name, MountPath: storage.MountPath}
	resLimits   = map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU: {},
	}
	resourceContraints = &corev1.ResourceRequirements{Limits: resLimits}
	key                = "key"
	crt                = "crt"
	ca                 = "ca"
	destCACert         = "destCACert"
	emptyString        = ""
)

type Test struct {
	test     string
	expected interface{}
	actual   interface{}
}

func TestCustomizeDeployment(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{ApplicationName: appImage, ApplicationVersion: applicationVersion, Replicas: &replicas}
	deployment, runtime := appsv1.Deployment{}, createRuntimeComponent(objMeta, spec)
	CustomizeDeployment(&deployment, runtime)

	// Test Deployment Spec
	testDP := []Test{
		{"Deployment selector", name, deployment.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]},
		{"Deployment labels", name, deployment.Labels["app.kubernetes.io/name"]},
		{"Deployment labels", "value1", deployment.Labels["key1"]},
		{"Deployment replicas", replicas, *deployment.Spec.Replicas},
	}
	verifyTests(testDP, t)

	deploymentStrategyTests(t)
}

func deploymentStrategyTests(t *testing.T) {
	// Set updateStrategy to RollingUpdate
	rollingUpdate := appsv1.RollingUpdateDeployment{MaxSurge: &intstr.IntOrString{IntVal: 1}, MaxUnavailable: &intstr.IntOrString{IntVal: 2}}
	updateStrategy := appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType, RollingUpdate: &rollingUpdate}
	runtimeDeployment := &appstacksv1beta2.RuntimeComponentDeployment{UpdateStrategy: &updateStrategy}
	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Deployment:       runtimeDeployment,
		Replicas:         &replicas,
	}
	deployment, runtime := appsv1.Deployment{}, createRuntimeComponent(objMeta, spec)
	CustomizeDeployment(&deployment, runtime)

	strategy := deployment.Spec.Strategy.Type
	maxSurge := deployment.Spec.Strategy.RollingUpdate.MaxSurge.IntVal
	maxUnavailable := (deployment.Spec.Strategy.RollingUpdate.MaxUnavailable).IntVal

	testStrategy1 := []Test{
		{"Strategy should be RollingUpdate", appsv1.RollingUpdateDeploymentStrategyType, strategy},
		{"MaxSurge should be 1", int32(1), maxSurge},
		{"MaxUnavailbale should be 2", int32(2), maxUnavailable},
	}
	verifyTests(testStrategy1, t)

	// Set updateStrategy to Recreate
	updateStrategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	runtimeDeployment = &appstacksv1beta2.RuntimeComponentDeployment{UpdateStrategy: &updateStrategy}
	spec.Deployment = runtimeDeployment
	deployment, runtime = appsv1.Deployment{}, createRuntimeComponent(objMeta, spec)
	CustomizeDeployment(&deployment, runtime)

	strategy = deployment.Spec.Strategy.Type

	testStrategy2 := []Test{
		{"Strategy should be Recreate", appsv1.RecreateDeploymentStrategyType, strategy},
	}
	verifyTests(testStrategy2, t)

	// Remove UpdateStrategy
	runtimeDeployment = &appstacksv1beta2.RuntimeComponentDeployment{}
	spec.Deployment = runtimeDeployment
	deployment, runtime = appsv1.Deployment{}, createRuntimeComponent(objMeta, spec)
	CustomizeDeployment(&deployment, runtime)

	strategy = deployment.Spec.Strategy.Type

	testStrategy3 := []Test{
		{"Strategy should be RollingUpdate", appsv1.RollingUpdateDeploymentStrategyType, strategy},
	}
	verifyTests(testStrategy3, t)
}

func TestCustomizeStatefulSet(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{ApplicationName: appImage, Service: service, ApplicationVersion: applicationVersion, Replicas: &replicas}
	statefulset, runtime := appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)
	CustomizeStatefulSet(&statefulset, runtime)

	// Test StatefulSet Spec
	testSS := []Test{
		{"Statefulset replicas", replicas, *statefulset.Spec.Replicas},
		{"Statefulset selector", name, statefulset.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]},
		{"Statefulset labels", name, statefulset.Labels["app.kubernetes.io/name"]},
		{"Statefulset labels", "value1", statefulset.Labels["key1"]},
		{"Statefulset service name", name + "-headless", statefulset.Spec.ServiceName},
	}
	verifyTests(testSS, t)

	statefulSetStrategyTests(t)
}

func statefulSetStrategyTests(t *testing.T) {
	var partition int32 = 2

	// Set updateStrategy to RollingUpdate
	rollingUpdate := appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition}
	updateStrategy := appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType, RollingUpdate: &rollingUpdate}
	runtimeStatefulSet := &appstacksv1beta2.RuntimeComponentStatefulSet{UpdateStrategy: &updateStrategy}
	spec = appstacksv1beta2.RuntimeComponentSpec{ApplicationImage: appImage, StatefulSet: runtimeStatefulSet, Replicas: &replicas}
	statefulSet, runtime := appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)
	CustomizeStatefulSet(&statefulSet, runtime)

	strategy := statefulSet.Spec.UpdateStrategy.Type
	partitionVal := *statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition

	testStrategy1 := []Test{
		{"Strategy should be RollingUpdate", appsv1.RollingUpdateStatefulSetStrategyType, strategy},
		{"Partition should be 1", int32(2), partitionVal},
	}
	verifyTests(testStrategy1, t)

	// Set updateStrategy to OnDelete
	updateStrategy = appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType}
	runtimeStatefulSet = &appstacksv1beta2.RuntimeComponentStatefulSet{UpdateStrategy: &updateStrategy}
	spec.StatefulSet = runtimeStatefulSet
	statefulSet, runtime = appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)
	CustomizeStatefulSet(&statefulSet, runtime)

	strategy = statefulSet.Spec.UpdateStrategy.Type

	testStrategy2 := []Test{
		{"Strategy should be OnDelete", appsv1.OnDeleteStatefulSetStrategyType, strategy},
	}
	verifyTests(testStrategy2, t)

	// Remove UpdateStrategy
	runtimeStatefulSet = &appstacksv1beta2.RuntimeComponentStatefulSet{}
	spec.StatefulSet = runtimeStatefulSet
	statefulSet, runtime = appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)
	CustomizeStatefulSet(&statefulSet, runtime)

	strategy = statefulSet.Spec.UpdateStrategy.Type

	testStrategy3 := []Test{
		{"Strategy should be RollingUpdate", appsv1.RollingUpdateStatefulSetStrategyType, strategy},
	}
	verifyTests(testStrategy3, t)
}

func TestCustomizeRoute(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service, Expose: &expose}
	route, runtime := &routev1.Route{}, createRuntimeComponent(objMetaAnnos, spec)

	//t.Errorf("route: (%v)", *route)
	CustomizeRoute(route, runtime, "", "", "", "")

	//TestGetLabels
	testCR := []Test{
		{"Route labels", name, route.Labels["app.kubernetes.io/instance"]},
		{"Route annotations", annotations, route.Annotations},
		{"Route target kind", "Service", route.Spec.To.Kind},
		{"Route target name", name, route.Spec.To.Name},
		{"Route target weight", int32(100), *route.Spec.To.Weight},
		{"Route service target port", intstr.FromString(strconv.Itoa(int(runtime.Spec.Service.Port)) + "-tcp"), route.Spec.Port.TargetPort},
	}
	verifyTests(testCR, t)

	helper := routev1.TLSTerminationEdge
	helper2 := routev1.InsecureEdgeTerminationPolicyNone
	runtime.Spec.Route = &appstacksv1beta2.RuntimeComponentRoute{Host: "routeHost", Path: "routePath", Termination: &helper, InsecureEdgeTerminationPolicy: &helper2}

	CustomizeRoute(route, runtime, key, crt, ca, destCACert)

	//TestEdge
	testCR = []Test{
		{"Route host", "routeHost", route.Spec.Host},
		{"Route host", "routePath", route.Spec.Path},
		{"Route Certificate", crt, route.Spec.TLS.Certificate},
		{"Route CACertificate", ca, route.Spec.TLS.CACertificate},
		{"Route Key", key, route.Spec.TLS.Key},
		{"Route DestinationCertificate", "", route.Spec.TLS.DestinationCACertificate},
		{"Route InsecureEdgeTerminationPolicy", helper2, route.Spec.TLS.InsecureEdgeTerminationPolicy},
	}
	verifyTests(testCR, t)

	helper = routev1.TLSTerminationReencrypt
	runtime.Spec.Route = &appstacksv1beta2.RuntimeComponentRoute{Termination: &helper, InsecureEdgeTerminationPolicy: &helper2}

	CustomizeRoute(route, runtime, key, crt, ca, destCACert)

	//TestReencrypt
	testCR = []Test{
		{"Route Certificate", crt, route.Spec.TLS.Certificate},
		{"Route CACertificate", ca, route.Spec.TLS.CACertificate},
		{"Route Key", key, route.Spec.TLS.Key},
		{"Route DestinationCertificate", destCACert, route.Spec.TLS.DestinationCACertificate},
		{"Route InsecureEdgeTerminationPolicy", helper2, route.Spec.TLS.InsecureEdgeTerminationPolicy},
		{"Route Target Port", "8443-tcp", route.Spec.Port.TargetPort.StrVal},
	}
	verifyTests(testCR, t)

	helper = routev1.TLSTerminationPassthrough
	runtime.Spec.Route = &appstacksv1beta2.RuntimeComponentRoute{Termination: &helper}
	runtime.Spec.Service.PortName = svcPortName

	CustomizeRoute(route, runtime, key, crt, ca, destCACert)

	//TestPassthrough
	testCR = []Test{
		{"Route Certificate", "", route.Spec.TLS.Certificate},
		{"Route CACertificate", "", route.Spec.TLS.CACertificate},
		{"Route Key", "", route.Spec.TLS.Key},
		{"Route DestinationCertificate", "", route.Spec.TLS.DestinationCACertificate},
		{"Route Target Port", svcPortName, route.Spec.Port.TargetPort.StrVal},
	}
	verifyTests(testCR, t)
}

func TestErrorIsNoMatchesForKind(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	newError := errors.New("test error")
	errorValue := ErrorIsNoMatchesForKind(newError, "kind", "version")

	testCR := []Test{
		{"Error", false, errorValue},
	}
	verifyTests(testCR, t)
}

func TestCustomizeService(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}
	svc, runtime := &corev1.Service{}, createRuntimeComponent(objMeta, spec)

	CustomizeService(svc, runtime)
	testCS := []Test{
		{"Service number of exposed ports", 1, len(svc.Spec.Ports)},
		{"Service first exposed port", runtime.Spec.Service.Port, svc.Spec.Ports[0].Port},
		{"Service first exposed target port", intstr.FromInt(int(runtime.Spec.Service.Port)), svc.Spec.Ports[0].TargetPort},
		{"Service type", *runtime.Spec.Service.Type, svc.Spec.Type},
		{"Service selector", name, svc.Spec.Selector["app.kubernetes.io/instance"]},
	}
	verifyTests(testCS, t)

	// Verify behaviour of optional target port functionality
	verifyTests(optionalTargetPortFunctionalityTests(), t)

	// verify optional nodePort functionality in NodePort service
	verifyTests(optionalNodePortFunctionalityTests(), t)

	additionalPortsTests(t)
}

func additionalPortsTests(t *testing.T) {
	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}
	svc, runtime := &corev1.Service{}, createRuntimeComponent(objMeta, spec)
	runtime.Spec.Service.Ports = ports

	CustomizeService(svc, runtime)

	testCS := []Test{
		{"Service number of exposed ports", 3, len(svc.Spec.Ports)},
		{"Second exposed port", ports[0].Port, svc.Spec.Ports[1].Port},
		{"Second exposed target port", targetHelper, svc.Spec.Ports[1].TargetPort.IntVal},
		{"Second exposed port name", ports[0].Name, svc.Spec.Ports[1].Name},
		{"Second nodeport", ports[0].NodePort, svc.Spec.Ports[1].NodePort},
		{"Third exposed port", ports[1].Port, svc.Spec.Ports[2].Port},
		{"Third exposed port name", fmt.Sprint(ports[1].Port) + "-tcp", svc.Spec.Ports[2].Name},
		{"Third nodeport", ports[1].NodePort, svc.Spec.Ports[2].NodePort},
	}
	verifyTests(testCS, t)

	runtime.Spec.Service.Ports = runtime.Spec.Service.Ports[:len(runtime.Spec.Service.Ports)-1]
	runtime.Spec.Service.Ports[0].NodePort = 3000
	runtime.Spec.Service.Type = &serviceType2
	CustomizeService(svc, runtime)

	testCS = []Test{
		{"Service number of exposed ports", 2, len(svc.Spec.Ports)},
		{"First nodeport", 3000, svc.Spec.Ports[0].NodePort},
		{"Port type", serviceType2, svc.Spec.Type},
	}

	runtime.Spec.Service.Ports = nil
	CustomizeService(svc, runtime)

	testCS = []Test{
		{"Service number of exposed ports", 1, len(svc.Spec.Ports)},
	}
	verifyTests(testCS, t)
}

func optionalTargetPortFunctionalityTests() []Test {
	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}
	spec.Service.TargetPort = &targetPort
	svc, runtime := &corev1.Service{}, createRuntimeComponent(objMeta, spec)

	CustomizeService(svc, runtime)
	testCS := []Test{
		{"Service number of exposed ports", 1, len(svc.Spec.Ports)},
		{"Service first exposed port", runtime.Spec.Service.Port, svc.Spec.Ports[0].Port},
		{"Service first exposed target port", intstr.FromInt(int(*runtime.Spec.Service.TargetPort)), svc.Spec.Ports[0].TargetPort},
		{"Service type", *runtime.Spec.Service.Type, svc.Spec.Type},
		{"Service selector", name, svc.Spec.Selector["app.kubernetes.io/instance"]},
	}
	return testCS
}

func optionalNodePortFunctionalityTests() []Test {
	serviceType := corev1.ServiceTypeNodePort
	service := &appstacksv1beta2.RuntimeComponentService{Type: &serviceType, Port: 8443, NodePort: &nodePort}
	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}
	svc, runtime := &corev1.Service{}, createRuntimeComponent(objMeta, spec)

	CustomizeService(svc, runtime)
	testCS := []Test{
		{"Service number of exposed ports", 1, len(svc.Spec.Ports)},
		{"Sercice first exposed port", runtime.Spec.Service.Port, svc.Spec.Ports[0].Port},
		{"Service first exposed target port", intstr.FromInt(int(runtime.Spec.Service.Port)), svc.Spec.Ports[0].TargetPort},
		{"Service type", *runtime.Spec.Service.Type, svc.Spec.Type},
		{"Service selector", name, svc.Spec.Selector["app.kubernetes.io/instance"]},
		{"Service nodePort port", *runtime.Spec.Service.NodePort, svc.Spec.Ports[0].NodePort},
	}
	return testCS
}

// Partial test for unittest TestCustomizeAffinity below
func partialTestCustomizeNodeAffinity(t *testing.T) {
	// required during scheduling ignored during execution
	rDSIDE := corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "key",
						Operator: corev1.NodeSelectorOpNotIn,
						Values:   []string{"large", "medium"},
					},
				},
			},
		},
	}
	// preferred during scheduling ignored during execution
	pDSIDE := []corev1.PreferredSchedulingTerm{
		{
			Weight: int32(20),
			Preference: corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "failure-domain.beta.kubernetes.io/zone",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"zoneB"},
					},
				},
			},
		},
	}
	labels := map[string]string{
		"customNodeLabel": "label1, label2",
	}
	affinityConfig := appstacksv1beta2.RuntimeComponentAffinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  &rDSIDE,
			PreferredDuringSchedulingIgnoredDuringExecution: pDSIDE,
		},
		NodeAffinityLabels: labels,
	}
	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Affinity:         &affinityConfig,
	}
	affinity, runtime := &corev1.Affinity{}, createRuntimeComponent(objMeta, spec)
	CustomizeAffinity(affinity, runtime)

	expectedMatchExpressions := []corev1.NodeSelectorRequirement{
		rDSIDE.NodeSelectorTerms[0].MatchExpressions[0],
		{
			Key:      "customNodeLabel",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{"label1", "label2"},
		},
	}
	testCNA := []Test{
		{"Node Affinity - Required Match Expressions", expectedMatchExpressions,
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions},
		{"Node Affinity - Prefered Match Expressions",
			pDSIDE[0].Preference.MatchExpressions,
			affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions},
	}
	verifyTests(testCNA, t)
}

// Partial test for unittest TestCustomizeAffinity bewlow
func partialTestCustomizePodAffinity(t *testing.T) {
	selectorA := makeInLabelSelector("service", []string{"Service-A"})
	selectorB := makeInLabelSelector("service", []string{"Service-B"})
	// required during scheduling ignored during execution
	rDSIDE := []corev1.PodAffinityTerm{
		{LabelSelector: &selectorA, TopologyKey: "failure-domain.beta.kubernetes.io/zone"},
	}
	// preferred during scheduling ignored during execution
	pDSIDE := []corev1.WeightedPodAffinityTerm{
		{
			Weight: int32(20),
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &selectorB, TopologyKey: "kubernetes.io/hostname",
			},
		},
	}
	affinityConfig := appstacksv1beta2.RuntimeComponentAffinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: rDSIDE,
		},
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: pDSIDE,
		},
	}
	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Affinity:         &affinityConfig,
	}
	affinity, runtime := &corev1.Affinity{}, createRuntimeComponent(objMeta, spec)
	CustomizeAffinity(affinity, runtime)

	testCPA := []Test{
		{"Pod Affinity - Required Affinity Term", rDSIDE,
			affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution},
		{"Pod AntiAffinity - Preferred Affinity Term", pDSIDE,
			affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution},
	}
	verifyTests(testCPA, t)
}

func TestCustomizeAffinity(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	partialTestCustomizeNodeAffinity(t)
	partialTestCustomizePodAffinity(t)
}

func TestCustomizePodSpecAnnotations(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Service:          service,
		Resources:        resourceContraints,
		Probes:           probes,
		VolumeMounts:     []corev1.VolumeMount{volumeMount},
		PullPolicy:       &pullPolicy,
		Env:              env,
		EnvFrom:          envFrom,
		Volumes:          []corev1.Volume{volume},
	}

	// No dep or set, annotation should be empty
	pts1, runtime1 := &corev1.PodTemplateSpec{}, createRuntimeComponent(objMeta, spec)
	CustomizePodSpec(pts1, runtime1)
	annolen1 := len(pts1.Annotations)
	testAnnotations1 := []Test{
		{"Shouldn't be any annotations", 0, annolen1},
	}
	verifyTests(testAnnotations1, t)

	// dep but not set, annotation should be dep annotations
	deployment := &appstacksv1beta2.RuntimeComponentDeployment{Annotations: deploymentAnnos}
	spec.Deployment = deployment
	pts2, runtime2 := &corev1.PodTemplateSpec{}, createRuntimeComponent(objMeta, spec)
	CustomizePodSpec(pts2, runtime2)
	annolen2 := len(pts2.Annotations)
	anno2 := pts2.Annotations["depAnno"]
	testAnnotations2 := []Test{
		{"Wrong annotations", "depAnno", anno2},
		{"Wrong number of annotations", 1, annolen2},
	}
	verifyTests(testAnnotations2, t)

	// set but not dep, annotation should be set annotations
	spec.Deployment = nil
	statefulSet := &appstacksv1beta2.RuntimeComponentStatefulSet{Annotations: ssAnnos}
	spec.StatefulSet = statefulSet
	pts3, runtime3 := &corev1.PodTemplateSpec{}, createRuntimeComponent(objMeta, spec)
	CustomizePodSpec(pts3, runtime3)
	annolen3 := len(pts3.Annotations)
	anno3 := pts3.Annotations["setAnno"]
	testAnnotations3 := []Test{
		{"Wrong annotations", "setAnno", anno3},
		{"Wrong number of annotations", 1, annolen3},
	}
	verifyTests(testAnnotations3, t)

	// dep and set, annotation should be set annotations
	spec.Deployment = deployment
	pts4, runtime4 := &corev1.PodTemplateSpec{}, createRuntimeComponent(objMeta, spec)
	CustomizePodSpec(pts4, runtime4)
	annolen4 := len(pts4.Annotations)
	anno4 := pts4.Annotations["setAnno"]
	testAnnotations4 := []Test{
		{"Wrong annotations", "setAnno", anno4},
		{"Wrong number of annotations", 1, annolen4},
	}
	verifyTests(testAnnotations4, t)
}

func TestCustomizePodSpec(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Service:          service,
		Resources:        resourceContraints,
		Probes:           probes,
		VolumeMounts:     []corev1.VolumeMount{volumeMount},
		PullPolicy:       &pullPolicy,
		Env:              env,
		EnvFrom:          envFrom,
		Volumes:          []corev1.Volume{volume},
	}
	pts, runtime := &corev1.PodTemplateSpec{}, createRuntimeComponent(objMeta, spec)
	// else cond
	CustomizePodSpec(pts, runtime)
	noCont := len(pts.Spec.Containers)
	noPorts := len(pts.Spec.Containers[0].Ports)
	ptsSAN := pts.Spec.ServiceAccountName
	affinityConfig := appstacksv1beta2.RuntimeComponentAffinity{
		Architecture: arch,
	}
	// if cond
	spec = appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Service: &appstacksv1beta2.RuntimeComponentService{
			Type:       &serviceType,
			Port:       8443,
			TargetPort: &targetPort,
		},
		Resources:          resourceContraints,
		Probes:             probes,
		VolumeMounts:       []corev1.VolumeMount{volumeMount},
		PullPolicy:         &pullPolicy,
		Env:                env,
		EnvFrom:            envFrom,
		Volumes:            []corev1.Volume{volume},
		ServiceAccountName: &serviceAccountName,
		Affinity:           &affinityConfig,
	}
	runtime = createRuntimeComponent(objMeta, spec)
	CustomizePodSpec(pts, runtime)
	ptsCSAN := pts.Spec.ServiceAccountName

	// affinity tests
	affArchs := pts.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
	weight := pts.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight
	prefAffArchs := pts.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions[0].Values[0]
	assignedTPort := pts.Spec.Containers[0].Ports[0].ContainerPort

	testCPS := []Test{
		{"No containers", 1, noCont},
		{"No port", 1, noPorts},
		{"No ServiceAccountName", name, ptsSAN},
		{"ServiceAccountName available", serviceAccountName, ptsCSAN},
	}
	verifyTests(testCPS, t)

	testCA := []Test{
		{"Archs", arch[0], affArchs},
		{"Weight", int32(1), int32(weight)},
		{"Archs", arch[0], prefAffArchs},
		{"No target port", targetPort, assignedTPort},
	}
	verifyTests(testCA, t)
}

func TestCustomizePersistence(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	runtimeStatefulSet := &appstacksv1beta2.RuntimeComponentStatefulSet{Storage: &storage}
	spec := appstacksv1beta2.RuntimeComponentSpec{StatefulSet: runtimeStatefulSet}
	statefulSet, runtime := &appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)
	statefulSet.Spec.Template.Spec.Containers = []corev1.Container{{}}
	statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{}
	// if vct == 0, runtimeVCT != nil, not found
	CustomizePersistence(statefulSet, runtime)
	ssK := statefulSet.Spec.VolumeClaimTemplates[0].TypeMeta.Kind
	ssMountPath := statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath

	//reset
	storageNilVCT := appstacksv1beta2.RuntimeComponentStorage{Size: "10Mi", MountPath: "/mnt/data", VolumeClaimTemplate: nil}
	runtimeStatefulSet = &appstacksv1beta2.RuntimeComponentStatefulSet{Storage: &storageNilVCT}
	spec = appstacksv1beta2.RuntimeComponentSpec{StatefulSet: runtimeStatefulSet}
	statefulSet, runtime = &appsv1.StatefulSet{}, createRuntimeComponent(objMeta, spec)

	statefulSet.Spec.Template.Spec.Containers = []corev1.Container{{}}
	statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = append(statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
	//runtimeVCT == nil, found
	CustomizePersistence(statefulSet, runtime)
	ssVolumeMountName := statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name
	size := statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]
	testCP := []Test{
		{"Persistence kind with VCT", volumeCT.TypeMeta.Kind, ssK},
		{"PVC size", storage.Size, size.String()},
		{"Mount path", storage.MountPath, ssMountPath},
		{"Volume Mount Name", volumeCT.Name, ssVolumeMountName},
	}
	verifyTests(testCP, t)
}

func TestCustomizeServiceAccount(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{PullSecret: &pullSecret}
	sa, runtime := &corev1.ServiceAccount{}, createRuntimeComponent(objMeta, spec)
	CustomizeServiceAccount(sa, runtime)
	emptySAIPS := sa.ImagePullSecrets[0].Name

	newSecret := "my-new-secret"
	spec = appstacksv1beta2.RuntimeComponentSpec{PullSecret: &newSecret}
	runtime = createRuntimeComponent(objMeta, spec)
	CustomizeServiceAccount(sa, runtime)

	testCSA := []Test{
		{"ServiceAccount image pull secrets is empty", pullSecret, emptySAIPS},
		{"ServiceAccount image pull secrets", newSecret, sa.ImagePullSecrets[0].Name},
	}
	verifyTests(testCSA, t)
}

func TestCustomizeKnativeService(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage: appImage,
		Service:          service,
		Probes:           probes,
		PullPolicy:       &pullPolicy,
		Env:              env,
		EnvFrom:          envFrom,
		Volumes:          []corev1.Volume{volume},
	}
	ksvc, runtime := &servingv1.Service{}, createRuntimeComponent(objMeta, spec)

	CustomizeKnativeService(ksvc, runtime)
	ksvcNumPorts := len(ksvc.Spec.Template.Spec.Containers[0].Ports)
	ksvcSAN := ksvc.Spec.Template.Spec.ServiceAccountName

	ksvcLPPort := ksvc.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port
	ksvcLPTCP := ksvc.Spec.Template.Spec.Containers[0].LivenessProbe.TCPSocket.Port
	ksvcRPPort := ksvc.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port
	ksvcRPTCP := ksvc.Spec.Template.Spec.Containers[0].ReadinessProbe.TCPSocket.Port
	ksvcSPPort := ksvc.Spec.Template.Spec.Containers[0].StartupProbe.HTTPGet.Port
	ksvcSPTCP := ksvc.Spec.Template.Spec.Containers[0].StartupProbe.TCPSocket.Port
	ksvcLabelNoExpose := ksvc.Labels["serving.knative.dev/visibility"]

	spec = appstacksv1beta2.RuntimeComponentSpec{
		ApplicationImage:   appImage,
		Service:            service,
		PullPolicy:         &pullPolicy,
		Env:                env,
		EnvFrom:            envFrom,
		Volumes:            []corev1.Volume{volume},
		ServiceAccountName: &serviceAccountName,
		Probes:             probes,
		Expose:             &expose,
	}
	runtime = createRuntimeComponent(objMeta, spec)
	CustomizeKnativeService(ksvc, runtime)
	ksvcLabelTrueExpose := ksvc.Labels["serving.knative.dev/visibility"]

	fls := false
	runtime.Spec.Expose = &fls
	CustomizeKnativeService(ksvc, runtime)
	ksvcLabelFalseExpose := ksvc.Labels["serving.knative.dev/visibility"]

	testCKS := []Test{
		{"ksvc container ports", 1, ksvcNumPorts},
		{"ksvc ServiceAccountName is nil", name, ksvcSAN},
		{"ksvc ServiceAccountName not nil", *runtime.Spec.ServiceAccountName, ksvc.Spec.Template.Spec.ServiceAccountName},
		{"liveness probe port", intstr.IntOrString{}, ksvcLPPort},
		{"liveness probe TCP socket port", intstr.IntOrString{}, ksvcLPTCP},
		{"Readiness probe port", intstr.IntOrString{}, ksvcRPPort},
		{"Readiness probe TCP socket port", intstr.IntOrString{}, ksvcRPTCP},
		{"Startup probe port", intstr.IntOrString{}, ksvcSPPort},
		{"Startup probe TCP socket port", intstr.IntOrString{}, ksvcSPTCP},
		{"expose not set", "cluster-local", ksvcLabelNoExpose},
		{"expose set to true", "", ksvcLabelTrueExpose},
		{"expose set to false", "cluster-local", ksvcLabelFalseExpose},
	}
	verifyTests(testCKS, t)
}

func TestCustomizeHPA(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{Autoscaling: autoscaling}
	hpa, runtime := &autoscalingv1.HorizontalPodAutoscaler{}, createRuntimeComponent(objMeta, spec)
	CustomizeHPA(hpa, runtime)
	nilSTRKind := hpa.Spec.ScaleTargetRef.Kind

	runtimeStatefulSet := &appstacksv1beta2.RuntimeComponentStatefulSet{Storage: &storage}
	spec = appstacksv1beta2.RuntimeComponentSpec{Autoscaling: autoscaling, StatefulSet: runtimeStatefulSet}
	runtime = createRuntimeComponent(objMeta, spec)
	CustomizeHPA(hpa, runtime)
	STRKind := hpa.Spec.ScaleTargetRef.Kind

	testCHPA := []Test{
		{"Max replicas", autoscaling.MaxReplicas, hpa.Spec.MaxReplicas},
		{"Min replicas", *autoscaling.MinReplicas, *hpa.Spec.MinReplicas},
		{"Target CPU utilization", *autoscaling.TargetCPUUtilizationPercentage, *hpa.Spec.TargetCPUUtilizationPercentage},
		{"", name, hpa.Spec.ScaleTargetRef.Name},
		{"", "apps/v1", hpa.Spec.ScaleTargetRef.APIVersion},
		{"Storage not enabled", "Deployment", nilSTRKind},
		{"Storage enabled", "StatefulSet", STRKind},
	}
	verifyTests(testCHPA, t)
}

func TestValidate(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	storage2 := &appstacksv1beta2.RuntimeComponentStorage{}
	runtimeStatefulSet := &appstacksv1beta2.RuntimeComponentStatefulSet{Storage: storage2}
	spec = appstacksv1beta2.RuntimeComponentSpec{Service: service, StatefulSet: runtimeStatefulSet}

	runtime := createRuntimeComponent(objMeta, spec)

	result, err := Validate(runtime)

	testVal := []Test{
		{"Error response", false, result},
		{"Error response", errors.New("validation failed: must set the field(s): spec.statefulSet.storage.size"), err},
	}
	verifyTests(testVal, t)

	storage2 = &appstacksv1beta2.RuntimeComponentStorage{Size: "size"}
	runtime.Spec.StatefulSet.Storage = storage2

	result, err = Validate(runtime)

	testVal = []Test{
		{"Error response", false, result},
		{"Error response", errors.New("validation failed: cannot parse 'size': quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"), err},
	}
	verifyTests(testVal, t)

	runtime.Spec.StatefulSet.Storage = &storage

	result, err = Validate(runtime)

	testVal = []Test{
		{"Result", true, result},
		{"Error response", nil, err},
	}
	verifyTests(testVal, t)
}

func TestCreateValidationError(t *testing.T) {
	result := createValidationError("Test Error")

	testVE := []Test{
		{"Validation error message", errors.New("validation failed: Test Error"), result},
	}
	verifyTests(testVE, t)
}

func TestRequiredFieldMessage(t *testing.T) {
	result := requiredFieldMessage("Required")

	testRFM := []Test{
		{"Required Field Message", "must set the field(s): Required", result},
	}
	verifyTests(testRFM, t)
}

func TestCustomizeServiceMonitor(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}

	params := map[string][]string{
		"params": {"param1", "param2"},
	}

	portValue := intstr.FromString("web")
	// Endpoint for runtime
	endpointApp := &prometheusv1.Endpoint{
		Port:            "web",
		Scheme:          "myScheme",
		Interval:        "myInterval",
		Path:            "myPath",
		TLSConfig:       &prometheusv1.TLSConfig{},
		BasicAuth:       &prometheusv1.BasicAuth{},
		Params:          params,
		ScrapeTimeout:   "myScrapeTimeout",
		BearerTokenFile: "myBearerTokenFile",
		TargetPort:      &portValue,
	}
	endpointsApp := make([]prometheusv1.Endpoint, 1)
	endpointsApp[0] = *endpointApp

	// Endpoint for sm
	endpointsSM := make([]prometheusv1.Endpoint, 0)

	labelMap := map[string]string{"app": "my-app"}
	selector := &metav1.LabelSelector{MatchLabels: labelMap}
	smspec := &prometheusv1.ServiceMonitorSpec{Endpoints: endpointsSM, Selector: *selector}

	sm, runtime := &prometheusv1.ServiceMonitor{Spec: *smspec}, createRuntimeComponent(objMeta, spec)
	runtime.Spec.Monitoring = &appstacksv1beta2.RuntimeComponentMonitoring{Labels: labelMap, Endpoints: endpointsApp}

	CustomizeServiceMonitor(sm, runtime)

	labelMatches := map[string]string{
		"monitor.rc.app.stacks/enabled": "true",
		"app.kubernetes.io/instance":    name,
	}

	allSMLabels := runtime.GetLabels()
	for key, value := range runtime.Spec.Monitoring.Labels {
		allSMLabels[key] = value
	}

	// Expected values
	appPort := runtime.Spec.Monitoring.Endpoints[0].Port
	appScheme := runtime.Spec.Monitoring.Endpoints[0].Scheme
	appInterval := runtime.Spec.Monitoring.Endpoints[0].Interval
	appPath := runtime.Spec.Monitoring.Endpoints[0].Path
	appTLSConfig := runtime.Spec.Monitoring.Endpoints[0].TLSConfig
	appBasicAuth := runtime.Spec.Monitoring.Endpoints[0].BasicAuth
	appParams := runtime.Spec.Monitoring.Endpoints[0].Params
	appScrapeTimeout := runtime.Spec.Monitoring.Endpoints[0].ScrapeTimeout
	appBearerTokenFile := runtime.Spec.Monitoring.Endpoints[0].BearerTokenFile

	testSM := []Test{
		{"Service Monitor label for app.kubernetes.io/instance", name, sm.Labels["app.kubernetes.io/instance"]},
		{"Service Monitor selector match labels", labelMatches, sm.Spec.Selector.MatchLabels},
		{"Service Monitor endpoints port", appPort, sm.Spec.Endpoints[0].Port},
		{"Service Monitor all labels", allSMLabels, sm.Labels},
		{"Service Monitor endpoints scheme", appScheme, sm.Spec.Endpoints[0].Scheme},
		{"Service Monitor endpoints interval", appInterval, sm.Spec.Endpoints[0].Interval},
		{"Service Monitor endpoints path", appPath, sm.Spec.Endpoints[0].Path},
		{"Service Monitor endpoints TLSConfig", appTLSConfig, sm.Spec.Endpoints[0].TLSConfig},
		{"Service Monitor endpoints basicAuth", appBasicAuth, sm.Spec.Endpoints[0].BasicAuth},
		{"Service Monitor endpoints params", appParams, sm.Spec.Endpoints[0].Params},
		{"Service Monitor endpoints scrapeTimeout", appScrapeTimeout, sm.Spec.Endpoints[0].ScrapeTimeout},
		{"Service Monitor endpoints bearerTokenFile", appBearerTokenFile, sm.Spec.Endpoints[0].BearerTokenFile},
	}
	verifyTests(testSM, t)
}
func TestGetCondition(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	status := &appstacksv1beta2.RuntimeComponentStatus{
		Conditions: []appstacksv1beta2.StatusCondition{
			{
				Status: corev1.ConditionTrue,
				Type:   appstacksv1beta2.StatusConditionTypeReconciled,
			},
		},
	}
	conditionType := appstacksv1beta2.StatusConditionTypeReconciled
	cond := GetCondition(conditionType, status)
	testGC := []Test{{"Set status condition", status.Conditions[0].Status, cond.Status}}
	verifyTests(testGC, t)

	status = &appstacksv1beta2.RuntimeComponentStatus{}
	cond = GetCondition(conditionType, status)
	testGC = []Test{{"Set status condition", 0, len(status.Conditions)}}
	verifyTests(testGC, t)
}

func TestSetCondition(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	status := &appstacksv1beta2.RuntimeComponentStatus{
		Conditions: []appstacksv1beta2.StatusCondition{
			{Type: appstacksv1beta2.StatusConditionTypeReconciled},
		},
	}
	condition := appstacksv1beta2.StatusCondition{
		Status: corev1.ConditionTrue,
		Type:   appstacksv1beta2.StatusConditionTypeReconciled,
	}
	SetCondition(condition, status)
	testSC := []Test{{"Set status condition", condition.Status, status.Conditions[0].Status}}
	verifyTests(testSC, t)
}

func TestGetWatchNamespaces(t *testing.T) {
	// Set the logger to development mode for verbose logs
	logger := zap.New()
	logf.SetLogger(logger)

	os.Setenv("WATCH_NAMESPACE", "")
	namespaces, err := GetWatchNamespaces()
	configMapConstTests := []Test{
		{"namespaces", []string{""}, namespaces},
		{"error", nil, err},
	}
	verifyTests(configMapConstTests, t)

	os.Setenv("WATCH_NAMESPACE", "ns1")
	namespaces, err = GetWatchNamespaces()
	configMapConstTests = []Test{
		{"namespaces", []string{"ns1"}, namespaces},
		{"error", nil, err},
	}
	verifyTests(configMapConstTests, t)

	os.Setenv("WATCH_NAMESPACE", "ns1,ns2,ns3")
	namespaces, err = GetWatchNamespaces()
	configMapConstTests = []Test{
		{"namespaces", []string{"ns1", "ns2", "ns3"}, namespaces},
		{"error", nil, err},
	}
	verifyTests(configMapConstTests, t)

	os.Setenv("WATCH_NAMESPACE", " ns1   ,  ns2,  ns3  ")
	namespaces, err = GetWatchNamespaces()
	configMapConstTests = []Test{
		{"namespaces", []string{"ns1", "ns2", "ns3"}, namespaces},
		{"error", nil, err},
	}
	verifyTests(configMapConstTests, t)
}

func TestGetOperatorNamespace(t *testing.T) {
	// Set the logger to development mode for verbose logs
	logger := zap.New()
	logf.SetLogger(logger)

	os.Setenv("POD_NAMESPACE", "")
	namespace, err := GetOperatorNamespace()
	configMapConstTests := []Test{
		{"namespaces", "", namespace},
		{"error", nil, err},
	}
	verifyTests(configMapConstTests, t)
}

func TestGetOpenShiftAnnotations(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service}
	runtime := createRuntimeComponent(objMeta, spec)

	annos := map[string]string{
		"image.opencontainers.org/source": "source",
	}
	runtime.Annotations = annos

	result := GetOpenShiftAnnotations(runtime)

	annos = map[string]string{
		"app.openshift.io/vcs-uri": "source",
	}
	testOSA := []Test{
		{"OpenShiftAnnotations", annos["app.openshift.io/vcs-uri"], result["app.openshift.io/vcs-uri"]},
	}
	verifyTests(testOSA, t)
}

func TestIsClusterWide(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	namespaces := []string{"namespace"}
	result := IsClusterWide(namespaces)

	testCW := []Test{
		{"One namespace", false, result},
	}
	verifyTests(testCW, t)

	namespaces = []string{""}
	result = IsClusterWide(namespaces)

	testCW = []Test{
		{"All namespaces", true, result},
	}
	verifyTests(testCW, t)

	namespaces = []string{"namespace1", "namespace2"}
	result = IsClusterWide(namespaces)

	testCW = []Test{
		{"Two namespaces", false, result},
	}
	verifyTests(testCW, t)
}

func TestCustomizeIngress(t *testing.T) {
	logger := zap.New()
	logf.SetLogger(logger)

	ing := networkingv1.Ingress{}
	var pathType networkingv1.PathType = networkingv1.PathType("Prefix")
	route := appstacksv1beta2.RuntimeComponentRoute{Host: "routeHost", Path: "myPath", PathType: pathType, Annotations: annotations}
	spec := appstacksv1beta2.RuntimeComponentSpec{Service: service, Route: &route}
	runtime := createRuntimeComponent(objMeta, spec)

	CustomizeIngress(&ing, runtime)

	testIng := []Test{
		{"Ingress Labels", labels["key1"], ing.Labels["key1"]},
		{"Ingress Annotations", annotations, ing.Annotations},
		{"Ingress Route Host", "routeHost", ing.Spec.Rules[0].Host},
		{"Ingress Route Path", "myPath", ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path},
		{"Ingress Route PathType", pathType, *ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].PathType},
		{"Ingress Route ServiceName", name, ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name},
		{"Ingress Route ServicePort", "myservice", ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Name},
		{"Ingress TLS", 0, len(ing.Spec.TLS)},
	}
	verifyTests(testIng, t)

	certSecretRef := "my-ref"
	route = appstacksv1beta2.RuntimeComponentRoute{Host: "routeHost", Path: "myPath", CertificateSecretRef: &certSecretRef}

	CustomizeIngress(&ing, runtime)

	testIng = []Test{
		{"Ingress TLS SecretName", certSecretRef, ing.Spec.TLS[0].SecretName},
	}
	verifyTests(testIng, t)
}

// Helper Functions
// Unconditionally set the proper tags for an enabled runtime omponent
func createAppDefinitionTags(app *appstacksv1beta2.RuntimeComponent) (map[string]string, map[string]string) {
	// The purpose of this function demands all fields configured
	if app.Spec.ApplicationVersion == "" {
		app.Spec.ApplicationVersion = "v1alpha"
	}
	// set fields
	label := map[string]string{
		"kappnav.app.auto-create": "true",
	}
	annotations := map[string]string{
		"kappnav.app.auto-create.name":          app.Spec.ApplicationName,
		"kappnav.app.auto-create.kinds":         "Deployment, StatefulSet, Service, Route, Ingress, ConfigMap",
		"kappnav.app.auto-create.label":         "app.kubernetes.io/part-of",
		"kappnav.app.auto-create.labels-values": app.Spec.ApplicationName,
		"kappnav.app.auto-create.version":       app.Spec.ApplicationVersion,
	}
	return label, annotations
}
func createRuntimeComponent(obj metav1.ObjectMeta, spec appstacksv1beta2.RuntimeComponentSpec) *appstacksv1beta2.RuntimeComponent {
	app := &appstacksv1beta2.RuntimeComponent{
		ObjectMeta: obj,
		Spec:       spec,
	}
	return app
}

// Used in TestCustomizeAffinity to make an IN selector with paramenters key and values.
func makeInLabelSelector(key string, values []string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      key,
				Operator: metav1.LabelSelectorOpIn,
				Values:   values,
			},
		},
	}
}

func verifyTests(tests []Test, t *testing.T) {
	for _, tt := range tests {
		if !reflect.DeepEqual(tt.actual, tt.expected) {
			t.Errorf("%s test expected: (%v) actual: (%v)", tt.test, tt.expected, tt.actual)
		}
	}
}
